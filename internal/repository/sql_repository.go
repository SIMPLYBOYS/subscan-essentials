package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/address"
	"github.com/go-sql-driver/mysql"
	"github.com/itering/substrate-api-rpc/metadata"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/common/log"
)

type sqlRepository struct {
	DB *gorm.DB
}

var protectedTables []string

func NewSqlRepository(sqlClient *gorm.DB) model.SqlRepository {
	return &sqlRepository{
		DB: sqlClient,
	}
}

func (s *sqlRepository) Close() {
	if s.DB != nil {
		_ = s.DB.Close()
	}
}

func (s *sqlRepository) Migration(blockNum int) {
	_ = s.DB.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(s.InternalTables(blockNum)...)

	for i := 0; i <= blockNum/model.SplitTableBlockNum; i++ {
		s.AddIndex(i * model.SplitTableBlockNum)
	}
}

func (s *sqlRepository) AddIndex(blockNum int) {

	if blockNum == 0 {
		s.DB.Model(model.RuntimeVersion{}).AddUniqueIndex("spec_version", "spec_version")
	}

	blockModel := model.ChainBlock{BlockNum: blockNum}
	eventModel := model.ChainEvent{BlockNum: blockNum}
	extrinsicModel := model.ChainExtrinsic{BlockNum: blockNum}
	logModel := model.ChainLog{BlockNum: blockNum}

	s.DB.Model(blockModel).AddUniqueIndex("hash", "hash")
	s.DB.Model(blockModel).AddUniqueIndex("block_num", "block_num")
	_ = s.DB.Model(blockModel).AddIndex("codec_error", "codec_error")

	s.DB.Model(extrinsicModel).AddIndex("extrinsic_hash", "extrinsic_hash")
	s.DB.Model(extrinsicModel).AddUniqueIndex("extrinsic_index", "extrinsic_index")
	s.DB.Model(extrinsicModel).AddIndex("block_num", "block_num")
	s.DB.Model(extrinsicModel).AddIndex("is_signed", "is_signed")
	s.DB.Model(extrinsicModel).AddIndex("account_id", "is_signed,account_id")
	s.DB.Model(extrinsicModel).AddIndex("call_module", "call_module")
	s.DB.Model(extrinsicModel).AddIndex("call_module_function", "call_module_function")

	s.DB.Model(eventModel).AddIndex("block_num", "block_num")
	s.DB.Model(eventModel).AddIndex("type", "type")
	s.DB.Model(eventModel).AddIndex("event_index", "event_index")
	s.DB.Model(eventModel).AddIndex("event_id", "event_id")
	s.DB.Model(eventModel).AddIndex("module_id", "module_id")
	s.DB.Model(eventModel).AddUniqueIndex("event_idx", "event_index", "event_idx")

	s.DB.Model(logModel).AddUniqueIndex("log_index", "log_index")
	s.DB.Model(logModel).AddIndex("block_num", "block_num")
}

func (s *sqlRepository) GetBlockList(blockNum, page, row int) []model.ChainBlock {
	var blocks []model.ChainBlock
	head := blockNum - page*row
	if head < 0 {
		return nil
	}
	end := head - row
	if end < 0 {
		end = 0
	}

	s.DB.Model(model.ChainBlock{BlockNum: head}).
		Joins(fmt.Sprintf("JOIN (SELECT id,block_num from %s where block_num BETWEEN %d and %d order by block_num desc ) as t on %s.id=t.id",
			model.ChainBlock{BlockNum: head}.TableName(),
			end, head,
			model.ChainBlock{BlockNum: head}.TableName(),
		)).
		Order("block_num desc").Scan(&blocks)

	if head/model.SplitTableBlockNum != end/model.SplitTableBlockNum {
		var endBlocks []model.ChainBlock
		s.DB.Model(model.ChainBlock{BlockNum: blockNum - model.SplitTableBlockNum}).
			Joins(fmt.Sprintf("JOIN (SELECT id,block_num from %s order by block_num desc limit %d) as t on %s.id=t.id",
				model.ChainBlock{BlockNum: blockNum - model.SplitTableBlockNum}.TableName(),
				row-(head%model.SplitTableBlockNum+1),
				model.ChainBlock{BlockNum: blockNum - model.SplitTableBlockNum}.TableName(),
			)).
			Order("block_num desc").Scan(&endBlocks)
		blocks = append(blocks, endBlocks...)
	}

	return blocks
}

func (s *sqlRepository) RuntimeVersionRaw(spec int) *metadata.RuntimeRaw {
	var one metadata.RuntimeRaw
	query := s.DB.Model(model.RuntimeVersion{}).
		Select("spec_version as spec ,raw_data as raw").
		Where("spec_version = ?", spec).
		Scan(&one)
	if query.RecordNotFound() {
		return nil
	}
	return &one
}

func (s *sqlRepository) RuntimeVersionList() []model.RuntimeVersion {
	var list []model.RuntimeVersion
	s.DB.Select("spec_version,modules").Model(model.RuntimeVersion{}).Find(&list)
	return list
}

func (s *sqlRepository) RuntimeVersionRecent() *model.RuntimeVersion {
	var list model.RuntimeVersion
	query := s.DB.Select("spec_version,raw_data").Model(model.RuntimeVersion{}).Order("spec_version DESC").First(&list)
	if query.RecordNotFound() {
		return nil
	}
	return &list
}

func (s *sqlRepository) SetRuntimeData(specVersion int, modules string, rawData string) int64 {
	query := s.DB.Model(model.RuntimeVersion{}).Where("spec_version=?", specVersion).UpdateColumn(model.RuntimeVersion{
		Modules: modules,
		RawData: rawData,
	})
	return query.RowsAffected
}

func (s *sqlRepository) CreateRuntimeVersion(name string, specVersion int) int64 {
	query := s.DB.Create(&model.RuntimeVersion{
		Name:        name,
		SpecVersion: specVersion,
	})
	return query.RowsAffected
}

func (s *sqlRepository) GetLogByBlockNum(blockNum int) []model.ChainLogJson {
	var logs []model.ChainLogJson
	query := s.DB.Model(&model.ChainLog{BlockNum: blockNum}).
		Where("block_num =?", blockNum).Order("id asc").Scan(&logs)
	if query == nil || query.Error != nil || query.RecordNotFound() {
		return nil
	}
	return logs
}

func (s *sqlRepository) GetExtrinsicsByHash(c context.Context, hash string, blockNum int) *model.ChainExtrinsic {
	var extrinsic model.ChainExtrinsic
	for index := blockNum / (model.SplitTableBlockNum); index >= 0; index-- {
		query := s.DB.Model(model.ChainExtrinsic{BlockNum: index * model.SplitTableBlockNum}).Where("extrinsic_hash = ?", hash).Order("id asc").Limit(1).Scan(&extrinsic)
		if query != nil && !query.RecordNotFound() {
			return &extrinsic
		}
	}
	return nil
}

func (s *sqlRepository) GetLogsByIndex(index string) *model.ChainLogJson {
	var Log model.ChainLogJson
	indexArr := strings.Split(index, "-")
	query := s.DB.Model(model.ChainLog{BlockNum: util.StringToInt(indexArr[0])}).Where("log_index = ?", index).Scan(&Log)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return &Log
}

func (s *sqlRepository) DropLogsNotFinalizedData(blockNum int, finalized bool) bool {
	var delExist bool
	if finalized {
		query := s.DB.Where("block_num = ?", blockNum).
			Delete(model.ChainLog{BlockNum: blockNum})
		delExist = query.RowsAffected > 0
	}
	return delExist
}

func (s *sqlRepository) CreateLog(txn *model.GormDB, ce *model.ChainLog) error {
	query := txn.Create(ce)
	return s.CheckDBError(query.Error)
}

func (s *sqlRepository) CheckDBError(err error) error {
	if err == mysql.ErrInvalidConn || err == driver.ErrBadConn {
		return err
	}
	return nil
}

func (s *sqlRepository) ExtrinsicsAsJson(e *model.ChainExtrinsic) *model.ChainExtrinsicJson {
	ej := &model.ChainExtrinsicJson{
		BlockNum:           e.BlockNum,
		BlockTimestamp:     e.BlockTimestamp,
		ExtrinsicIndex:     e.ExtrinsicIndex,
		ExtrinsicHash:      e.ExtrinsicHash,
		Success:            e.Success,
		CallModule:         e.CallModule,
		CallModuleFunction: e.CallModuleFunction,
		From:               address.SS58Address(e.AccountId),
		Signature:          e.Signature,
		Nonce:              e.Nonce,
		Fee:                e.Fee,
	}

	if block := s.GetBlockByNum(e.BlockNum); block != nil {
		ej.Finalized = block.Finalized
	}

	util.UnmarshalAny(&ej.Params, e.Params)
	// for pi, param := range ej.Params {
	// 	if ej.Params[pi].Type == "Address" {
	// 		ej.Params[pi].Value = address.SS58Address(param.Value.(map[string]interface{})["Id"].(string))
	// 	}
	// }
	return ej
}

func (s *sqlRepository) GetExtrinsicList(c context.Context, page, row int, order string, blockNum int, ms map[string]string, queryWhere ...string) ([]model.ChainExtrinsic, int) {
	var extrinsics []model.ChainExtrinsic
	var count int

	for index := blockNum / model.SplitTableBlockNum; index >= 0; index-- {
		var tableData []model.ChainExtrinsic
		var tableCount int
		queryOrigin := s.DB.Model(model.ChainExtrinsic{BlockNum: index * model.SplitTableBlockNum})
		for _, w := range queryWhere {
			queryOrigin = queryOrigin.Where(w)
		}

		queryOrigin.Count(&tableCount)

		if tableCount == 0 {
			continue
		}
		preCount := count
		count += tableCount
		if len(extrinsics) >= row {
			continue
		}
		query := queryOrigin.Order("block_num desc").Offset(page*row - preCount).Limit(row - len(extrinsics)).Scan(&tableData)
		if query == nil || query.Error != nil || query.RecordNotFound() {
			continue
		}
		extrinsics = append(extrinsics, tableData...)

	}

	if len(queryWhere) == 0 {
		count = util.StringToInt(ms["count_extrinsic"])
	}
	return extrinsics, count
}

func (s *sqlRepository) GetExtrinsicsDetailByHash(c context.Context, hash string, blockNum int) *model.ExtrinsicDetail {
	if extrinsic := s.GetExtrinsicsByHash(c, hash, blockNum); extrinsic != nil {
		return s.extrinsicsAsDetail(c, extrinsic)
	}
	return nil
}

func (s *sqlRepository) GetExtrinsicsDetailByIndex(c context.Context, index string) *model.ExtrinsicDetail {
	var extrinsic model.ChainExtrinsic
	indexArr := strings.Split(index, "-")
	query := s.DB.Model(model.ChainExtrinsic{BlockNum: util.StringToInt(indexArr[0])}).
		Where("extrinsic_index = ?", index).Scan(&extrinsic)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return s.extrinsicsAsDetail(c, &extrinsic)
}

func (s *sqlRepository) GetBlockByNum(blockNum int) *model.ChainBlock {
	var block model.ChainBlock
	query := s.DB.Model(&model.ChainBlock{BlockNum: blockNum}).Where("block_num = ?", blockNum).Scan(&block)
	if query == nil || query.Error != nil || query.RecordNotFound() {
		return nil
	}
	return &block
}

func (s *sqlRepository) GetEventsByIndex(extrinsicIndex string) []model.ChainEvent {
	var Event []model.ChainEvent
	indexArr := strings.Split(extrinsicIndex, "-")
	query := s.DB.Model(model.ChainEvent{BlockNum: util.StringToInt(indexArr[0])}).
		Where("event_index = ?", extrinsicIndex).Scan(&Event)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return Event
}

func (s *sqlRepository) extrinsicsAsDetail(c context.Context, e *model.ChainExtrinsic) *model.ExtrinsicDetail {
	detail := model.ExtrinsicDetail{
		BlockTimestamp:     e.BlockTimestamp,
		ExtrinsicIndex:     e.ExtrinsicIndex,
		BlockNum:           e.BlockNum,
		CallModule:         e.CallModule,
		CallModuleFunction: e.CallModuleFunction,
		AccountId:          address.SS58Address(e.AccountId),
		Signature:          e.Signature,
		Nonce:              e.Nonce,
		ExtrinsicHash:      e.ExtrinsicHash,
		Success:            e.Success,
		Fee:                e.Fee,
	}
	util.UnmarshalAny(&detail.Params, e.Params)

	if block := s.GetBlockByNum(detail.BlockNum); block != nil {
		detail.Finalized = block.Finalized
	}

	events := s.GetEventsByIndex(e.ExtrinsicIndex)
	for k, event := range events {
		events[k].Params = util.ToString(event.Params)
	}

	detail.Event = &events

	return &detail
}

func (s *sqlRepository) DropEventNotFinalizedData(blockNum int, finalized bool) bool {
	var delExist bool
	if finalized {
		query := s.DB.Where("block_num = ?", blockNum).Delete(model.ChainEvent{BlockNum: blockNum})
		delExist = query.RowsAffected > 0
	}
	return delExist
}

func (s *sqlRepository) CreateEvent(txn *model.GormDB, event *model.ChainEvent) *gorm.DB {
	extrinsicHash := util.AddHex(event.ExtrinsicHash)
	e := model.ChainEvent{
		EventIndex:    event.EventIndex,
		BlockNum:      event.BlockNum,
		Type:          event.Type,
		ModuleId:      event.ModuleId,
		Params:        util.ToString(event.Params),
		EventIdx:      event.EventIdx,
		EventId:       event.EventId,
		ExtrinsicIdx:  event.ExtrinsicIdx,
		ExtrinsicHash: extrinsicHash,
	}
	query := txn.Create(&e)
	return query
}

func (s *sqlRepository) DbBegin() *model.GormDB {
	txn := s.DB.Begin()
	if txn.Error != nil {
		panic(txn.Error)
	}
	return &model.GormDB{txn, false}
}

func (s *sqlRepository) DbRollback(c *model.GormDB) {
	if c.GdbDone {
		return
	}
	tx := c.Rollback()
	c.GdbDone = true
	if err := tx.Error; err != nil && err != sql.ErrTxDone {
		log.Error("Fatal error DbRollback", err)
	}
}

func (s *sqlRepository) DropExtrinsicNotFinalizedData(c context.Context, blockNum int) *gorm.DB {
	query := s.DB.Where("block_num = ?", blockNum).Delete(model.ChainExtrinsic{BlockNum: blockNum})
	return query
}

func (s *sqlRepository) CreateExtrinsic(c context.Context, txn *model.GormDB, extrinsic *model.ChainExtrinsic) *gorm.DB {
	ce := model.ChainExtrinsic{
		BlockTimestamp:     extrinsic.BlockTimestamp,
		ExtrinsicIndex:     extrinsic.ExtrinsicIndex,
		BlockNum:           extrinsic.BlockNum,
		ExtrinsicLength:    extrinsic.ExtrinsicLength,
		VersionInfo:        extrinsic.VersionInfo,
		CallCode:           extrinsic.CallCode,
		CallModuleFunction: extrinsic.CallModuleFunction,
		CallModule:         extrinsic.CallModule,
		Params:             util.ToString(extrinsic.Params),
		AccountId:          extrinsic.AccountId,
		Signature:          extrinsic.Signature,
		Era:                extrinsic.Era,
		ExtrinsicHash:      util.AddHex(extrinsic.ExtrinsicHash),
		Nonce:              extrinsic.Nonce,
		Success:            extrinsic.Success,
		IsSigned:           extrinsic.Signature != "",
		Fee:                extrinsic.Fee,
	}
	query := txn.Create(&ce)
	return query
}

func (s *sqlRepository) CreateBlock(txn *model.GormDB, cb *model.ChainBlock) (err error) {
	query := txn.Create(cb)
	if !s.DB.HasTable(model.ChainBlock{BlockNum: cb.BlockNum + model.SplitTableBlockNum}) {
		go func() {
			_ = s.DB.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(
				s.InternalTables(cb.BlockNum + model.SplitTableBlockNum)...)
			s.AddIndex(cb.BlockNum + model.SplitTableBlockNum)
		}()
	}
	return query.Error
}

func (s *sqlRepository) DbCommit(c *model.GormDB) {
	if c.GdbDone {
		return
	}
	tx := c.Commit()
	c.GdbDone = true
	if err := tx.Error; err != nil && err != sql.ErrTxDone {
		log.Error("Fatal error DbCommit", err)
	}
}

func (s *sqlRepository) InternalTables(blockNum int) (models []interface{}) {
	models = append(models, model.RuntimeVersion{})
	for i := 0; i <= blockNum/model.SplitTableBlockNum; i++ {
		models = append(
			models,
			model.ChainBlock{BlockNum: blockNum},
			model.ChainEvent{BlockNum: blockNum},
			model.ChainExtrinsic{BlockNum: blockNum},
			model.ChainLog{BlockNum: blockNum})
	}
	var tablesName []string
	for _, m := range models {
		tablesName = append(tablesName, s.DB.Unscoped().NewScope(m).TableName()) // TODO add network prefix like polkadot / kusama
	}
	protectedTables = tablesName
	return models
}

func (s *sqlRepository) UpdateEventAndExtrinsic(txn *model.GormDB, block *model.ChainBlock, eventCount, extrinsicsCount, blockTimestamp int, validator string, codecError bool, finalized bool) error {
	query := txn.Where("block_num = ?", block.BlockNum).Model(block).UpdateColumn(map[string]interface{}{
		"event_count":      eventCount,
		"extrinsics_count": extrinsicsCount,
		"block_timestamp":  blockTimestamp,
		"validator":        validator,
		"codec_error":      codecError,
		"hash":             block.Hash,
		"parent_hash":      block.ParentHash,
		"state_root":       block.StateRoot,
		"extrinsics_root":  block.ExtrinsicsRoot,
		"extrinsics":       block.Extrinsics,
		"event":            block.Event,
		"logs":             block.Logs,
		"finalized":        finalized,
	})
	return query.Error
}

func (s *sqlRepository) GetNearBlock(blockNum int) *model.ChainBlock {
	var block model.ChainBlock
	query := s.DB.Model(&model.ChainBlock{BlockNum: blockNum}).Where("block_num > ?", blockNum).Order("block_num desc").Scan(&block)
	if query == nil || query.Error != nil || query.RecordNotFound() {
		return nil
	}
	return &block
}

func (s *sqlRepository) SetBlockFinalized(block *model.ChainBlock) {
	s.DB.Model(block).UpdateColumn(model.ChainBlock{Finalized: true})
}

func (s *sqlRepository) BlocksReverseByNum(blockNums []int) map[int]model.ChainBlock {
	var blocks []model.ChainBlock
	if len(blockNums) == 0 {
		return nil
	}
	sort.Ints(blockNums)
	lastNum := blockNums[len(blockNums)-1]
	for index := lastNum / model.SplitTableBlockNum; index >= 0; index-- {
		var tableData []model.ChainBlock
		query := s.DB.Model(model.ChainBlock{BlockNum: index * model.SplitTableBlockNum}).Where("block_num in (?)", blockNums).Scan(&tableData)
		if query == nil || query.Error != nil || query.RecordNotFound() {
			continue
		}
		blocks = append(blocks, tableData...)
	}

	toMap := make(map[int]model.ChainBlock)
	for _, block := range blocks {
		toMap[block.BlockNum] = block
	}

	return toMap
}

func (s *sqlRepository) GetBlockByHash(c context.Context, hash string, blockNum uint64) *model.ChainBlock {
	var block model.ChainBlock
	for index := int(blockNum / uint64(model.SplitTableBlockNum)); index >= 0; index-- {
		query := s.DB.Model(&model.ChainBlock{BlockNum: index * model.SplitTableBlockNum}).Where("hash = ?", hash).Scan(&block)
		if query != nil && !query.RecordNotFound() {
			return &block
		}
	}
	return nil
}

func (s *sqlRepository) BlockAsJson(c context.Context, block *model.ChainBlock) *model.ChainBlockJson {
	bj := model.ChainBlockJson{
		BlockNum:        block.BlockNum,
		BlockTimestamp:  block.BlockTimestamp,
		Hash:            block.Hash,
		ParentHash:      block.ParentHash,
		StateRoot:       block.StateRoot,
		EventCount:      block.EventCount,
		ExtrinsicsCount: block.ExtrinsicsCount,
		ExtrinsicsRoot:  block.ExtrinsicsRoot,
		Extrinsics:      s.GetExtrinsicsByBlockNum(block.BlockNum),
		Events:          s.GetEventByBlockNum(block.BlockNum),
		Logs:            s.GetLogByBlockNum(block.BlockNum),
		Validator:       address.SS58Address(block.Validator),
		Finalized:       block.Finalized,
	}
	return &bj
}

func (s *sqlRepository) GetRawExtrinsicsByBlockNum(blockNum int) []model.ChainExtrinsic {
	var extrinsics []model.ChainExtrinsic
	query := s.DB.Model(model.ChainExtrinsic{BlockNum: blockNum}).
		Where("block_num = ?", blockNum).Order("id asc").Scan(&extrinsics)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return extrinsics
}

func (s *sqlRepository) GetExtrinsicsByBlockNum(blockNum int) []model.ChainExtrinsicJson {
	var extrinsics []model.ChainExtrinsic
	query := s.DB.Model(model.ChainExtrinsic{BlockNum: blockNum}).
		Where("block_num = ?", blockNum).Order("id asc").Scan(&extrinsics)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	var list []model.ChainExtrinsicJson
	for _, extrinsic := range extrinsics {
		list = append(list, *s.ExtrinsicsAsJson(&extrinsic))
	}
	return list
}

func (s *sqlRepository) GetRawEventByBlockNum(blockNum int, where ...string) []model.ChainEvent {
	var events []model.ChainEvent
	queryOrigin := s.DB.Model(model.ChainEvent{BlockNum: blockNum}).Where("block_num = ?", blockNum)
	for _, w := range where {
		queryOrigin = queryOrigin.Where(w)
	}
	query := queryOrigin.Order("id asc").Scan(&events)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return events
}

func (s *sqlRepository) GetEventByBlockNum(blockNum int, where ...string) []model.ChainEventJson {
	var events []model.ChainEventJson
	queryOrigin := s.DB.Model(model.ChainEvent{BlockNum: blockNum}).Where("block_num = ?", blockNum)
	for _, w := range where {
		queryOrigin = queryOrigin.Where(w)
	}
	query := queryOrigin.Order("id asc").Scan(&events)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return events
}

func (s *sqlRepository) GetEventList(page, row, blockNum int, order string, where ...string) ([]model.ChainEvent, int) {
	var Events []model.ChainEvent
	var count int

	for index := blockNum / model.SplitTableBlockNum; index >= 0; index-- {
		var tableData []model.ChainEvent
		var tableCount int
		queryOrigin := s.DB.Model(model.ChainEvent{BlockNum: index * model.SplitTableBlockNum})
		for _, w := range where {
			queryOrigin = queryOrigin.Where(w)
		}

		queryOrigin.Count(&tableCount)

		if tableCount == 0 {
			continue
		}
		preCount := count
		count += tableCount
		if len(Events) >= row {
			continue
		}
		query := queryOrigin.Order(fmt.Sprintf("block_num %s", order)).Offset(page*row - preCount).Limit(row - len(Events)).Scan(&tableData)
		if query == nil || query.Error != nil || query.RecordNotFound() {
			continue
		}
		Events = append(Events, tableData...)

	}
	return Events, count
}

func (s *sqlRepository) GetEventByIdx(index string) *model.ChainEvent {
	var Event model.ChainEvent
	indexArr := strings.Split(index, "-")
	if len(indexArr) < 2 {
		return nil
	}
	query := s.DB.Model(model.ChainEvent{BlockNum: util.StringToInt(indexArr[0])}).
		Where("block_num = ?", indexArr[0]).
		Where("event_idx = ?", indexArr[1]).Scan(&Event)
	if query == nil || query.RecordNotFound() {
		return nil
	}
	return &Event
}
