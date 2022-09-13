package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

import context "golang.org/x/net/context"

type ControllerServer struct {
	mock.Mock
}

// Ping provides a mock function with given fields: _a0, _a1
func (_m *ControllerServer) Ping(_a0 *datamodel.PingArg, _a1 controllerapi.Controller_PingServer) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.PingArg, controllerapi.Controller_PingServer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Traceroute provides a mock function with given fields: _a0, _a1
func (_m *ControllerServer) Traceroute(_a0 *datamodel.TracerouteArg, _a1 controllerapi.Controller_TracerouteServer) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.TracerouteArg, controllerapi.Controller_TracerouteServer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetVPs provides a mock function with given fields: _a0, _a1
func (_m *ControllerServer) GetVPs(_a0 context.Context, _a1 *datamodel.VPRequest) (*datamodel.VPReturn, error) {
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

// ReceiveSpoofedProbes provides a mock function with given fields: _a0
func (_m *ControllerServer) ReceiveSpoofedProbes(_a0 controllerapi.Controller_ReceiveSpoofedProbesServer) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(controllerapi.Controller_ReceiveSpoofedProbesServer) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
