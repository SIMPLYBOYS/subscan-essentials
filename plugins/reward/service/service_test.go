package service_test

import (
	"testing"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/model/mocks"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/service"
	"github.com/stretchr/testify/assert"
)

type page struct {
	Row     int
	Page    int
	Address string
}

func TestGetRewardList(t *testing.T) {
	mockRewardListRepo := new(mocks.RewardRepository)

	mockReward := []model.Reward{
		{
			AccountId:     "76729e17ad31469debcb60f3ce3622f79143e442e77b58d6e2195d9ea998680d",
			Amount:        "484253744395",
			EventIndex:    "5096104-1",
			BlockNum:      5096104,
			ExtrinsicIdx:  1,
			ModuleId:      "staking",
			EventId:       "Reward",
			Params:        "[{\"type\":\"AccountId\",\"value\":\"ee34a3280459b5bfa65127bc69ca669d0273476b30c3c6f613c3468383f0e078\"},{\"type\":\"Balance\",\"value\":\"2370093221\"}]",
			ExtrinsicHash: "0x36328559e0b714b48956bba78e782f0a07e1f7185b606b0e32945560b75bd6de",
			EventIdx:      1,
		},
	}

	t.Run("Success", func(t *testing.T) {
		p := page{Row: 10, Page: 0, Address: "13gJhYAWEuZomHsp7nBushCwqizG5ZPXoNZ2Z9hP5dynmcnJ"}
		mockRewardListRepo.On("GetRewardListByAddr", p.Page, p.Row, p.Address).Return(mockReward, nil)
		s := service.New(mockRewardListRepo)
		rewards, _ := s.GetRewardListJson(p.Page, p.Row, p.Address)
		assert.Equal(t, len(rewards), 1)
		mockRewardListRepo.AssertCalled(t, "GetRewardListByAddr", p.Page, p.Row, p.Address)
	})
}

func TestNewRewardEvent(t *testing.T) {
	mockRewardListRepo := new(mocks.RewardRepository)

	mockBlock := m.Block{
		BlockNum:       5096104,
		BlockTimestamp: 1621217148,
		Hash:           "0x0dd1681b802bbb270d1cd91bd11bfeada72993241f7340d01086ddca60def9f1",
		SpecVersion:    30,
		Validator:      "80d8a3f4317249a895e4b49badcfa7293cfbd215d6e552d1c07024d36acfbd5d",
		Finalized:      true,
	}

	mockEvent := m.Event{
		BlockNum:      5096104,
		ExtrinsicIdx:  1,
		ModuleId:      "staking",
		EventId:       "Reward",
		Params:        []byte("[{\"type\":\"AccountId\",\"value\":\"76729e17ad31469debcb60f3ce3622f79143e442e77b58d6e2195d9ea998680d\"},{\"type\":\"Balance\",\"value\":\"484253744395\"}]"),
		ExtrinsicHash: "0x36328559e0b714b48956bba78e782f0a07e1f7185b606b0e32945560b75bd6de",
		EventIdx:      1,
	}

	mockParams := []m.EventParam{{
		Type:  "AccountId",
		Value: "76729e17ad31469debcb60f3ce3622f79143e442e77b58d6e2195d9ea998680d",
	}, {
		Type:  "Balance",
		Value: "484253744395",
	}}

	t.Run("Success", func(t *testing.T) {
		s := service.New(mockRewardListRepo)
		mockRewardListRepo.On("NewRewardEvent", &mockBlock, &mockEvent, mockParams).Return(nil)
		e := s.NewRewardEvent(&mockBlock, &mockEvent, mockParams)
		assert.NoError(t, e)
		mockRewardListRepo.AssertCalled(t, "NewRewardEvent", &mockBlock, &mockEvent, mockParams)
	})

}
