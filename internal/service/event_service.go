package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/shopspring/decimal"
)

type eventService struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
	PluginService   model.PluginService
}

type EventConfig struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
}

func NewEventService(c *EventConfig, p model.PluginService) model.EventService {
	return &eventService{
		SqlRepository:   c.SqlRepository,
		RedisRepository: c.RedisRepository,
		PluginService:   p,
	}
}

func (s *eventService) AddEvent(txn *model.GormDB, block *model.ChainBlock, e []model.ChainEvent, hashMap map[string]string, feeMap map[string]decimal.Decimal) (eventCount int, err error) {
	var incrCount int
	for _, event := range e {
		event.ModuleId = strings.ToLower(event.ModuleId)
		event.ExtrinsicHash = hashMap[fmt.Sprintf("%d-%d", block.BlockNum, event.ExtrinsicIdx)]
		event.EventIndex = fmt.Sprintf("%d-%d", block.BlockNum, event.ExtrinsicIdx)
		event.BlockNum = block.BlockNum

		// log.Info("")
		// log.Info("eeeeeeeeeeeeeeeeeeeeeeeeeeeee")
		// log.Info("ModuleId: ", event.ModuleId)
		// log.Info("blockNum: ", block.BlockNum)
		// log.Info("EventIndex: ", event.EventIndex)
		// log.Info("ExtrinsicHash: ", event.ExtrinsicHash)
		// log.Info("Fee", feeMap[event.EventIndex])
		// log.Info("feeMap", feeMap)
		// log.Info("eeeeeeeeeeeeeeeeeeeeeeeeeeeee")

		query := s.SqlRepository.CreateEvent(txn, &event)
		if query.RowsAffected > 0 {
			incrCount++
			_ = s.RedisRepository.IncrMetadata(context.TODO(), "count_event", incrCount)
		}

		if err = s.SqlRepository.CheckDBError(query.Error); err != nil {
			return 0, err
		}

		// deprecated fill-in process
		// if err = s.SqlRepository.CheckDBError(query.Error); err == nil {
		// 	log.Info("=== BEFORE EMITEVENT ===!!")
		// 	log.Info("event.EventIndex: ", event.EventIndex)
		// 	log.Info("Fee:", feeMap[event.EventIndex])
		// 	log.Info("=== BEFORE EMITEVENT ===!!")
		// 	// go s.PluginService.EmitEvent(block, &event, feeMap)
		// } else {
		// 	return 0, err
		// }
		eventCount++
	}
	return eventCount, err
}

func (s *eventService) EventByIndex(index string) *model.ChainEvent {
	return s.SqlRepository.GetEventByIdx(index)
}

func (s *eventService) RenderEvents(page, row int, order string, where ...string) ([]model.ChainEventJson, int) {
	var (
		result    []model.ChainEventJson
		blockNums []int
	)

	blockNum, _ := s.RedisRepository.GetFillBestBlockNum(context.TODO())
	list, count := s.SqlRepository.GetEventList(page, row, blockNum, order, where...)
	for _, event := range list {
		blockNums = append(blockNums, event.BlockNum)
	}
	blockMap := s.SqlRepository.BlocksReverseByNum(blockNums)

	for _, event := range list {
		ej := model.ChainEventJson{
			ExtrinsicIdx:  event.ExtrinsicIdx,
			EventIndex:    event.EventIndex,
			BlockNum:      event.BlockNum,
			ModuleId:      event.ModuleId,
			EventId:       event.EventId,
			Params:        util.ToString(event.Params),
			EventIdx:      event.EventIdx,
			ExtrinsicHash: event.ExtrinsicHash,
		}
		if block, ok := blockMap[event.BlockNum]; ok {
			ej.BlockTimestamp = block.BlockTimestamp
		}
		result = append(result, ej)
	}
	return result, count
}
