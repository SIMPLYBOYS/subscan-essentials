package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/itering/substrate-api-rpc/rpc"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/shopspring/decimal"
)

type extrinsicService struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
	PluginService   model.PluginService
}

type ExtrinsicConfig struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
}

func NewExtrinsicService(c *ExtrinsicConfig, p model.PluginService) model.ExtrinsicService {
	return &extrinsicService{
		SqlRepository:   c.SqlRepository,
		RedisRepository: c.RedisRepository,
		PluginService:   p,
	}
}

func (e *extrinsicService) GetExtrinsicDetailByHash(hash string) *model.ExtrinsicDetail {
	c := context.TODO()
	blockNum, _ := e.RedisRepository.GetFillFinalizedBlockNum(c)
	return e.SqlRepository.GetExtrinsicsDetailByHash(c, hash, blockNum)
}

func (e *extrinsicService) GetExtrinsicList(page, row int, order string, query ...string) ([]*model.ChainExtrinsicJson, int) {
	c := context.TODO()
	blockNum, _ := e.RedisRepository.GetFillFinalizedBlockNum(c)
	ms, _ := e.RedisRepository.GetMetadata(c)
	list, count := e.SqlRepository.GetExtrinsicList(c, page, row, order, blockNum, ms, query...)
	var ejs []*model.ChainExtrinsicJson
	for _, extrinsic := range list {
		ejs = append(ejs, e.SqlRepository.ExtrinsicsAsJson(&extrinsic))
	}
	return ejs, count
}

func (s *extrinsicService) CheckoutExtrinsicEvents(e []model.ChainEvent, blockNumInt int) map[string][]model.ChainEvent {
	eventMap := make(map[string][]model.ChainEvent)
	for _, event := range e {
		extrinsicIndex := fmt.Sprintf("%d-%d", blockNumInt, event.ExtrinsicIdx)
		// log.Info("")
		// log.Info("-----------------------")
		// log.Info("extrinsicIndex: ", extrinsicIndex)
		// log.Info("event: ", event)
		// log.Info("-----------------------")
		// log.Info("")
		eventMap[extrinsicIndex] = append(eventMap[extrinsicIndex], event)
	}
	return eventMap
}

func (s *extrinsicService) CreateExtrinsic(c context.Context,
	txn *model.GormDB,
	block *model.ChainBlock,
	encodeExtrinsics []string,
	decodeExtrinsics []map[string]interface{},
	eventMap map[string][]model.ChainEvent,
) (int, int, map[string]string, map[string]decimal.Decimal, error) {
	var (
		blockTimestamp int
		e              []model.ChainExtrinsic
		err            error
	)
	extrinsicFee := make(map[string]decimal.Decimal)

	eb, _ := json.Marshal(decodeExtrinsics)
	_ = json.Unmarshal(eb, &e)

	hash := make(map[string]string)

	for index, extrinsic := range e {
		extrinsic.CallModule = strings.ToLower(extrinsic.CallModule)
		extrinsic.BlockNum = block.BlockNum
		extrinsic.ExtrinsicIndex = fmt.Sprintf("%d-%d", extrinsic.BlockNum, index)
		extrinsic.Success = s.GetExtrinsicSuccess(eventMap[extrinsic.ExtrinsicIndex])

		if tp := s.getTimestamp(&extrinsic); tp > 0 {
			blockTimestamp = tp
		}

		block.BlockTimestamp = blockTimestamp
		extrinsic.BlockTimestamp = blockTimestamp

		if extrinsic.ExtrinsicHash != "" {
			fee, err := s.GetExtrinsicFee(nil, encodeExtrinsics[index], block.Hash)
			if err != nil {
				return 0, 0, nil, nil, err
			}
			extrinsic.Fee = fee
			extrinsicFee[extrinsic.ExtrinsicIndex] = fee
			hash[extrinsic.ExtrinsicIndex] = extrinsic.ExtrinsicHash
		}
		query := s.SqlRepository.CreateExtrinsic(c, txn, &extrinsic)

		if query.RowsAffected > 0 {
			_ = s.RedisRepository.IncrMetadata(c, "count_extrinsic", 1)
			if extrinsic.Signature != "" {
				_ = s.RedisRepository.IncrMetadata(c, "count_signed_extrinsic", 1)
			}
		}

		if err := s.SqlRepository.CheckDBError(query.Error); err != nil {
			return 0, 0, nil, nil, err
		}

		// deprecated fill-in process
		// if err := s.SqlRepository.CheckDBError(query.Error); err == nil {
		// go s.PluginService.EmitExtrinsic(block, &extrinsic, eventMap)
		// } else {
		// 	return 0, 0, nil, nil, err
		// }
	}
	return len(e), blockTimestamp, hash, extrinsicFee, err
}

func (s *extrinsicService) GetExtrinsicSuccess(e []model.ChainEvent) bool {
	for _, event := range e {
		if strings.EqualFold(event.ModuleId, "system") {
			return !strings.EqualFold(event.EventId, "ExtrinsicFailed")
		}
	}
	return true
}

func (s *extrinsicService) getTimestamp(extrinsic *model.ChainExtrinsic) (blockTimestamp int) {
	if extrinsic.CallModule != "timestamp" {
		return
	}

	var paramsInstant []model.ExtrinsicParam
	util.UnmarshalAny(&paramsInstant, extrinsic.Params)

	for _, p := range paramsInstant {
		if p.Name == "now" {
			extrinsic.BlockTimestamp = util.IntFromInterface(p.Value)
			return extrinsic.BlockTimestamp
		}
	}
	return
}

func (s *extrinsicService) getPaymentQueryInfo(p websocket.WsConn, encodedExtrinsic string, blockHash string) (error, *rpc.PaymentQueryInfo) {
	v := &rpc.JsonRpcResult{}
	r := rpc.Param{Id: rand.Intn(10000), Method: "payment_queryInfo", Params: []string{encodedExtrinsic, blockHash}}
	r.JsonRpc = "2.0"
	b, _ := json.Marshal(r)
	err := websocket.SendWsRequest(p, v, b)
	return err, v.ToPaymentQueryInfo()
}

func (s *extrinsicService) GetExtrinsicFee(p websocket.WsConn, encodedExtrinsic string, blockHash string) (fee decimal.Decimal, err error) {
	var paymentInfo *rpc.PaymentQueryInfo
	// for i := 0; i < retry; i++ {
	err, paymentInfo = s.getPaymentQueryInfo(p, encodedExtrinsic, blockHash)

	// if err == nil {
	// 	break
	// }

	// fmt.Fprintf(os.Stderr, "Request error: %+v\n", err)
	// fmt.Fprintf(os.Stderr, "%d times Retrying in %v\n", i+1, 5*time.Second)
	// time.Sleep(5 * time.Second)
	// }
	if paymentInfo != nil {
		return paymentInfo.PartialFee, nil
	}
	return decimal.Zero, err
}

func (s *extrinsicService) GetExtrinsicByIndex(index string) *model.ExtrinsicDetail {
	c := context.TODO()
	return s.SqlRepository.GetExtrinsicsDetailByIndex(c, index)
}

func (s *extrinsicService) GetExtrinsicByHash(hash string) *model.ChainExtrinsic {
	c := context.TODO()
	blockNum, _ := s.RedisRepository.GetFillBestBlockNum(c)
	return s.SqlRepository.GetExtrinsicsByHash(c, hash, blockNum)
}

func (s *extrinsicService) GetTimestamp(extrinsic *model.ChainExtrinsic) (blockTimestamp int) {
	if extrinsic.CallModule != "timestamp" {
		return
	}

	var paramsInstant []model.ExtrinsicParam
	util.UnmarshalAny(&paramsInstant, extrinsic.Params)

	for _, p := range paramsInstant {
		if p.Name == "now" {
			extrinsic.BlockTimestamp = util.IntFromInterface(p.Value)
			return extrinsic.BlockTimestamp
		}
	}
	return
}
