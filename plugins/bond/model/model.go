package model

import (
	"github.com/CoolBitX-Technology/subscan/model"
)

type Bond struct {
	ID                      uint   `gorm:"primary_key" json:"-"`
	Account                 string `json:"account"`
	ExtrinsicIndex          string `json:"extrinsic_index" sql:"default: null;size:100"`
	StartAt                 int64  `json:"start_at"`
	Month                   int    `json:"month"`
	Amount                  string `json:"amount" sql:"size:100;"`
	Status                  string `json:"status"`
	ExpireAt                int64  `json:"expired_at"`
	UnbondingExtrinsicIndex string `json:"unbonding_extrinsic_index"`
	UnbondingAt             int64  `json:"unbonding_at"`
	UnbondingEnd            int64  `json:"unbonding_end"`
	Currency                string `json:"currency"`
	Unlock                  bool   `json:"unlock"`
	UnbondingBlockEnd       int    `json:"unbonding_block_end"`
}

type BondDelivery interface {
	BondList(page int, row int, address string, status string, locked int) ([]Bond, error)
}

type BondService interface {
	NewBondExtrinsic(b *model.Block, e *model.Extrinsic, params []model.ExtrinsicParam, status string) error
	GetBondListJson(page, row int, addr string, status string, locked int) ([]Bond, error)
}

type BondRepository interface {
	NewBondExtrinsic(b *model.Block, e *model.Extrinsic, params []model.ExtrinsicParam, status string) error
	GetBondListByAddr(page, row int, addr string, status string, locked int) ([]Bond, error)
}
