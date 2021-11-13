package repository

import (
	"fmt"
	"reflect"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/prometheus/common/log"
)

type sqlTransferRepository struct {
	DB m.Dao
}

func NewsqlTransferRepository(db m.Dao) model.TransferRepository {
	return &sqlTransferRepository{
		DB: db,
	}
}

func (s *sqlTransferRepository) NewTransferExtrinsic(b *m.Block, e *m.Extrinsic, params []m.ExtrinsicParam) error {
	txn := s.DB.DbBegin()
	defer s.DB.DbRollback(txn)
	t := &model.Transfer{
		ExtrinsicIndex: e.ExtrinsicIndex,
		ExtrinsicHash:  e.ExtrinsicHash,
		BlockNum:       b.BlockNum,
		BlockTimestamp: b.BlockTimestamp,
		Amount:         params[1].Value.(string),
		Success:        e.Success,
		Fee:            e.Fee,
		FromAddr:       e.AccountId,
		// ToAddr:         params[0].Value.(map[string]interface{})["Id"].(string), // TODO on block 7157677 cause error
		// [{"name":"dest","type":"Address","value":{"Id":"86adbd5205188b0176be681b6a153adb4dd7eb37b3c8766a82cfe3bfdef8ee3d"}},{"name":"value","type":"Compact\u003cBalance\u003e","value":"12447772443"}]
		// [{"name":"dest","type":"Address","value":{"Address20":"0x60e5f32c0353b56cdb7697383e9ef4678c040e77"}},{"name":"value","type":"Compact\u003cBalance\u003e","value":"43000000000"}]
		//    [{"name": "dest","type": "Address","value": "d6b71ad01f548464adebb5f206d5223f3e1997e85e37fcf1a2b07d8650b82301","value_raw": "d6b71ad01f548464adebb5f206d5223f3e1997e85e37fcf1a2b07d8650b82301"},{"name": "value","type": "Compact<Balance>","value": "69000000000","value_raw": "070072b81010"}]
		// [{"name":"dest","type":"Address","value":"2e92f5f2de7a56893a04e3460be0d83d856ac2871bb1779dbb12c767eaddd461"},{"name":"value","type":"Compact\u003cBalance\u003e","value":"100000000000000"}]
	}

	v := reflect.ValueOf(params[0].Value)

	switch v.Kind() {
	case reflect.Map:
		if val, ok := params[0].Value.(map[string]interface{})["Id"].(string); ok {
			t.ToAddr = val
		} else {
			t.ToAddr = params[0].Value.(map[string]interface{})["Address20"].(string)
		}
	case reflect.String:
		t.ToAddr = params[0].Value.(string)
	}

	tableName := fmt.Sprintf("%s_%s", "transfer", txn.DB.Unscoped().NewScope(&t).TableName())
	query := txn.DB.Table(tableName).Create(&t)
	if query.Error == nil {
		s.DB.DbCommit(txn)
		log.Info("New a tranfer extrinsic with extrinsicIndex: ", e.ExtrinsicIndex)
	}
	return query.Error
}

func (s *sqlTransferRepository) GetExtrinsicByIndex(ei string) (model.Transfer, error) {
	var transfer model.Transfer
	opt := m.Option{PluginPrefix: "transfer", Page: 0, PageSize: 10}
	err := s.DB.FindBy(&transfer, map[string]interface{}{"extrinsic_index": ei}, &opt)
	return transfer, err
}

func (s *sqlTransferRepository) GetTransfersList(page, row int) ([]model.Transfer, int) {
	var transfers []model.Transfer
	opt := m.Option{PluginPrefix: "transfer", Page: page, PageSize: row}
	s.DB.FindBy(&transfers, nil, &opt)
	return transfers, len(transfers)
}

func (s *sqlTransferRepository) GetTransfersByAddr(page, row int, addr string) ([]model.Transfer, error) {
	var transfers []model.Transfer
	var v []string
	opt := m.Option{PluginPrefix: "transfer", Page: page, PageSize: row, Order: "block_num desc"}
	account := ss58.Decode(addr, util.StringToInt(util.AddressType))
	v = append(v, fmt.Sprintf("from_addr = '%s'", account))
	v = append(v, fmt.Sprintf("to_addr = '%s'", account))
	err := s.DB.FindBy(&transfers, v, &opt)
	return transfers, err
}
