package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/itering/substrate-api-rpc/rpc"
	"github.com/itering/substrate-api-rpc/websocket"
	ws "github.com/itering/substrate-api-rpc/websocket"
	"github.com/panjf2000/ants"
	"github.com/prometheus/common/log"
	"github.com/shopspring/decimal"
)

const pSize = 100 // process rows per time
const retry = 1000

type repairService struct {
	done            chan struct{}
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
	CommonService   model.CommonService
	RuntimeService  model.RuntimeService
	BlockService    model.BlockService
	PluginService   model.PluginService
	DbStorage       *DbStorage
}

type RepairConfig struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
	DbStorage       *DbStorage
}

func NewRepairService(c *RepairConfig, done chan struct{}, cs model.CommonService, r model.RuntimeService, b model.BlockService, p model.PluginService) model.RepairService {
	return &repairService{
		done:            done,
		SqlRepository:   c.SqlRepository,
		RedisRepository: c.RedisRepository,
		CommonService:   cs,
		RuntimeService:  r,
		BlockService:    b,
		PluginService:   p,
	}
}

func (s *repairService) initRepairService(done chan struct{}) *repairService {
	return &repairService{
		done:            done,
		SqlRepository:   s.SqlRepository,
		RedisRepository: s.RedisRepository,
		CommonService:   s.CommonService,
		RuntimeService:  s.RuntimeService,
		BlockService:    s.BlockService,
		PluginService:   s.PluginService,
	}
}

func (s *repairService) processUnit(rs *repairService, head, size int) (err error) {
	log.Info("block head: ", head)
	bs := s.BlockService.GetMissingBlockMap(head, 0, size)
	// bs, err := s.BlockService.InitialMissingBlockSet(head, 0, size)

	// var bs []string
	// TODO getMissing Blocks from redis set record

	log.Info("bs: ", bs)
	j := 0
	for j < retry {
		rs.RepairBlocks(&bs)
		c := rs.filter(bs)
		var count = 0
		c.Range(func(key int, value bool) bool {
			// log.Info("key: ", key, " value: ", value)
			count++
			return true
		})
		if count == 0 {
			break
		}
		j++
	}

	// k := 0
	// for k < retry {
	// 	if bs, err = s.BlockService.GetMissingBlockSet(head, 0, size); err != nil {
	// 		return err
	// 	}
	// 	rs.RepairBlocksBySet(bs)
	// 	if bs, err = s.BlockService.GetMissingBlockSet(head, 0, size); err != nil {
	// 		return err
	// 	}
	// 	if len(bs) == 0 {
	// 		break
	// 	}
	// 	k++
	// }

	l := 0
	for l < retry {
		rs.RepairPlugins(&bs)
		c := rs.filter(bs)
		var count = 0
		c.Range(func(key int, value bool) bool {
			log.Info("key: ", key, " value: ", value)
			count++
			return true
		})
		if count == 0 {
			break
		}
		l++
	}
	return nil
}

func (s *repairService) Repair(conn ws.WsConn, interrupt chan os.Signal, head, size int) {
	signal.Notify(interrupt, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	log.Info("head: ", head, " size: ", size)
	l := size / pSize

	done := make(chan struct{})
	repairSrv := s.initRepairService(done)
	go func() {
		defer close(done)
		for i := 0; i < l+1; i++ {
			s.processUnit(repairSrv, head-(i*pSize), pSize)
			// time.Sleep(60 * time.Second) // Gap interval 1 min
		}
		done <- struct{}{}
	}()

	for {
		select {
		case <-done:
			log.Info("finish repair")
			os.Exit(0)
		}
	}
}

func (s *repairService) filter(bs model.IntBoolMap) (vsf model.IntBoolMap) {
	// log.Info("--- filter repair group status ---")
	bs.Range(func(key int, value bool) bool {
		// log.Info("key: ", key, " value: ", value)
		if !value {
			vsf.Store(key, value)
		}
		return true
	})
	return
}

func (s *repairService) RepairBlocksBySet(bs []string) {
	log.Info("== RepairBlocksBySet ==")
	var wg sync.WaitGroup

	p, _ := ants.NewPoolWithFunc(16, func(i interface{}) {
		blockFinalized := i.(model.BlockFinalized)
		func(bf model.BlockFinalized) {
			if err := s.fillBlockDataBySet(nil, bf.BlockNum, bs); err != nil {
				log.Error("fillBlockData get error ", err)
			} else {
				s.CommonService.SetHeartBeat(fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, "substrate"))
			}
		}(blockFinalized)
		wg.Done()
	}, ants.WithOptions(ants.Options{PanicHandler: func(c interface{}) {}, PreAlloc: true}))

	defer p.Release()

	for _, block := range bs {
		wg.Add(1)
		b, _ := strconv.Atoi(block)
		_ = p.Invoke(model.BlockFinalized{BlockNum: b, Finalized: true})
	}

	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.Running())
}

func (s *repairService) RepairBlocks(bs *model.IntBoolMap) { // this method should be deprecated
	log.Info("== RepairBlocks ==")
	var wg sync.WaitGroup

	p, _ := ants.NewPoolWithFunc(16, func(i interface{}) {
		blockFinalized := i.(model.BlockFinalized)
		func(bf model.BlockFinalized) {
			if err := s.fillBlockData(nil, bf.BlockNum, bs); err != nil {
				log.Error("fillBlockData get error ", err)
			} else {
				s.CommonService.SetHeartBeat(fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, "substrate"))
			}
		}(blockFinalized)
		wg.Done()
	}, ants.WithOptions(ants.Options{PanicHandler: func(c interface{}) {}, PreAlloc: true}))

	defer p.Release()

	bs.Range(func(key int, value bool) bool {
		if value != true {
			wg.Add(1)
			_ = p.Invoke(model.BlockFinalized{BlockNum: key, Finalized: true})
			return true
		}
		return false
	})
	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.Running())
}

func (s *repairService) RepairPlugins(bs *model.IntBoolMap) {
	log.Info("== RepairPlugins ==")
	var wg sync.WaitGroup

	p, _ := ants.NewPoolWithFunc(8, func(i interface{}) {
		blockNum := i.(model.BlockFinalized)
		func(bf model.BlockFinalized) {
			if err := s.fillPluginData(bf.BlockNum, bs); err != nil {
				log.Error("fillPluginData get error ", err)
			} else {
				s.CommonService.SetHeartBeat(fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, "substrate"))
			}
		}(blockNum)
		wg.Done()
	}, ants.WithOptions(ants.Options{PanicHandler: func(c interface{}) {
	}}))

	defer p.Release()

	bs.Range(func(key int, value bool) bool {
		wg.Add(1)
		_ = p.Invoke(model.BlockFinalized{BlockNum: key, Finalized: true})
		return true
	})
	wg.Wait()
}

func (s *repairService) PluginRegister() {
	log.Info("--- PluginRegister ---")
	for name, plugin := range plugins.RegisteredPlugins {
		log.Info("name: ", name)
		db := s.DbStorage
		db.Prefix = name
		plugin.InitDao(db)
		for _, moduleId := range plugin.SubscribeExtrinsic() {
			subscribeExtrinsic[moduleId] = append(subscribeExtrinsic[moduleId], plugin)
		}
		for _, moduleId := range plugin.SubscribeEvent() {
			subscribeEvent[moduleId] = append(subscribeEvent[moduleId], plugin)
		}
	}
}

func (s *repairService) fillBlockDataBySet(conn websocket.WsConn, blockNum int, bs []string) (err error) {
	// block := s.SqlRepository.GetBlockByNum(blockNum)
	// if block != nil && block.Finalized && block.ExtrinsicsCount != 0 {
	// 	bs.Store(blockNum, true)
	// 	return nil
	// }

	v := &rpc.JsonRpcResult{}

	err = ws.SendWsRequest(conn, v, rpc.ChainGetBlockHash(wsBlockHash, blockNum))
	blockHash, err := v.ToString()
	if err != nil || blockHash == "" {
		return fmt.Errorf("ChainGetBlockHash get error %v", err)
	}

	err = ws.SendWsRequest(conn, v, rpc.ChainGetBlock(wsBlock, blockHash))
	if err != nil {
		return fmt.Errorf("ChainGetBlock get error %v", err)
	}
	rpcBlock := v.ToBlock()

	err = ws.SendWsRequest(conn, v, rpc.StateGetStorage(wsEvent, util.EventStorageKey, blockHash))
	if err != nil {
		return fmt.Errorf("StateGetStorage get error %v", err)
	}
	event, _ := v.ToString()

	err = ws.SendWsRequest(conn, v, rpc.ChainGetRuntimeVersion(wsSpec, blockHash))

	if err != nil {
		return fmt.Errorf("ChainGetRuntimeVersion get error %v", err)
	}

	var specVersion int

	if r := v.ToRuntimeVersion(); r == nil {
		specVersion = s.CommonService.GetCurrentRuntimeSpecVersion(blockNum)
	} else {
		specVersion = r.SpecVersion
		_ = s.RuntimeService.RegRuntimeVersion(r.ImplName, specVersion, blockHash)
	}

	if specVersion > util.CurrentRuntimeSpecVersion {
		util.CurrentRuntimeSpecVersion = specVersion
	}

	if rpcBlock == nil || specVersion == -1 {
		return errors.New("nil block data")
	}

	// // refresh finalized info for update
	// if block != nil {
	// 	// Confirm data, only set block Finalized, refresh all block data
	// 	block.ExtrinsicsRoot = rpcBlock.Block.Header.ExtrinsicsRoot
	// 	block.Hash = blockHash
	// 	block.ParentHash = rpcBlock.Block.Header.ParentHash
	// 	block.StateRoot = rpcBlock.Block.Header.StateRoot
	// 	block.Extrinsics = util.ToString(rpcBlock.Block.Extrinsics)
	// 	block.Logs = util.ToString(rpcBlock.Block.Header.Digest.Logs)
	// 	block.Event = event
	// 	block.CodecError = false
	// 	if err = s.BlockService.UpdateBlockData(conn, block, true); err == nil {
	// 		bs.Store(blockNum, true)
	// 	}
	// 	return
	// }

	// for Create
	if err = s.BlockService.CreateChainBlock(conn, blockHash, &rpcBlock.Block, event, specVersion, true); err == nil {
		s.RedisRepository.AddRepairedBlock(context.TODO(), blockNum)
	} else {
		log.Error("Create chain block error ", err)
		if strings.Contains(err.Error(), "Recovering from panic in DecodeExtrinsic") {
			s.RedisRepository.AddRepairedBlock(context.TODO(), blockNum)
			log.Error("Recovering from panic in DecodeExtrinsic at block: ", blockNum)
		}
	}
	return
}

func (s *repairService) fillBlockData(conn websocket.WsConn, blockNum int, bs *model.IntBoolMap) (err error) {
	block := s.SqlRepository.GetBlockByNum(blockNum)
	if block != nil && block.Finalized && block.ExtrinsicsCount != 0 {
		bs.Store(blockNum, true)
		return nil
	}

	v := &rpc.JsonRpcResult{}

	err = ws.SendWsRequest(conn, v, rpc.ChainGetBlockHash(wsBlockHash, blockNum))
	blockHash, err := v.ToString()
	if err != nil || blockHash == "" {
		return fmt.Errorf("ChainGetBlockHash get error %v", err)
	}

	err = ws.SendWsRequest(conn, v, rpc.ChainGetBlock(wsBlock, blockHash))
	if err != nil {
		return fmt.Errorf("ChainGetBlock get error %v", err)
	}
	rpcBlock := v.ToBlock()

	err = ws.SendWsRequest(conn, v, rpc.StateGetStorage(wsEvent, util.EventStorageKey, blockHash))
	if err != nil {
		return fmt.Errorf("StateGetStorage get error %v", err)
	}
	event, _ := v.ToString()

	err = ws.SendWsRequest(conn, v, rpc.ChainGetRuntimeVersion(wsSpec, blockHash))

	if err != nil {
		return fmt.Errorf("ChainGetRuntimeVersion get error %v", err)
	}

	var specVersion int

	if r := v.ToRuntimeVersion(); r == nil {
		specVersion = s.CommonService.GetCurrentRuntimeSpecVersion(blockNum)
	} else {
		specVersion = r.SpecVersion
		_ = s.RuntimeService.RegRuntimeVersion(r.ImplName, specVersion, blockHash)
	}

	if specVersion > util.CurrentRuntimeSpecVersion {
		util.CurrentRuntimeSpecVersion = specVersion
	}

	if rpcBlock == nil || specVersion == -1 {
		return errors.New("nil block data")
	}

	// refresh finalized info for update
	if block != nil {
		// Confirm data, only set block Finalized, refresh all block data
		block.ExtrinsicsRoot = rpcBlock.Block.Header.ExtrinsicsRoot
		block.Hash = blockHash
		block.ParentHash = rpcBlock.Block.Header.ParentHash
		block.StateRoot = rpcBlock.Block.Header.StateRoot
		block.Extrinsics = util.ToString(rpcBlock.Block.Extrinsics)
		block.Logs = util.ToString(rpcBlock.Block.Header.Digest.Logs)
		block.Event = event
		block.CodecError = false
		if err = s.BlockService.UpdateBlockData(conn, block, true); err == nil {
			bs.Store(blockNum, true)
		}
		return
	}

	// for Create
	if err = s.BlockService.CreateChainBlock(conn, blockHash, &rpcBlock.Block, event, specVersion, true); err == nil {
		bs.Store(blockNum, true)
	} else {
		log.Error("Create chain block error ", err)
	}
	return

}

func (s *repairService) fillPluginData(blockNum int, bs *model.IntBoolMap) (err error) {
	var block *model.ChainBlock
	var extrinsics []model.ChainExtrinsic
	var events []model.ChainEvent

	if block = s.SqlRepository.GetBlockByNum(blockNum); block == nil {
		return nil
	}
	if extrinsics = s.SqlRepository.GetRawExtrinsicsByBlockNum(blockNum); extrinsics == nil {
		return nil
	}
	if events = s.SqlRepository.GetRawEventByBlockNum(blockNum); events == nil {
		return nil
	}

	eventMap := make(map[string][]model.ChainEvent)
	for _, e := range events {
		extrinsicIndex := fmt.Sprintf("%d-%d", blockNum, e.ExtrinsicIdx)
		eventMap[extrinsicIndex] = append(eventMap[extrinsicIndex], model.ChainEvent{
			BlockNum:      e.BlockNum,
			ExtrinsicIdx:  e.ExtrinsicIdx,
			ModuleId:      e.ModuleId,
			EventId:       e.EventId,
			Params:        e.Params,
			ExtrinsicHash: e.ExtrinsicHash,
			EventIdx:      e.EventIdx,
		})
	}

	feeMap := make(map[string]decimal.Decimal)
	for _, extrinsic := range extrinsics {
		if extrinsic.ExtrinsicHash != "" {
			feeMap[extrinsic.ExtrinsicIndex] = extrinsic.Fee
		}
		err = s.PluginService.EmitExtrinsic(block, &extrinsic, eventMap)
		if err != nil {
			bs.Store(blockNum, false)
			return err
		}
	}
	for _, event := range events {
		err = s.PluginService.EmitEvent(block, &event, feeMap)
		if err != nil {
			log.Info(err)
			bs.Store(blockNum, false)
			return err
		}
	}

	if err == nil {
		bs.Store(blockNum, true)
	}

	return err
}
