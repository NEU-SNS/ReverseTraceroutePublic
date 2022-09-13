package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

type Controller_ReceiveSpoofedProbesServer struct {
	mock.Mock
}

// SendAndClose provides a mock function with given fields: _a0
func (_m *Controller_ReceiveSpoofedProbesServer) SendAndClose(_a0 *datamodel.ReceiveSpoofedProbesResponse) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.ReceiveSpoofedProbesResponse) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Recv provides a mock function with given fields:
func (_m *Controller_ReceiveSpoofedProbesServer) Recv() (*datamodel.Probe, error) {
	ret := _m.Called()

	var r0 *datamodel.Probe
	if rf, ok := ret.Get(0).(func() *datamodel.Probe); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.Probe)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
