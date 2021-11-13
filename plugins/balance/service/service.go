package service

import (
	"github.com/CoolBitX-Technology/subscan/plugins/balance/dao"
	"github.com/CoolBitX-Technology/subscan/plugins/balance/model"
	"github.com/itering/subscan-plugin/storage"
)

type Service struct {
	d storage.Dao
}

func (s *Service) GetAccountListJson(page, row int) ([]model.Account, int) {
	return dao.GetAccountList(s.d, page, row)
}

func New(d storage.Dao) *Service {
	return &Service{
		d: d,
	}
}
