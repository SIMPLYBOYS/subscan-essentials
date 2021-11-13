// Code generated by mockery 2.7.5. DO NOT EDIT.

package mocks

import (
	model "github.com/CoolBitX-Technology/subscan/plugins/reward/model"
	mock "github.com/stretchr/testify/mock"
)

// RewardDelivery is an autogenerated mock type for the RewardDelivery type
type RewardDelivery struct {
	mock.Mock
}

// RewardList provides a mock function with given fields: page, row, address
func (_m *RewardDelivery) RewardList(page int, row int, address string) ([]model.Reward, int, error) {
	ret := _m.Called(page, row, address)

	var r0 []model.Reward
	if rf, ok := ret.Get(0).(func(int, int, string) []model.Reward); ok {
		r0 = rf(page, row, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Reward)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(int, int, string) int); ok {
		r1 = rf(page, row, address)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int, string) error); ok {
		r2 = rf(page, row, address)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
