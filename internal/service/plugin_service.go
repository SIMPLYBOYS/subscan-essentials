package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/gorilla/websocket"
	"github.com/itering/substrate-api-rpc/rpc"
	ws "github.com/itering/substrate-api-rpc/websocket"
	"github.com/panjf2000/ants"
	"github.com/prometheus/common/log"
	"github.com/shopspring/decimal"
)

var (
	subscribeExtrinsic = make(map[string][]plugins.PluginFactory)
	subscribeEvent     = make(map[string][]plugins.PluginFactory)
)

// registered storage
func (p *pluginService) PluginRegister() {
	log.Info("--- PluginRegister ---")
	for name, plugin := range plugins.RegisteredPlugins {
		log.Info("name: ", name)
		db := p.DbStorage
		db.Prefix = name // TODO add network prefix
		plugin.InitDao(db)
		for _, moduleId := range plugin.SubscribeExtrinsic() {
			subscribeExtrinsic[moduleId] = append(subscribeExtrinsic[moduleId], plugin)
		}
		for _, moduleId := range plugin.SubscribeEvent() {
			subscribeEvent[moduleId] = append(subscribeEvent[moduleId], plugin)
		}
	}
}

// TODO get plugin by name method

type pluginService struct {
	newHead         chan bool
	newFinHead      chan bool
	done            chan struct{}
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
	CommonService   model.CommonService
	DbStorage       *DbStorage
}

type PluginConfig struct {
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
	DbStorage       *DbStorage
}

func NewPluginService(c *PluginConfig, cs model.CommonService) model.PluginService {
	return &pluginService{
		RedisRepository: c.RedisRepository,
		SqlRepository:   c.SqlRepository,
		DbStorage:       c.DbStorage,
		CommonService:   cs,
	}
}

func (p *pluginService) initPluginService(done chan struct{}) *pluginService {
	return &pluginService{
		newHead:         make(chan bool, 1),
		newFinHead:      make(chan bool, 1),
		done:            done,
		RedisRepository: p.RedisRepository,
		SqlRepository:   p.SqlRepository,
		CommonService:   p.CommonService,
		DbStorage:       p.DbStorage,
	}
}

func (p *pluginService) Subscribe(conn ws.WsConn, interrupt chan os.Signal) {
	var err error

	signal.Notify(interrupt, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	defer conn.Close()

	done := make(chan struct{})

	pluginSrv := p.initPluginService(done)
	go func() {
		defer close(done)
		for {
			if !conn.IsConnected() {
				continue
			}
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Error("read: ", err)
				continue
			}
			if err = pluginSrv.Parser(message); err != nil {
				log.Error("Parsing error: ", err)
			}
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
				log.Info("SystemHealth get error: ", err)
				if !conn.IsConnected() {
					log.Info("--- SetUp Plugins WebSocket Connection ---")
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

func (p *pluginService) Parser(message []byte) (err error) {
	var j rpc.JsonRpcResult
	if err = json.Unmarshal(message, &j); err != nil {
		return err
	}

	switch j.Method {
	case ChainFinalizedHead:
		go func() {
			p.newFinHead <- true
			onceFinHead.Do(func() {
				go p.PluginsFetchBlock()
			})
		}()
	default:
		return
	}
	return
}

func (p *pluginService) PluginsFetchBlock() {
	log.Info("--- PluginsFetchBlock ---")
	var wg sync.WaitGroup
	var lastNum uint64
	ctx := context.TODO()

	pool, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		blockNum := i.(model.BlockFinalized)
		func(bf model.BlockFinalized) {
			if err := p.fillPluginData(bf.BlockNum, bf.Finalized); err != nil {
				log.Error("fill-in block data cause error ", err)
			} else {
				c := fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, "plugins")
				p.CommonService.SetHeartBeat(c)
			}
		}(blockNum)
		wg.Done()
	}, ants.WithOptions(ants.Options{PanicHandler: func(c interface{}) {}}))

	defer pool.Release()
	for {
		select {
		case <-p.newFinHead:
			final, err := p.RedisRepository.GetFillFinalizedBlockNum(ctx)
			if err != nil || final == 0 {
				time.Sleep(BlockTime * time.Second)
				return
			}
			if lastNum, err = p.RedisRepository.GetFinalizedBlockNumForPlugin(ctx); err != nil {
				log.Warn(err)
			}
			log.Info("plugins sync lastNum: ", lastNum)
			startBlock := lastNum + 1
			if lastNum == 0 {
				startBlock = lastNum
			}
			for i := int(startBlock); i <= int(final-FinalizedWaitingBlockCountForPlugin); i++ {
				wg.Add(1)
				if err := pool.Invoke(model.BlockFinalized{BlockNum: i, Finalized: true}); err != nil {
					log.Error("Invoke fillPluginData error: ", err)
				}
			}
			wg.Wait()
		case <-p.done:
			return
		}
	}
}

func (p *pluginService) fillPluginData(blockNum int, finalized bool) (err error) {
	var block *model.ChainBlock
	var extrinsics []model.ChainExtrinsic
	var events []model.ChainEvent

	log.Info("fillPluginData with block: ", blockNum)

	if bestNum, _ := p.RedisRepository.GetFillBestBlockNum(context.TODO()); blockNum > bestNum {
		return nil
	}

	if block = p.SqlRepository.GetBlockByNum(blockNum); block == nil {
		return nil
	}
	if extrinsics = p.SqlRepository.GetRawExtrinsicsByBlockNum(blockNum); extrinsics == nil {
		return nil
	}
	if events = p.SqlRepository.GetRawEventByBlockNum(blockNum); events == nil {
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
		if err = p.EmitExtrinsic(block, &extrinsic, eventMap); err != nil {
			return err
		}
	}
	for _, event := range events {
		if err = p.EmitEvent(block, &event, feeMap); err != nil {
			return err
		}
	}

	if err == nil {
		p.updateChainMetadata(map[string]interface{}{"plugins:finalized_blockNum": blockNum})
	}

	return nil
}

// after extrinsic created, emit extrinsic data to subscribe plugins
func (p *pluginService) EmitExtrinsic(block *model.ChainBlock, extrinsic *model.ChainExtrinsic, eventsMap map[string][]model.ChainEvent) (err error) {
	pBlock := block.AsPlugin()
	pExtrinsic := extrinsic.AsPlugin()
	events := eventsMap[pExtrinsic.ExtrinsicIndex]

	log.Info("pExtrinsic: ", pExtrinsic.ExtrinsicIndex)
	log.Info(extrinsic.CallModule)
	log.Info(subscribeExtrinsic[extrinsic.CallModule])

	var pEvents []model.Event
	for _, event := range events {
		pEvents = append(pEvents, *event.AsPlugin())
	}

	for _, plugin := range subscribeExtrinsic[extrinsic.CallModule] {
		log.Info("plugin: ", plugin)
		err = plugin.ProcessExtrinsic(pBlock, pExtrinsic, pEvents)
		if err != nil && strings.Contains(err.Error(), "Duplicate entry") != true {
			log.Error(err)
			return err
		}
	}
	return nil
}

// after event created, emit event data to subscribe plugins
func (p *pluginService) EmitEvent(block *model.ChainBlock, event *model.ChainEvent, feeMap map[string]decimal.Decimal) (err error) {

	pBlock := block.AsPlugin()
	pEvent := event.AsPlugin()

	log.Info("pEvent: ", pEvent.ModuleId)
	log.Info(pEvent.ModuleId)
	log.Info(subscribeEvent[event.ModuleId])

	fee := feeMap[event.EventIndex]
	for _, plugin := range subscribeEvent[event.ModuleId] {
		err = plugin.ProcessEvent(pBlock, pEvent, fee)
		// log.Info(strings.Contains(err.Error(), "Duplicate entry"))
		if err != nil && strings.Contains(err.Error(), "Duplicate entry") != true {
			return err
		}
	}
	return nil
}

func (p *pluginService) updateChainMetadata(metadata map[string]interface{}) (err error) {
	c := context.TODO()
	err = p.RedisRepository.SetMetadata(c, metadata)
	return
}
