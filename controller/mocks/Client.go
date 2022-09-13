package mocks

import "github.com/stretchr/testify/mock"

import "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
import "github.com/NEU-SNS/ReverseTraceroute/datamodel"
import "golang.org/x/net/context"

type Client struct {
	mock.Mock
}

// Ping provides a mock function with given fields: _a0, _a1
func (_m *Client) Ping(_a0 context.Context, _a1 *datamodel.PingArg) (controllerapi.Controller_PingClient, error) {
	ret := _m.Called(_a0, _a1)

	var r0 controllerapi.Controller_PingClient
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.PingArg) controllerapi.Controller_PingClient); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_PingClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.PingArg) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Traceroute provides a mock function with given fields: _a0, _a1
func (_m *Client) Traceroute(_a0 context.Context, _a1 *datamodel.TracerouteArg) (controllerapi.Controller_TracerouteClient, error) {
	ret := _m.Called(_a0, _a1)

	var r0 controllerapi.Controller_TracerouteClient
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.TracerouteArg) controllerapi.Controller_TracerouteClient); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_TracerouteClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.TracerouteArg) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVps provides a mock function with given fields: _a0, _a1
func (_m *Client) GetVps(_a0 context.Context, _a1 *datamodel.VPRequest) (*datamodel.VPReturn, error) {
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
func (_m *Client) ReceiveSpoofedProbes(_a0 context.Context) (controllerapi.Controller_ReceiveSpoofedProbesClient, error) {
	ret := _m.Called(_a0)

	var r0 controllerapi.Controller_ReceiveSpoofedProbesClient
	if rf, ok := ret.Get(0).(func(context.Context) controllerapi.Controller_ReceiveSpoofedProbesClient); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_ReceiveSpoofedProbesClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
