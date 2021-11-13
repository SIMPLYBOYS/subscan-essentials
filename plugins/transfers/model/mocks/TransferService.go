// Code generated by mockery 2.7.5. DO NOT EDIT.

package mocks

import (
	model "github.com/CoolBitX-Technology/subscan/model"
	mock "github.com/stretchr/testify/mock"

	transfersmodel "github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
)

// TransferService is an autogenerated mock type for the TransferService type
type TransferService struct {
	mock.Mock
}

// BalancesTransaction provides a mock function with given fields: b, e, params
func (_m *TransferService) BalancesTransaction(b *model.Block, e *model.Extrinsic, params []model.ExtrinsicParam) error {
	ret := _m.Called(b, e, params)

	var r0 error
	if rf, ok := ret.Get(0).(func(*model.Block, *model.Extrinsic, []model.ExtrinsicParam) error); ok {
		r0 = rf(b, e, params)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetTransfersListJson provides a mock function with given fields: page, row, addr
func (_m *TransferService) GetTransfersListJson(page int, row int, addr string) ([]transfersmodel.Transfer, error) {
	ret := _m.Called(page, row, addr)

	var r0 []transfersmodel.Transfer
	if rf, ok := ret.Get(0).(func(int, int, string) []transfersmodel.Transfer); ok {
		r0 = rf(page, row, addr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]transfersmodel.Transfer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int, string) error); ok {
		r1 = rf(page, row, addr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}