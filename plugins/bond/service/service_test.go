package service_test

import (
	"testing"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/model/mocks"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type page struct {
	Row     int
	Page    int
	Address string
}

func TestGetBondList(t *testing.T) {
	mockBondListRepo := new(mocks.BondRepository)

	mockBond := []model.Bond{
		{
			ExtrinsicIndex: "5099018-3",
			Account:        "3475d6301e958ac5d93f9b4d7cc261c467d609e8155cfdb5968aac34de1e5c5e",
			StartAt:        1621234638,
			Amount:         "3937708845038",
			Status:         "bonding",
			Unlock:         false,
		},
	}

	t.Run("Success", func(t *testing.T) {
		p := page{Row: 10, Page: 0, Address: "12BnVhXxGBZXoq9QAkSv9UtVcdBs1k38yNx6sHUJWasTgYrm"}
		mockBondListRepo.On("GetBondListByAddr", p.Page, p.Row, p.Address, "bonding", 0).Return(mockBond, nil)
		s := service.New(mockBondListRepo)
		bonds, _ := s.GetBondListJson(p.Page, p.Row, p.Address, "bonding", 0)
		assert.Equal(t, len(bonds), 1)
		mockBondListRepo.AssertCalled(t, "GetBondListByAddr", p.Page, p.Row, p.Address, "bonding", 0)
	})
}

func TestNewBondExtrinsic(t *testing.T) {
	mockBondListRepo := new(mocks.BondRepository)

	mockBlock := m.Block{
		BlockNum:       5099018,
		BlockTimestamp: 1621234908,
		Hash:           "0x51df7d36dd2d9d4e2987da8a3f67d6334e6af0dfa399175e6d581663e196f065",
		SpecVersion:    30,
		Validator:      "9c665073980c9bdbd5620ef9a860b9f1efbeda8f10e13ef7431f6970d765a257",
		Finalized:      true,
	}

	mockExtrinsic := m.Extrinsic{
		ExtrinsicIndex:     "5099018-3",
		CallModuleFunction: "bond",
		CallModule:         "staking",
		AccountId:          "631495cbcbdf6d04a65863ed55e3e84d94337ad37bbd07d1f77f4fef5bfd9934",
		Signature:          "0x5c313d8a1711708c67bba4e323714b5cc376da7f80f17d208bdaaa692b04245a8444b2b38728d432c1568af6fa4d69e9304bfbf801f5db41f9c57c301c3f130d",
		Nonce:              2,
		Era:                "a503",
		ExtrinsicHash:      "0x837887bc7f1a556a8b5570b32de2e821ec4b84868cce6b1793fa81e3298b0eb1",
		Success:            true,
		Fee:                decimal.New(158000047, 0),
	}

	mockExtrinsicParam := []m.ExtrinsicParam{
		{Type: "AccountId", Value: "5db6ff3eb16cf438f588cd7e3bc1de9d055a4e3beaa888f078b6ff2ab18ba45a"},
		{Type: "Balance", Value: "142408084752"},
	}

	t.Run("Sucess", func(t *testing.T) {
		s := service.New(mockBondListRepo)
		mockBondListRepo.On("NewBondExtrinsic", &mockBlock, &mockExtrinsic, mockExtrinsicParam, "bonded").Return(nil)
		e := s.NewBondExtrinsic(&mockBlock, &mockExtrinsic, mockExtrinsicParam, "bonded")
		assert.NoError(t, e)
		mockBondListRepo.AssertCalled(t, "NewBondExtrinsic", &mockBlock, &mockExtrinsic, mockExtrinsicParam, "bonded")
	})
}
