package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

type Controller_PingClient struct {
	mock.Mock
}

// Recv provides a mock function with given fields:
func (_m *Controller_PingClient) Recv() (*datamodel.Ping, error) {
	ret := _m.Called()

	var r0 *datamodel.Ping
	if rf, ok := ret.Get(0).(func() *datamodel.Ping); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.Ping)
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
