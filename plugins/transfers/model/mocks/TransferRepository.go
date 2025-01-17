// Code generated by mockery 2.7.5. DO NOT EDIT.

package mocks

import (
	subscanmodel "github.com/CoolBitX-Technology/subscan/model"
	model "github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	mock "github.com/stretchr/testify/mock"
)

// TransferRepository is an autogenerated mock type for the TransferRepository type
type TransferRepository struct {
	mock.Mock
}

// GetExtrinsicByIndex provides a mock function with given fields: ei
func (_m *TransferRepository) GetExtrinsicByIndex(ei string) (model.Transfer, error) {
	ret := _m.Called(ei)

	var r0 model.Transfer
	if rf, ok := ret.Get(0).(func(string) model.Transfer); ok {
		r0 = rf(ei)
	} else {
		r0 = ret.Get(0).(model.Transfer)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ei)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransfersByAddr provides a mock function with given fields: page, row, addr
func (_m *TransferRepository) GetTransfersByAddr(page int, row int, addr string) ([]model.Transfer, error) {
	ret := _m.Called(page, row, addr)

	var r0 []model.Transfer
	if rf, ok := ret.Get(0).(func(int, int, string) []model.Transfer); ok {
		r0 = rf(page, row, addr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Transfer)
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

// GetTransfersList provides a mock function with given fields: page, row
func (_m *TransferRepository) GetTransfersList(page int, row int) ([]model.Transfer, int) {
	ret := _m.Called(page, row)

	var r0 []model.Transfer
	if rf, ok := ret.Get(0).(func(int, int) []model.Transfer); ok {
		r0 = rf(page, row)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Transfer)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(int, int) int); ok {
		r1 = rf(page, row)
	} else {
		r1 = ret.Get(1).(int)
	}

	return r0, r1
}

// NewTransferExtrinsic provides a mock function with given fields: b, e, params
func (_m *TransferRepository) NewTransferExtrinsic(b *subscanmodel.Block, e *subscanmodel.Extrinsic, params []subscanmodel.ExtrinsicParam) error {
	ret := _m.Called(b, e, params)

	var r0 error
	if rf, ok := ret.Get(0).(func(*subscanmodel.Block, *subscanmodel.Extrinsic, []subscanmodel.ExtrinsicParam) error); ok {
		r0 = rf(b, e, params)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
