package service

import (
	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
)

type Service struct {
	sql model.TransferRepository
}

func New(s model.TransferRepository) model.TransferService {
	return &Service{
		sql: s,
	}
}

func (s *Service) GetTransfersListJson(page, row int, addr string) ([]model.Transfer, error) {
	return s.sql.GetTransfersByAddr(page, row, addr)
}

func (s *Service) BalancesTransaction(b *m.Block, e *m.Extrinsic, params []m.ExtrinsicParam) (err error) {
	return s.sql.NewTransferExtrinsic(b, e, params)
}
