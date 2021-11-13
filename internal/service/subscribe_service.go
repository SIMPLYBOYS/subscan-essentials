package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/gorilla/websocket"
	"github.com/itering/substrate-api-rpc/rpc"
	ws "github.com/itering/substrate-api-rpc/websocket"
	"github.com/panjf2000/ants"
	"github.com/prometheus/common/log"
)

const (
	FinalizedWaitingBlockCount          = 3
	FinalizedWaitingBlockCountForPlugin = 6
	ChainNewHead                        = "chain_newHead"
	ChainFinalizedHead                  = "chain_finalizedHead"
	StateStorage                        = "state_storage"
	BlockTime                           = 6
)

var (
	onceNewHead, onceFinHead sync.Once
	subscriptionIds          = []subscription{{Topic: ChainNewHead}, {Topic: ChainFinalizedHead}, {Topic: StateStorage}}
)

const (
	runtimeVersion = iota + 1
	newHeader
	finalizeHeader
)

type subscription struct {
	Topic  string `json:"topic"`
	Latest int64  `json:"latest"`
}

type subscribeService struct {
	newHead         chan bool
	newFinHead      chan bool
	done            chan struct{}
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
	CommonService   model.CommonService
	RuntimeService  model.RuntimeService
	BlockService    model.BlockService
}

type SubscribeConfig struct {
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
}

func NewSubscribeService(c *SubscribeConfig, done chan struct{}, cs model.CommonService, r model.RuntimeService, b model.BlockService) model.SubscribeService {
	return &subscribeService{
		newHead:         make(chan bool, 1),
		newFinHead:      make(chan bool, 1),
		done:            done,
		RedisRepository: c.RedisRepository,
		SqlRepository:   c.SqlRepository,
		CommonService:   cs,
		RuntimeService:  r,
		BlockService:    b,
	}
}

func (s *subscribeService) initSubscribeService(done chan struct{}) *subscribeService {
	return &subscribeService{
		newHead:         make(chan bool, 1),
		newFinHead:      make(chan bool, 1),
		done:            done,
		RedisRepository: s.RedisRepository,
		SqlRepository:   s.SqlRepository,
		CommonService:   s.CommonService,
		RuntimeService:  s.RuntimeService,
		BlockService:    s.BlockService,
	}
}

func (s *subscribeService) Subscribe(conn ws.WsConn, interrupt chan os.Signal) {
	var err error

	signal.Notify(interrupt, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	defer conn.Close()

	done := make(chan struct{})

	subscribeSrv := s.initSubscribeService(done)
	go func() {
		defer close(done)
		for {
			if !conn.IsConnected() {
				continue
			}
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Error("read: ", err)
				log.Info("--- Close Substrate mode Connection ---")
				conn.Close()
				continue
			}
			_ = subscribeSrv.Parser(message)
		}
	}()

	if err = conn.WriteMessage(websocket.TextMessage, rpc.ChainGetRuntimeVersion(runtimeVersion)); err != nil {
		log.Info("write: ", err)
	}
	if err = conn.WriteMessage(websocket.TextMessage, rpc.ChainSubscribeNewHead(newHeader)); err != nil {
		log.Info("write: ", err)
	}
	if err = conn.WriteMessage(websocket.TextMessage, rpc.ChainSubscribeFinalizedHeads(finalizeHeader)); err != nil {
		log.Info("write: ", err)
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.TextMessage, rpc.SystemHealth(rand.Intn(100)+finalizeHeader)); err != nil {
				log.Error("SystemHealth get error: ", err)
				if !conn.IsConnected() {
					log.Info("--- SetUp Substrate WebSocket Connection ---")
					conn.CloseAndReconnect()
				}
			}
		case <-interrupt:
			close(done)
			log.Info("interrupt")
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Error("write close: ", err)
				return
			}
			return
		}
	}
}

func (s *subscribeService) Parser(message []byte) (err error) {
	upgradeHealth := func(topic string) {
		for index, subscript := range subscriptionIds {
			if subscript.Topic == topic {
				subscriptionIds[index].Latest = time.Now().Unix()
			}
		}
	}

	var j rpc.JsonRpcResult
	if err = json.Unmarshal(message, &j); err != nil {
		return err
	}

	switch j.Id {
	case runtimeVersion:
		r := j.ToRuntimeVersion()
		_ = s.RuntimeService.RegRuntimeVersion(r.ImplName, r.SpecVersion)
		_ = s.updateChainMetadata(map[string]interface{}{"implName": r.ImplName, "specVersion": r.SpecVersion})
		util.CurrentRuntimeSpecVersion = r.SpecVersion
		return
	}

	switch j.Method {
	case ChainNewHead:
		r := j.ToNewHead()
		_ = s.updateChainMetadata(map[string]interface{}{"blockNum": util.HexToNumStr(r.Number)})
		upgradeHealth(j.Method)
	case ChainFinalizedHead:
		r := j.ToNewHead()
		_ = s.updateChainMetadata(map[string]interface{}{"finalized_blockNum": util.HexToNumStr(r.Number)})
		upgradeHealth(j.Method)
		go func() {
			s.newFinHead <- true
			onceFinHead.Do(func() {
				go s.SubscribeFetchBlock()
			})
		}()
	case StateStorage:
		upgradeHealth(j.Method)
	default:
		return
	}
	return
}

func (s *subscribeService) SubscribeFetchBlock() {
	var wg sync.WaitGroup
	ctx := context.TODO()

	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		blockNum := i.(model.BlockFinalized)
		func(bf model.BlockFinalized) {
			if err := s.fillBlockData(nil, bf.BlockNum, bf.Finalized); err != nil {
				log.Error("fillBlockData get error ", err)
			} else {
				s.CommonService.SetHeartBeat(fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, "substrate"))
			}
		}(blockNum)
		wg.Done()
	}, ants.WithOptions(ants.Options{PanicHandler: func(c interface{}) {}}))

	defer p.Release()
	for {
		select {
		case <-s.newFinHead:
			final, err := s.RedisRepository.GetFinalizedBlockNum(context.TODO())
			if err != nil || final == 0 {
				time.Sleep(BlockTime * time.Second)
				return
			}

			lastNum, _ := s.RedisRepository.GetFillFinalizedBlockNum(ctx)
			startBlock := lastNum + 1
			if lastNum == 0 {
				startBlock = lastNum
			}

			for i := startBlock; i <= int(final-FinalizedWaitingBlockCount); i++ {
				wg.Add(1)
				_ = p.Invoke(model.BlockFinalized{BlockNum: i, Finalized: true})
			}
			wg.Wait()
		case <-s.done:
			return
		}
	}
}

const (
	wsBlockHash = iota + 1
	wsBlock
	wsEvent
	wsSpec
)

func (s *subscribeService) fillBlockData(conn ws.WsConn, blockNum int, finalized bool) (err error) {
	block := s.SqlRepository.GetBlockByNum(blockNum)
	if block != nil && block.Finalized && !block.CodecError && block.ExtrinsicsCount != 0 {
		return nil
	}

	v := &rpc.JsonRpcResult{}

	// Block Hash
	if err = ws.SendWsRequest(conn, v, rpc.ChainGetBlockHash(wsBlockHash, blockNum)); err != nil {
		return fmt.Errorf("websocket send error: %v", err)
	}

	blockHash, err := v.ToString()
	if err != nil || blockHash == "" {
		return fmt.Errorf("ChainGetBlockHash get error %v", err)
	}
	log.Info("Block num: ", blockNum, " hash: ", blockHash)

	// block
	if err = ws.SendWsRequest(conn, v, rpc.ChainGetBlock(wsBlock, blockHash)); err != nil {
		return fmt.Errorf("websocket send error: %v", err)
	}

	rpcBlock := v.ToBlock()
	// event
	if err = ws.SendWsRequest(conn, v, rpc.StateGetStorage(wsEvent, util.EventStorageKey, blockHash)); err != nil {
		return fmt.Errorf("websocket send error: %v", err)
	}

	event, _ := v.ToString()

	// runtime
	if err = ws.SendWsRequest(conn, v, rpc.ChainGetRuntimeVersion(wsSpec, blockHash)); err != nil {
		return fmt.Errorf("websocket send error: %v", err)

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

	var setFinalized = func() {
		if finalized {
			_ = s.RedisRepository.SaveFillAlreadyFinalizedBlockNum(context.TODO(), blockNum)
		}
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
		_ = s.BlockService.UpdateBlockData(conn, block, finalized)
		return
	}

	// for Create
	if err = s.BlockService.CreateChainBlock(conn, blockHash, &rpcBlock.Block, event, specVersion, finalized); err == nil {
		_ = s.RedisRepository.SaveFillAlreadyBlockNum(context.TODO(), blockNum)
		setFinalized()
	} else {
		log.Error("Create chain block error ", err)
	}
	return
}

func (s *subscribeService) updateChainMetadata(metadata map[string]interface{}) (err error) {
	c := context.TODO()
	err = s.RedisRepository.SetMetadata(c, metadata)
	return
}
