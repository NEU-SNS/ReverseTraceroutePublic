package mocks

import "github.com/stretchr/testify/mock"

import dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
import "golang.org/x/net/context"

type VPService struct {
	mock.Mock
}

// GetVPs provides a mock function with given fields: _a0, _a1
func (_m *VPService) GetVPs(_a0 context.Context, _a1 *dm.VPRequest) (*dm.VPReturn, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *dm.VPReturn
	if rf, ok := ret.Get(0).(func(context.Context, *dm.VPRequest) *dm.VPReturn); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dm.VPReturn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dm.VPRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
