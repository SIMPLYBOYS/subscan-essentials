package service

import (
	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/model"
)

type Service struct {
	sql model.RewardRepository
}

func New(r model.RewardRepository) model.RewardService {
	return &Service{
		sql: r,
	}
}

func (s *Service) NewRewardEvent(b *m.Block, e *m.Event, params []m.EventParam) (err error) {
	return s.sql.NewRewardEvent(b, e, params)
}

func (s *Service) GetRewardListJson(page, row int, addr string) ([]model.Reward, error) {
	return s.sql.GetRewardListByAddr(page, row, addr)
}

func (s *Service) GetAccountNonce(addr string) (int, error) {
	return s.sql.GetNonceByAddr(addr)
}
