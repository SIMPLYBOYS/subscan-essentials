// Code generated by mockery 2.7.5. DO NOT EDIT.

package mocks

import (
	model "github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	mock "github.com/stretchr/testify/mock"
)

// TransferDelivery is an autogenerated mock type for the TransferDelivery type
type TransferDelivery struct {
	mock.Mock
}

// TransferList provides a mock function with given fields: page, row, address
func (_m *TransferDelivery) TransferList(page int, row int, address string) ([]model.Transfer, error) {
	ret := _m.Called(page, row, address)

	var r0 []model.Transfer
	if rf, ok := ret.Get(0).(func(int, int, string) []model.Transfer); ok {
		r0 = rf(page, row, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Transfer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int, string) error); ok {
		r1 = rf(page, row, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
