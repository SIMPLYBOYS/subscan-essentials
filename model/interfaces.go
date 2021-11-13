package model

import (
	"context"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/itering/subscan-plugin/router"
	"github.com/itering/substrate-api-rpc/metadata"
	"github.com/itering/substrate-api-rpc/rpc"
	"github.com/itering/substrate-api-rpc/storage"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type RedisRepository interface {
	Close()
	Ping(context.Context) (err error)
	SetHeartBeatNow(context.Context, string) error
	DaemonHealth(context.Context) map[string]bool
	SaveFillAlreadyBlockNum(context.Context, int) error
	SaveFillAlreadyFinalizedBlockNum(c context.Context, blockNum int) (err error)
	GetFillBestBlockNum(c context.Context) (num int, err error)
	GetFillFinalizedBlockNum(c context.Context) (num int, err error)
	AddMissingBlocks(c context.Context, num int) error
	AddRepairedBlock(c context.Context, num int) error
	AddMissingBlocksInBulk(c context.Context, blockNum int, page, row int) error
	AddRepairedBlocksInBulk(c context.Context, blks []ChainBlock) error

	// AddRepairBlock(c context.Context, num int) error
	SetMetadata(c context.Context, metadata map[string]interface{}) (err error)
	IncrMetadata(c context.Context, filed string, incrNum int) (err error)
	GetMetadata(c context.Context) (ms map[string]string, err error)
	GetBestBlockNum(c context.Context) (uint64, error)
	GetFinalizedBlockNum(c context.Context) (uint64, error)
	GetFinalizedBlockNumForPlugin(c context.Context) (uint64, error)
	GetMissingBlockSet(c context.Context) ([]string, error)
}

type SqlRepository interface {
	Close()
	Migration(blockNum int)
	AddIndex(blockNum int)
	DbBegin() *GormDB
	DbCommit(*GormDB)
	DbRollback(*GormDB)
	CreateBlock(*GormDB, *ChainBlock) (err error)
	UpdateEventAndExtrinsic(*GormDB, *ChainBlock, int, int, int, string, bool, bool) error
	GetNearBlock(int) *ChainBlock
	SetBlockFinalized(*ChainBlock)
	BlocksReverseByNum([]int) map[int]ChainBlock
	GetBlockByHash(c context.Context, hash string, blockNum uint64) *ChainBlock
	CheckDBError(err error) error
	GetBlockByNum(int) *ChainBlock
	GetBlockList(blockNum int, page, row int) []ChainBlock
	BlockAsJson(c context.Context, block *ChainBlock) *ChainBlockJson
	CreateEvent(txn *GormDB, event *ChainEvent) *gorm.DB
	DropEventNotFinalizedData(blockNum int, finalized bool) bool
	GetRawEventByBlockNum(blockNum int, where ...string) []ChainEvent
	GetEventByBlockNum(blockNum int, where ...string) []ChainEventJson
	GetEventList(page, row, blockNum int, order string, where ...string) ([]ChainEvent, int)
	GetEventsByIndex(extrinsicIndex string) []ChainEvent
	GetEventByIdx(index string) *ChainEvent
	CreateExtrinsic(c context.Context, txn *GormDB, extrinsic *ChainExtrinsic) *gorm.DB
	DropExtrinsicNotFinalizedData(c context.Context, blockNum int) *gorm.DB
	GetExtrinsicsByBlockNum(blockNum int) []ChainExtrinsicJson
	GetRawExtrinsicsByBlockNum(blockNum int) []ChainExtrinsic
	GetExtrinsicList(c context.Context, page, row int, order string, blockNum int, ms map[string]string, queryWhere ...string) ([]ChainExtrinsic, int)
	GetExtrinsicsByHash(c context.Context, hash string, blockNum int) *ChainExtrinsic
	GetExtrinsicsDetailByHash(c context.Context, hash string, blockNum int) *ExtrinsicDetail
	GetExtrinsicsDetailByIndex(c context.Context, index string) *ExtrinsicDetail
	ExtrinsicsAsJson(e *ChainExtrinsic) *ChainExtrinsicJson
	CreateLog(txn *GormDB, ce *ChainLog) error
	DropLogsNotFinalizedData(blockNum int, finalized bool) bool
	GetLogsByIndex(index string) *ChainLogJson
	GetLogByBlockNum(blockNum int) []ChainLogJson
	CreateRuntimeVersion(name string, specVersion int) int64
	SetRuntimeData(specVersion int, modules string, rawData string) int64
	RuntimeVersionList() []RuntimeVersion
	RuntimeVersionRaw(spec int) *metadata.RuntimeRaw
	RuntimeVersionRecent() *RuntimeVersion
}

type CommonService interface {
	Close()
	InitSubRuntimeLatest()
	ValidatorsList(conn websocket.WsConn, hash string) (validatorList []string)
	GetCurrentRuntimeSpecVersion(blockNum int) int
	Ping(ctx context.Context, e *empty.Empty) (*empty.Empty, error)
	SetHeartBeat(action string)
	DaemonHealth(ctx context.Context) map[string]bool
	Metadata() (map[string]string, error)
	ReadTypeRegistry() ([]byte, error)
	EmitLog(txn *GormDB, blockNum int, l []storage.DecoderLog, finalized bool, validatorList []string) (validator string, err error)
}

type BlockService interface {
	CreateChainBlock(conn websocket.WsConn, hash string, block *rpc.Block, event string, spec int, finalized bool) (err error)
	UpdateBlockData(conn websocket.WsConn, block *ChainBlock, finalized bool) (err error)
	GetBlocksSampleByNums(page, row int) []SampleBlockJson
	GetMissingBlockMap(blockNum int, page, row int) IntBoolMap
	GetMissingBlockSet(blockNum int, page, row int) ([]string, error)
	GetBlockByHashJson(hash string) *ChainBlockJson
	GetBlockByNum(num int) *ChainBlockJson
	GetBlockByHash(hash string) *ChainBlock
	BlockAsSampleJson(block *ChainBlock) *SampleBlockJson
	GetCurrentBlockNum(c context.Context) (uint64, error)
	InitialMissingBlockSet(blockNum int, page, row int) (mblks []string, err error)
}

type ExtrinsicService interface {
	CheckoutExtrinsicEvents(e []ChainEvent, blockNumInt int) map[string][]ChainEvent
	GetExtrinsicList(page, row int, order string, query ...string) ([]*ChainExtrinsicJson, int)
	GetExtrinsicByIndex(index string) *ExtrinsicDetail
	GetExtrinsicDetailByHash(hash string) *ExtrinsicDetail
	GetExtrinsicByHash(hash string) *ChainExtrinsic
	CreateExtrinsic(c context.Context, txn *GormDB, block *ChainBlock, encodeExtrinsics []string, decodeExtrinsics []map[string]interface{}, eventMap map[string][]ChainEvent) (int, int, map[string]string, map[string]decimal.Decimal, error)
	GetTimestamp(extrinsic *ChainExtrinsic) (blockTimestamp int)
	GetExtrinsicSuccess(e []ChainEvent) bool
	GetExtrinsicFee(p websocket.WsConn, encodeExtrinsic string, blockHash string) (fee decimal.Decimal, err error)
}

type EventService interface {
	EventByIndex(index string) *ChainEvent
	AddEvent(txn *GormDB, block *ChainBlock, e []ChainEvent, hashMap map[string]string, feeMap map[string]decimal.Decimal) (eventCount int, err error)
	RenderEvents(page, row int, order string, where ...string) ([]ChainEventJson, int)
}

type PluginService interface {
	Subscribe(conn websocket.WsConn, interrupt chan os.Signal)
	Parser(message []byte) (err error)
	EmitEvent(block *ChainBlock, event *ChainEvent, feeMap map[string]decimal.Decimal) (err error)
	EmitExtrinsic(block *ChainBlock, extrinsic *ChainExtrinsic, eventsMap map[string][]ChainEvent) (err error)
	PluginsFetchBlock()
	PluginRegister()
}

type RuntimeService interface {
	SubstrateRuntimeList() []RuntimeVersion
	SubstrateRuntimeInfo(spec int) *metadata.Instant
	RegRuntimeVersion(name string, spec int, hash ...string) error
	RegCodecMetadata(hash ...string) (coded string, err error)
	SetRuntimeData(spec int, runtime *metadata.Instant, rawData string) int64
	GetMetadataInstant(spec int, hash string) (metadataInstant *metadata.Instant, err error)
}

type SubscribeService interface {
	Subscribe(conn websocket.WsConn, interrupt chan os.Signal)
	Parser(message []byte) (err error)
	SubscribeFetchBlock()
}

type RepairService interface {
	Repair(conn websocket.WsConn, interrupt chan os.Signal, head, size int)
	RepairBlocks(bs *IntBoolMap)
	RepairBlocksBySet(bs []string)
	RepairPlugins(bs *IntBoolMap)
	PluginRegister()
}

type DB interface {
	// Can query database all tables data
	// Query ** no prefix ** table default, option PluginPrefix can specify other plugin model
	FindBy(record interface{}, query interface{}, option *Option) error

	// Only can exec plugin relate tables
	// Migration
	AutoMigration(model interface{}) error
	// Add column Index
	AddIndex(model interface{}, indexName string, columns ...string) error
	// Add column unique index
	AddUniqueIndex(model interface{}, indexName string, columns ...string) error

	DbBegin() *GormDB

	DbRollback(c *GormDB)

	DbCommit(c *GormDB)

	// Create one record
	Create(c *GormDB, record interface{}) *GormDB
	// Update one or more column
	Update(c *GormDB, model interface{}, query interface{}, attr map[string]interface{}) *GormDB
	// Delete one or more record
	Delete(model interface{}, query interface{}) error
}

type Option struct {
	PluginPrefix string
	PageSize     int
	Page         int
	Order        string
}

type Dao interface {
	DB
	// Find spec metadata raw
	SpecialMetadata(int) string

	// Substrate websocket rpc pool
	RPCPool() *websocket.PoolConn

	// Plugin set prefix
	SetPrefix(string)
}

type Block struct {
	BlockNum       int    `json:"block_num"`
	BlockTimestamp int    `json:"block_timestamp"`
	Hash           string `json:"hash"`
	SpecVersion    int    `json:"spec_version"`
	Validator      string `json:"validator"`
	Finalized      bool   `json:"finalized"`
}

type Extrinsic struct {
	ExtrinsicIndex     string          `json:"extrinsic_index" `
	CallCode           string          `json:"call_code"`
	CallModuleFunction string          `json:"call_module_function" `
	CallModule         string          `json:"call_module"`
	Params             []byte          `json:"params"`
	AccountId          string          `json:"account_id"`
	Signature          string          `json:"signature"`
	Nonce              int             `json:"nonce"`
	Era                string          `json:"era"`
	ExtrinsicHash      string          `json:"extrinsic_hash"`
	Success            bool            `json:"success"`
	Fee                decimal.Decimal `json:"fee"`
}

type Event struct {
	BlockNum      int    `json:"block_num"`
	ExtrinsicIdx  int    `json:"extrinsic_idx"`
	ModuleId      string `json:"module_id"`
	EventId       string `json:"event_id"`
	Params        []byte `json:"params"`
	ExtrinsicHash string `json:"extrinsic_hash"`
	EventIdx      int    `json:"event_idx"`
}

type Plugin interface {
	// Init storage interface
	InitDao(d Dao)

	// Init http router
	InitHttp() []router.Http

	// Receive Extrinsic data when subscribe extrinsic dispatch
	ProcessExtrinsic(*Block, *Extrinsic, []Event) error

	// Receive Extrinsic data when subscribe extrinsic dispatch
	ProcessEvent(*Block, *Event, decimal.Decimal) error

	// Mysql tables schema auto migrate
	Migrate()

	// Subscribe Extrinsic with special module
	SubscribeExtrinsic() []string

	// Subscribe Events with special module
	SubscribeEvent() []string

	// Plugins version
	Version() string
}
