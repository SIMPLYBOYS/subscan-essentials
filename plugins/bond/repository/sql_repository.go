package repository

import (
	"fmt"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/prometheus/common/log"
)

type sqlBondRepository struct {
	DB m.Dao
}

func NewsqlBondRepository(db m.Dao) model.BondRepository {
	return &sqlBondRepository{
		DB: db,
	}
}

func (s *sqlBondRepository) NewBondExtrinsic(b *m.Block, e *m.Extrinsic, params []m.ExtrinsicParam, status string) error {
	var amount string
	var bond *model.Bond
	accoutId := e.AccountId
	txn := s.DB.DbBegin()
	defer s.DB.DbRollback(txn)

	if len(params) == 3 {
		amount = params[1].Value.(string)
	} else {
		amount = params[0].Value.(string)
	}

	switch status {
	case "bonded":
		bond = &model.Bond{
			ExtrinsicIndex: e.ExtrinsicIndex,
			Account:        accoutId,
			StartAt:        int64(b.BlockTimestamp) * 1000,
			Status:         status,
			Amount:         amount,
			ExpireAt:       int64(b.BlockTimestamp) * 1000,
		}
	case "unbonding":
		bond = &model.Bond{
			ExtrinsicIndex:          e.ExtrinsicIndex,
			Account:                 accoutId,
			StartAt:                 int64(b.BlockTimestamp) * 1000,
			Status:                  status,
			Amount:                  amount,
			ExpireAt:                int64(b.BlockTimestamp) * 1000,
			UnbondingExtrinsicIndex: e.ExtrinsicIndex,
			UnbondingAt:             int64(b.BlockTimestamp) * 1000,
			UnbondingEnd:            int64(b.BlockTimestamp+1209600) * 1000,
			UnbondingBlockEnd:       b.BlockNum + 403200,
		}
	}

	tableName := fmt.Sprintf("%s_%s", "bond", txn.DB.Unscoped().NewScope(&bond).TableName())
	query := txn.DB.Table(tableName).Create(&bond)
	if query.Error == nil {
		s.DB.DbCommit(txn)
		log.Info("New a ", status, " extrinsic ", " with extrinsicIndex: ", e.ExtrinsicIndex)
	}
	return query.Error
}

func (s *sqlBondRepository) GetBondListByAddr(page, row int, addr string, status string, locked int) ([]model.Bond, error) {
	var bondlist []model.Bond
	opt := m.Option{PluginPrefix: "bond", Page: page, PageSize: row, Order: "start_at desc"}
	account := ss58.Decode(addr, util.StringToInt(util.AddressType))
	err := s.DB.FindBy(&bondlist, map[string]interface{}{"account": account, "status": status, "unlock": locked}, &opt)

	return bondlist, err
}
