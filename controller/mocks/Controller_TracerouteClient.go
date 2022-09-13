package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

type Controller_TracerouteClient struct {
	mock.Mock
}

// Recv provides a mock function with given fields:
func (_m *Controller_TracerouteClient) Recv() (*datamodel.Traceroute, error) {
	ret := _m.Called()

	var r0 *datamodel.Traceroute
	if rf, ok := ret.Get(0).(func() *datamodel.Traceroute); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.Traceroute)
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
