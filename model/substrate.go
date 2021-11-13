package model

import (
	"fmt"
	"sync"

	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

// SplitTableBlockNum
var SplitTableBlockNum = 1000000

const (
	Ok                   = 100001
	UnknownError         = 100002
	AddressValidateError = 100003
	QueryBindingError    = 100004
	DataBaseError        = 100101
)

type GormDB struct {
	*gorm.DB
	GdbDone bool
}

var (
	DaemonAction = []string{"substrate", "plugins", "repair"}
)

type ChainBlock struct {
	ID              uint   `gorm:"primary_key" json:"id"`
	BlockNum        int    `json:"block_num"`
	BlockTimestamp  int    `json:"block_timestamp"`
	Hash            string `sql:"default: null;size:100" json:"hash"`
	ParentHash      string `sql:"default: null;size:100" json:"parent_hash"`
	StateRoot       string `sql:"default: null;size:100" json:"state_root"`
	ExtrinsicsRoot  string `sql:"default: null;size:100" json:"extrinsics_root"`
	Logs            string `json:"logs" sql:"type:text;"`
	Extrinsics      string `json:"extrinsics" sql:"type:MEDIUMTEXT;"`
	EventCount      int    `json:"event_count"`
	ExtrinsicsCount int    `json:"extrinsics_count"`
	Event           string `json:"event" sql:"type:MEDIUMTEXT;"`
	SpecVersion     int    `json:"spec_version"`
	Validator       string `json:"validator"`
	CodecError      bool   `json:"codec_error"`
	Finalized       bool   `json:"finalized"`
}

func (c ChainBlock) TableName() string {
	if c.BlockNum/SplitTableBlockNum == 0 {
		return "chain_blocks"
	}

	return fmt.Sprintf("chain_blocks_%d", c.BlockNum/SplitTableBlockNum)
}

func (c *ChainBlock) AsPlugin() *Block {
	return &Block{
		BlockNum:       c.BlockNum,
		BlockTimestamp: c.BlockTimestamp,
		Hash:           c.Hash,
		SpecVersion:    c.SpecVersion,
		Validator:      c.Validator,
		Finalized:      c.Finalized,
	}
}

type ChainEvent struct {
	ID            uint        `gorm:"primary_key" json:"-"`
	EventIndex    string      `sql:"default: null;size:100;" json:"event_index"`
	BlockNum      int         `json:"block_num" `
	ExtrinsicIdx  int         `json:"extrinsic_idx"`
	Type          string      `json:"-"`
	ModuleId      string      `json:"module_id" `
	EventId       string      `json:"event_id" `
	Params        interface{} `json:"params" sql:"type:text;" `
	ExtrinsicHash string      `json:"extrinsic_hash" sql:"default: null" `
	EventIdx      int         `json:"event_idx"`
}

func (c ChainEvent) TableName() string {
	if c.BlockNum/SplitTableBlockNum == 0 {
		return "chain_events"
	}
	return fmt.Sprintf("chain_events_%d", c.BlockNum/SplitTableBlockNum)
}

func (c *ChainEvent) AsPlugin() *Event {
	return &Event{
		BlockNum:      c.BlockNum,
		ExtrinsicIdx:  c.ExtrinsicIdx,
		ModuleId:      c.ModuleId,
		EventId:       c.EventId,
		Params:        []byte(util.ToString(c.Params)),
		ExtrinsicHash: c.ExtrinsicHash,
		EventIdx:      c.EventIdx,
	}
}

type ChainExtrinsic struct {
	ID                 uint            `gorm:"primary_key"`
	ExtrinsicIndex     string          `json:"extrinsic_index" sql:"default: null;size:100"`
	BlockNum           int             `json:"block_num" `
	BlockTimestamp     int             `json:"block_timestamp"`
	ExtrinsicLength    string          `json:"extrinsic_length"`
	VersionInfo        string          `json:"version_info"`
	CallCode           string          `json:"call_code"`
	CallModuleFunction string          `json:"call_module_function"  sql:"size:100"`
	CallModule         string          `json:"call_module"  sql:"size:100"`
	Params             interface{}     `json:"params" sql:"type:MEDIUMTEXT;" `
	AccountId          string          `json:"account_id"`
	Signature          string          `json:"signature"`
	Nonce              int             `json:"nonce"`
	Era                string          `json:"era"`
	ExtrinsicHash      string          `json:"extrinsic_hash" sql:"default: null" `
	IsSigned           bool            `json:"is_signed"`
	Success            bool            `json:"success"`
	Fee                decimal.Decimal `json:"fee" sql:"type:decimal(30,0);"`
	BatchIndex         int             `json:"-" gorm:"-"`
}

func (c ChainExtrinsic) TableName() string {
	if c.BlockNum/SplitTableBlockNum == 0 {
		return "chain_extrinsics"
	}
	return fmt.Sprintf("chain_extrinsics_%d", c.BlockNum/SplitTableBlockNum)
}

func (c *ChainExtrinsic) AsPlugin() *Extrinsic {
	return &Extrinsic{
		ExtrinsicIndex:     c.ExtrinsicIndex,
		CallModule:         c.CallModule,
		CallModuleFunction: c.CallModuleFunction,
		Params:             []byte(util.ToString(c.Params)),
		AccountId:          c.AccountId,
		Signature:          c.Signature,
		Nonce:              c.Nonce,
		Era:                c.Era,
		ExtrinsicHash:      c.ExtrinsicHash,
		Success:            c.Success,
		Fee:                c.Fee,
	}
}

type RuntimeVersion struct {
	Id          int    `json:"-"`
	Name        string `json:"-"`
	SpecVersion int    `json:"spec_version"`
	Modules     string `json:"modules  sql:"type:TEXT;"`
	RawData     string `json:"-" sql:"type:MEDIUMTEXT;"`
}

type ChainLog struct {
	ID        uint   `gorm:"primary_key"`
	BlockNum  int    `json:"block_num" `
	LogIndex  string `json:"log_index" sql:"default: null;size:100"`
	LogType   string `json:"log_type" `
	Data      string `json:"data" sql:"type:text;"`
	Finalized bool   `json:"finalized"`
}

func (c ChainLog) TableName() string {
	if c.BlockNum/SplitTableBlockNum == 0 {
		return "chain_logs"
	}
	return fmt.Sprintf("chain_logs_%d", c.BlockNum/SplitTableBlockNum)
}

type ExtrinsicParam struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
	// ValueRaw string      `json:"valueRaw"`
}

type EventParam struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type R struct {
	Code        int         `json:"code"`
	Message     string      `json:"message"`
	GeneratedAt int64       `json:"generated_at"`
	Data        interface{} `json:"data,omitempty"`
}

type BlockFinalized struct {
	BlockNum  int  `json:"block_num"`
	Finalized bool `json:"finalized"`
}

type PluginDataSource struct {
	b *ChainBlock             `json:"b"`
	e ChainExtrinsic          `json:"e"`
	c map[string][]ChainEvent `json:"c"`
}

type IntBoolMap struct {
	m sync.Map
}

func (iMap *IntBoolMap) Delete(key int) {
	iMap.m.Delete(key)
}

func (iMap *IntBoolMap) Load(key int) (value bool, ok bool) {
	v, ok := iMap.m.Load(key)
	if v != nil {
		value = v.(bool)
	}
	return
}

func (iMap *IntBoolMap) LoadOrStore(key int, value bool) (actual bool, loaded bool) {
	a, loaded := iMap.m.LoadOrStore(key, value)
	actual = a.(bool)
	return
}

func (iMap *IntBoolMap) Range(f func(key int, value bool) bool) {
	f1 := func(key, value interface{}) bool {
		return f(key.(int), value.(bool))
	}
	iMap.m.Range(f1)
}

func (iMap *IntBoolMap) Store(key int, value bool) {
	iMap.m.Store(key, value)
}
