package mocks

import "github.com/stretchr/testify/mock"

import datamodel "github.com/NEU-SNS/ReverseTraceroute/datamodel"
import context "golang.org/x/net/context"

type VPServiceServer struct {
	mock.Mock
}

// GetVPs provides a mock function with given fields: _a0, _a1
func (_m *VPServiceServer) GetVPs(_a0 context.Context, _a1 *datamodel.VPRequest) (*datamodel.VPReturn, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *datamodel.VPReturn
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.VPRequest) *datamodel.VPReturn); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.VPReturn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.VPRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
