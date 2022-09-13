package mocks

import "github.com/stretchr/testify/mock"

import dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"

type DataAccess struct {
	mock.Mock
}

// GetPingBySrcDst provides a mock function with given fields: src, dst
func (_m *DataAccess) GetPingBySrcDst(src uint32, dst uint32) ([]*dm.Ping, error) {
	ret := _m.Called(src, dst)

	var r0 []*dm.Ping
	if rf, ok := ret.Get(0).(func(uint32, uint32) []*dm.Ping); ok {
		r0 = rf(src, dst)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.Ping)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, uint32) error); ok {
		r1 = rf(src, dst)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPingsMulti provides a mock function with given fields: _a0
func (_m *DataAccess) GetPingsMulti(_a0 []*dm.PingMeasurement) ([]*dm.Ping, error) {
	ret := _m.Called(_a0)

	var r0 []*dm.Ping
	if rf, ok := ret.Get(0).(func([]*dm.PingMeasurement) []*dm.Ping); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.Ping)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*dm.PingMeasurement) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorePing provides a mock function with given fields: _a0
func (_m *DataAccess) StorePing(_a0 *dm.Ping) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*dm.Ping) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetTRBySrcDst provides a mock function with given fields: _a0, _a1
func (_m *DataAccess) GetTRBySrcDst(_a0 uint32, _a1 uint32) ([]*dm.Traceroute, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*dm.Traceroute
	if rf, ok := ret.Get(0).(func(uint32, uint32) []*dm.Traceroute); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.Traceroute)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, uint32) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTraceMulti provides a mock function with given fields: _a0
func (_m *DataAccess) GetTraceMulti(_a0 []*dm.TracerouteMeasurement) ([]*dm.Traceroute, error) {
	ret := _m.Called(_a0)

	var r0 []*dm.Traceroute
	if rf, ok := ret.Get(0).(func([]*dm.TracerouteMeasurement) []*dm.Traceroute); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.Traceroute)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*dm.TracerouteMeasurement) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreTraceroute provides a mock function with given fields: _a0
func (_m *DataAccess) StoreTraceroute(_a0 *dm.Traceroute) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*dm.Traceroute) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *DataAccess) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
