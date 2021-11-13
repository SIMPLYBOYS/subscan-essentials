package model

import (
	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/shopspring/decimal"
)

type Transfer struct {
	ID             uint            `gorm:"primary_key" json:"-"`
	ExtrinsicIndex string          `json:"extrinsic_index" sql:"default: null;size:100"`
	ExtrinsicHash  string          `json:"extrinsic_hash" sql:"size:100;"`
	BlockNum       int             `json:"block_num"`
	BlockTimestamp int             `json:"block_timestamp"`
	Amount         string          `json:"amount" sql:"size:100;"`
	Success        bool            `json:"success"`
	Fee            decimal.Decimal `json:"fee" sql:"type:decimal(30,0);"`
	FromAddr       string          `json:"from_addr"`
	ToAddr         string          `json:"to_addr"`
}

type TransferDelivery interface {
	TransferList(page int, row int, address string) ([]Transfer, error)
}

type TransferService interface {
	GetTransfersListJson(page, row int, addr string) ([]Transfer, error)
	BalancesTransaction(b *model.Block, e *model.Extrinsic, params []model.ExtrinsicParam) error
}

type TransferRepository interface {
	NewTransferExtrinsic(b *model.Block, e *model.Extrinsic, params []model.ExtrinsicParam) error
	GetExtrinsicByIndex(ei string) (Transfer, error)
	GetTransfersList(page, row int) ([]Transfer, int)
	GetTransfersByAddr(page, row int, addr string) ([]Transfer, error)
}
