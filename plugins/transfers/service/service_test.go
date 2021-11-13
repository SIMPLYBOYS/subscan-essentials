package service_test

import (
	"testing"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model/mocks"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type page struct {
	Row     int
	Page    int
	Address string
}

func TestGetTransferList(t *testing.T) {
	mockTransferRepo := new(mocks.TransferRepository)

	mockTransfer := []model.Transfer{
		{
			ExtrinsicIndex: "5098085-1",
			ExtrinsicHash:  "4d7b855621bcf6a87c1e22093189d293bf33dba50cceb120e067964220d59468",
			BlockNum:       5098085,
			BlockTimestamp: 1621229040,
			Amount:         "13735927092600",
			Success:        true,
			Fee:            decimal.New(160000014, 0),
			FromAddr:       "d21a5689680a5e569d3c4370d2a94daab5fbdf5befaa07b58d0a1658b0c6a4ad",
			ToAddr:         "0a16963b40d8d28338f7f586a96fa93d2062620f8b57d393c52c907501b8797f",
		},
	}

	t.Run("Success", func(t *testing.T) {
		p := page{Row: 10, Page: 0, Address: "15kUt2i86LHRWCkE3D9Bg1HZAoc2smhn1fwPzDERTb1BXAkX"}
		mockTransferRepo.On("GetTransfersByAddr", p.Page, p.Row, p.Address).Return(mockTransfer, nil).Once()
		m := service.New(mockTransferRepo)
		txs, _ := m.GetTransfersListJson(p.Page, p.Row, p.Address)
		assert.Equal(t, len(txs), 1)
		mockTransferRepo.AssertCalled(t, "GetTransfersByAddr", p.Page, p.Row, p.Address)
	})
}

func TestNewTransferExtrinsic(t *testing.T) {
	mockTransferRepo := new(mocks.TransferRepository)

	mockBlock := m.Block{
		BlockNum:       5095844,
		BlockTimestamp: 1621215588,
		Hash:           "0x78f3105efd294a89bef4034c17587fe0d691843e0dd4158a778dfb8a3a7f8e17",
		SpecVersion:    30,
		Validator:      "de1491e4b9f70678bf4eecc652a9e392fa6d2ccebee58879ad463aeda836fe33",
		Finalized:      true,
	}

	mockExtrinsic := m.Extrinsic{
		ExtrinsicIndex:     "5095844-1",
		CallModuleFunction: "transfer",
		CallModule:         "balances",
		AccountId:          "f4e1f21a11b5b74c2b43f26bff6a046a2e1f16d08bbe0dc9ba7582bdb4745c6b",
		Signature:          "0x241be89f526f67c065d36f1c02fa98da6880f49993e5703b8e988f57c535fb7bd34abed790df9c3715d68f2ae5f1981cd9cf6a78977745e7a8be4b8b8a0cb784",
		Nonce:              0,
		Era:                "0502",
		ExtrinsicHash:      "0x0240a7f02414712568d4d0a6ae360b9ed9d9d8b9b39e788d0b76d8fde8451f14",
		Success:            true,
		Fee:                decimal.New(156000015, 0),
	}

	mockExtrinsicParam := []m.ExtrinsicParam{
		{Type: "Address", Value: map[string]interface{}{"Id": "8274c1f83eda177fafc715e5845bcb424a5c5b6939abfedc504c7d72aec18fa8"}},
		{Type: "Compact<Balance>", Value: "193309000000000"},
		{Type: "Balance", Value: "142408084752"},
	}

	t.Run("Sucess", func(t *testing.T) {
		s := service.New(mockTransferRepo)
		mockTransferRepo.On("NewTransferExtrinsic", &mockBlock, &mockExtrinsic, mockExtrinsicParam).Return(nil)
		e := s.BalancesTransaction(&mockBlock, &mockExtrinsic, mockExtrinsicParam)
		assert.NoError(t, e)
		mockTransferRepo.AssertCalled(t, "NewTransferExtrinsic", &mockBlock, &mockExtrinsic, mockExtrinsicParam)
	})
}
