package model

import (
	"github.com/CoolBitX-Technology/subscan/model"
)

type Reward struct {
	ID            uint        `gorm:"primary_key" json:"-"`
	AccountId     string      `json:"account"`
	Amount        string      `json:"amount" sql:"size:100;"`
	EventIndex    string      `json:"event_index" sql:"default: null;size:100"`
	BlockNum      int         `json:"block_num"`
	ExtrinsicIdx  int         `json:"extrinsic_idx" sql:"default: null;size:100"`
	ModuleId      string      `json:"module_id"`
	EventId       string      `json:"event_id"`
	Params        interface{} `json:"params" sql:"type:text;" `
	ExtrinsicHash string      `json:"extrinsic_hash" sql:"size:100;"`
	EventIdx      int         `json:"event_idx"`
}

type Account struct {
	ID      uint   `gorm:"primary_key" json:"-"`
	Address string `sql:"default: null;size:100" json:"address"`
	Nonce   int    `json:"nonce"`
}

type RewardDelivery interface {
	RewardList(page int, row int, address string) ([]Reward, int, error)
}

type RewardService interface {
	NewRewardEvent(b *model.Block, e *model.Event, params []model.EventParam) error
	GetRewardListJson(page, row int, addr string) ([]Reward, error)
	GetAccountNonce(addr string) (int, error)
}

type RewardRepository interface {
	NewRewardEvent(b *model.Block, e *model.Event, params []model.EventParam) error
	GetRewardListByAddr(page, row int, addr string) ([]Reward, error)
	GetNonceByAddr(addr string) (int, error)
}
