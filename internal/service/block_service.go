package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/address"
	"github.com/itering/substrate-api-rpc"
	"github.com/itering/substrate-api-rpc/rpc"
	"github.com/itering/substrate-api-rpc/storage"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/prometheus/common/log"
)

type blockService struct {
	SqlRepository    model.SqlRepository
	RedisRepository  model.RedisRepository
	RuntimeService   model.RuntimeService
	ExtrinsicService model.ExtrinsicService
	EventService     model.EventService
	CommonService    model.CommonService
}

type BlockConfig struct {
	SqlRepository   model.SqlRepository
	RedisRepository model.RedisRepository
}

func NewBlockService(c *BlockConfig, r model.RuntimeService, e model.ExtrinsicService, s model.EventService, cs model.CommonService) model.BlockService {
	return &blockService{
		SqlRepository:    c.SqlRepository,
		RedisRepository:  c.RedisRepository,
		RuntimeService:   r,
		ExtrinsicService: e,
		EventService:     s,
		CommonService:    cs,
	}
}

func (b *blockService) GetBlocksSampleByNums(page, row int) []model.SampleBlockJson {
	var blockJson []model.SampleBlockJson
	blockNum, _ := b.RedisRepository.GetFillBestBlockNum(context.TODO())
	blocks := b.SqlRepository.GetBlockList(blockNum, page, row)
	for _, block := range blocks {
		bj := b.BlockAsSampleJson(&block)
		blockJson = append(blockJson, *bj)
	}
	return blockJson
}

func (b *blockService) GetMissingBlockMap(blockNum int, page, row int) (bs model.IntBoolMap) {
	log.Info("=== GetMissingBlockMap ===")
	blocks := b.SqlRepository.GetBlockList(blockNum, page, row)
	for i := 0; i < (page+1)*row; i++ {
		bs.Store(blockNum-i, false)
	}
	for _, block := range blocks {
		if block.EventCount == 0 || block.ExtrinsicsCount == 0 {
			bs.Store(block.BlockNum, false)
		} else {
			bs.Store(block.BlockNum, true)
		}
	}
	return bs
}

func (b *blockService) InitialMissingBlockSet(blockNum int, page, row int) (mblks []string, err error) {
	c := context.TODO()
	log.Info("=== GetMissingBlockSet ===")
	b.RedisRepository.AddMissingBlocksInBulk(c, blockNum, page, row)
	blocks := b.SqlRepository.GetBlockList(blockNum, page, row)
	for _, block := range blocks {
		if block.EventCount != 0 && block.ExtrinsicsCount != 0 {
			b.RedisRepository.AddRepairedBlock(c, block.BlockNum)
		}
	}

	if mblks, err = b.RedisRepository.GetMissingBlockSet(c); err != nil {
		return nil, err
	}
	return mblks, nil
}

func (b *blockService) GetMissingBlockSet(blockNum int, page, row int) (mblks []string, err error) {
	c := context.TODO()
	if mblks, err = b.RedisRepository.GetMissingBlockSet(c); err != nil {
		return nil, err
	}
	return mblks, nil
}

func (b *blockService) GetCurrentBlockNum(c context.Context) (uint64, error) {
	return b.RedisRepository.GetBestBlockNum(c)
}

func (b *blockService) GetExtrinsicByHash(hash string) *model.ChainExtrinsic {
	c := context.TODO()
	blockNum, _ := b.RedisRepository.GetFillBestBlockNum(c)
	return b.SqlRepository.GetExtrinsicsByHash(c, hash, blockNum)
}

func (b *blockService) BlockAsSampleJson(block *model.ChainBlock) *model.SampleBlockJson {
	m := model.SampleBlockJson{
		BlockNum:        block.BlockNum,
		BlockTimestamp:  block.BlockTimestamp,
		Hash:            block.Hash,
		EventCount:      block.EventCount,
		ExtrinsicsCount: block.ExtrinsicsCount,
		Validator:       address.SS58Address(block.Validator),
		Finalized:       block.Finalized,
	}
	return &m
}

func (b *blockService) CreateChainBlock(conn websocket.WsConn, hash string, block *rpc.Block, event string, spec int, finalized bool) (err error) {
	var (
		decodeExtrinsics []map[string]interface{}
		decodeEvent      interface{}
		logs             []storage.DecoderLog
		validator        string
	)
	c := context.TODO()
	blockNum := util.StringToInt(util.HexToNumStr(block.Header.Number))
	metadataInstant, err := b.RuntimeService.GetMetadataInstant(spec, hash)
	if err != nil {
		log.Error(err)
	}

	// Extrinsic

	// log.Info("block.Extrinsics: ", block.Extrinsics)
	// log.Info("metadataInstant: ", metadataInstant)
	decodeExtrinsics, err = substrate.DecodeExtrinsic(block.Extrinsics, metadataInstant, spec)
	if err != nil {
		log.Error(err)
		return err
	}

	// event
	if err == nil {
		decodeEvent, err = substrate.DecodeEvent(event, metadataInstant, spec)
		if err != nil {
			log.Error(err)
		}
	}

	// log
	if err == nil {
		logs, err = substrate.DecodeLogDigest(block.Header.Digest.Logs)
		if err != nil {
			log.Error(err)
		}
	}

	txn := b.SqlRepository.DbBegin()
	defer b.SqlRepository.DbRollback(txn)

	var e []model.ChainEvent
	util.UnmarshalAny(&e, decodeEvent)

	eventMap := b.ExtrinsicService.CheckoutExtrinsicEvents(e, blockNum)

	cb := model.ChainBlock{
		Hash:           hash,
		BlockNum:       blockNum,
		ParentHash:     block.Header.ParentHash,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Logs:           util.ToString(block.Header.Digest.Logs),
		Extrinsics:     util.ToString(block.Extrinsics),
		Event:          event,
		SpecVersion:    spec,
		Finalized:      finalized,
	}

	extrinsicsCount, _, extrinsicHash, extrinsicFee, err := b.ExtrinsicService.CreateExtrinsic(c, txn, &cb, block.Extrinsics, decodeExtrinsics, eventMap)

	// log.Info("")
	// log.Info("=====================================")
	// log.Info("extrinsicsCount: ", extrinsicsCount)
	// log.Info("extrinsicHash: ", extrinsicHash)
	// log.Info("extrinsicFee: ", extrinsicFee)
	// log.Info("blockTimestamp: ", blockTimestamp)
	// log.Info("=====================================")

	if err != nil {
		return err
	}
	eventCount, err := b.EventService.AddEvent(txn, &cb, e, extrinsicHash, extrinsicFee)
	if err != nil {
		return err
	}
	if validator, err = b.CommonService.EmitLog(txn, blockNum, logs, finalized, b.CommonService.ValidatorsList(conn, hash)); err != nil {
		return err
	}

	cb.Validator = validator
	cb.CodecError = validator == "" && blockNum != 0
	cb.ExtrinsicsCount = extrinsicsCount
	cb.EventCount = eventCount

	if err = b.SqlRepository.CreateBlock(txn, &cb); err == nil {
		log.Info("CreateChainBlock ", blockNum, " @", time.Now())
		b.SqlRepository.DbCommit(txn)
	}
	return err
}

func (b *blockService) UpdateBlockData(conn websocket.WsConn, block *model.ChainBlock, finalized bool) (err error) {
	c := context.TODO()

	var (
		decodeEvent      interface{}
		encodeExtrinsics []string
		decodeExtrinsics []map[string]interface{}
	)

	_ = json.Unmarshal([]byte(block.Extrinsics), &encodeExtrinsics)

	spec := block.SpecVersion
	metadataInstant, err := b.RuntimeService.GetMetadataInstant(spec, block.Hash)
	if err != nil {
		log.Error(err)
		return
	}

	// Event
	decodeEvent, err = substrate.DecodeEvent(block.Event, metadataInstant, spec)
	if err != nil {
		log.Info("ERR: Decode Event get error ", err, " @block: ", block.BlockNum)
		return
	}

	// Extrinsic
	decodeExtrinsics, err = substrate.DecodeExtrinsic(encodeExtrinsics, metadataInstant, spec)
	if err != nil {
		log.Info("ERR: Decode Extrinsic get error ", err)
		return
	}

	// Log
	var rawList []string
	_ = json.Unmarshal([]byte(block.Logs), &rawList)
	logs, err := substrate.DecodeLogDigest(rawList)
	if err != nil {
		log.Info("ERR: Decode Logs get error ", err)
		return
	}

	var e []model.ChainEvent
	util.UnmarshalAny(&e, decodeEvent)

	for _, event := range e {
		log.Info("event: ", event)
	}
	eventMap := b.ExtrinsicService.CheckoutExtrinsicEvents(e, block.BlockNum)

	txn := b.SqlRepository.DbBegin()
	defer b.SqlRepository.DbRollback(txn)

	extrinsicsCount, blockTimestamp, extrinsicHash, extrinsicFee, err := b.ExtrinsicService.CreateExtrinsic(c, txn, block, encodeExtrinsics, decodeExtrinsics, eventMap)
	if err != nil {
		return err
	}

	eventCount, err := b.EventService.AddEvent(txn, block, e, extrinsicHash, extrinsicFee)
	if err != nil {
		return err
	}

	validator, err := b.CommonService.EmitLog(txn, block.BlockNum, logs, finalized, b.CommonService.ValidatorsList(conn, block.Hash))
	if err != nil {
		return err
	}

	// TODO check validator blank issue !!
	log.Info("BlockNum:", block.BlockNum, " eventCount: ", eventCount, "\n\n\n")
	// if err = b.SqlRepository.UpdateEventAndExtrinsic(txn, block, eventCount, extrinsicsCount, blockTimestamp, validator, validator == "" && block.BlockNum != 0, finalized); err != nil {
	// 	return
	// }

	if err = b.SqlRepository.UpdateEventAndExtrinsic(txn, block, eventCount, extrinsicsCount, blockTimestamp, validator, false, finalized); err != nil {
		log.Info("UpdateChainBlock ", block.BlockNum, " @", time.Now())
		return
	}

	b.SqlRepository.DbCommit(txn)
	return
}

func (b *blockService) GetBlockByHashJson(hash string) *model.ChainBlockJson {
	c := context.TODO()
	blockNum, _ := b.RedisRepository.GetBestBlockNum(c)
	block := b.SqlRepository.GetBlockByHash(c, hash, blockNum)
	if block == nil {
		return nil
	}
	return b.SqlRepository.BlockAsJson(c, block)
}

func (b *blockService) GetBlockByNum(num int) *model.ChainBlockJson {
	c := context.TODO()
	block := b.SqlRepository.GetBlockByNum(num)
	if block == nil {
		return nil
	}
	return b.SqlRepository.BlockAsJson(c, block)
}

func (b *blockService) GetBlockByHash(hash string) *model.ChainBlock {
	c := context.TODO()
	blockNum, _ := b.RedisRepository.GetBestBlockNum(c)
	block := b.SqlRepository.GetBlockByHash(c, hash, blockNum)
	if block == nil {
		return nil
	}
	return block
}
