package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

type Controller_PingServer struct {
	mock.Mock
}

// Send provides a mock function with given fields: _a0
func (_m *Controller_PingServer) Send(_a0 *datamodel.Ping) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.Ping) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
