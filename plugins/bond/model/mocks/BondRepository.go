// Code generated by mockery 2.7.5. DO NOT EDIT.

package mocks

import (
	subscanmodel "github.com/CoolBitX-Technology/subscan/model"
	model "github.com/CoolBitX-Technology/subscan/plugins/bond/model"
	mock "github.com/stretchr/testify/mock"
)

// BondRepository is an autogenerated mock type for the BondRepository type
type BondRepository struct {
	mock.Mock
}

// GetBondListByAddr provides a mock function with given fields: page, row, addr, status, locked
func (_m *BondRepository) GetBondListByAddr(page int, row int, addr string, status string, locked int) ([]model.Bond, error) {
	ret := _m.Called(page, row, addr, status, locked)

	var r0 []model.Bond
	if rf, ok := ret.Get(0).(func(int, int, string, string, int) []model.Bond); ok {
		r0 = rf(page, row, addr, status, locked)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Bond)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int, string, string, int) error); ok {
		r1 = rf(page, row, addr, status, locked)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewBondExtrinsic provides a mock function with given fields: b, e, params, status
func (_m *BondRepository) NewBondExtrinsic(b *subscanmodel.Block, e *subscanmodel.Extrinsic, params []subscanmodel.ExtrinsicParam, status string) error {
	ret := _m.Called(b, e, params, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(*subscanmodel.Block, *subscanmodel.Extrinsic, []subscanmodel.ExtrinsicParam, string) error); ok {
		r0 = rf(b, e, params, status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
