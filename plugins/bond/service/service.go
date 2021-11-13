package service

import (
	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/model"
)

type Service struct {
	sql model.BondRepository
}

func New(b model.BondRepository) model.BondService {
	return &Service{
		sql: b,
	}
}

func (s *Service) NewBondExtrinsic(b *m.Block, e *m.Extrinsic, params []m.ExtrinsicParam, status string) (err error) {
	return s.sql.NewBondExtrinsic(b, e, params, status)
}

func (s *Service) GetBondListJson(page, row int, addr string, status string, locked int) ([]model.Bond, error) {
	return s.sql.GetBondListByAddr(page, row, addr, status, locked)
}
