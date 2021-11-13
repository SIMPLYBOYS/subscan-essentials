package repository

import (
	"fmt"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/prometheus/common/log"
)

type sqlRewardRepository struct {
	DB m.Dao
}

var PluginPrefix = "reward"

func NewsqlRewardRepository(db m.Dao) model.RewardRepository {
	return &sqlRewardRepository{
		DB: db,
	}
}

func (s *sqlRewardRepository) NewRewardEvent(b *m.Block, e *m.Event, params []m.EventParam) (err error) {
	txn := s.DB.DbBegin()
	defer s.DB.DbRollback(txn)
	// opt := m.Option{PluginPrefix: PluginPrefix}
	r := &model.Reward{
		AccountId:     params[0].Value.(string),
		EventIndex:    fmt.Sprintf("%d-%d", e.BlockNum, e.EventIdx),
		BlockNum:      e.BlockNum,
		ExtrinsicIdx:  e.ExtrinsicIdx,
		ModuleId:      e.ModuleId,
		Amount:        params[1].Value.(string),
		EventId:       e.EventId,
		Params:        e.Params,
		ExtrinsicHash: e.ExtrinsicHash,
		EventIdx:      e.EventIdx,
	}

	tableReward := fmt.Sprintf("%s_%s", PluginPrefix, txn.DB.Unscoped().NewScope(&r).TableName())
	queryReward := txn.DB.Table(tableReward).Create(&r)

	// var rewardlist []model.Reward
	// err = s.DB.FindBy(&rewardlist, map[string]interface{}{"account_id": params[0].Value.(string)}, &opt)

	// a := &model.Account{
	// 	Address: params[0].Value.(string),
	// 	Nonce:   len(rewardlist) + 1,
	// }

	// var ar model.Account
	// err = s.DB.FindBy(&ar, map[string]interface{}{"address": params[0].Value.(string)}, &opt)
	// tableAccount := fmt.Sprintf("%s_%s", PluginPrefix, txn.DB.Unscoped().NewScope(&a).TableName())

	// if ar.ID == 0 {
	// 	_ = txn.DB.Table(tableAccount).Create(&a)
	// } else {
	// 	_ = txn.DB.Table(tableAccount).Where("address = ?", ar.Address).UpdateColumn("nonce", len(rewardlist)+1)
	// }

	if queryReward.Error != nil {
		return queryReward.Error
	}

	s.DB.DbCommit(txn)

	return
}

func (s *sqlRewardRepository) GetRewardListByAddr(page, row int, addr string) ([]model.Reward, error) {
	var rewardlist []model.Reward
	opt := m.Option{PluginPrefix: PluginPrefix, Page: page, PageSize: row, Order: "event_index desc"}
	account := ss58.Decode(addr, util.StringToInt(util.AddressType))
	err := s.DB.FindBy(&rewardlist, map[string]interface{}{"account_id": account}, &opt)
	return rewardlist, err
}

func (s *sqlRewardRepository) GetNonceByAddr(addr string) (int, error) {
	log.Info("=== GetNonceByAddr ==", addr)
	txn := s.DB.DbBegin()
	defer s.DB.DbRollback(txn)
	account := ss58.Decode(addr, util.StringToInt(util.AddressType))
	var count int
	r := &model.Reward{
		AccountId: account,
	}
	tableReward := fmt.Sprintf("%s_%s", PluginPrefix, txn.DB.Unscoped().NewScope(&r).TableName())
	txn.DB.Table(tableReward).Model(&model.Reward{}).Where("account_id = ?", account).Count(&count)

	return count, nil
}
