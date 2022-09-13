package mocks

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/types"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"
)

type TRStore struct {
	mock.Mock
}

func (_m *TRStore) MonitorDB() () {
	
}
	
func (_m *TRStore) StoreAtlasTracerouteBulk([]*datamodel.Traceroute) ([]int64, error) {
	return nil, nil
}

func (_m *TRStore) StoreTrRRHop([]types.TrIDRRHop) error {
	return nil 
}

func (_m *TRStore) CheckIntersectingPath(int64, uint32) (*pb.Path, error) {
	return nil, nil 
}

func (_m *TRStore) MarkTracerouteStale(uint32, int64, int64) error {
	return nil
}

func (_m *TRStore) MarkTraceroutesStale([]int64) error {
	return nil
}

func (_m *TRStore) MarkTracerouteStaleSource(uint32) error {
	return nil
}

func (_m *TRStore) CleanAtlas(uint32) error {
	return nil
}

func (_m *TRStore) RunAtlas(uint32) error {
	return nil
}



func (_m *TRStore) CompareOldAndNewTraceroute(uint32, int64, int64) (bool, bool, error) {
	return false, false, nil
}
func (_m *TRStore) UpdateRRIntersectionTracerouteNonStale(oldID int64, newID int64) error {
	return nil 
}

func (_m *TRStore) GetIPAliases([]uint32) (map[uint32]int64, map[int64][]uint32,  error) {
	return nil, nil , nil
}
// FindIntersectingTraceroute provides a mock function with given fields: _a0
func (_m *TRStore) FindIntersectingTraceroute(_a0 types.IntersectionQuery) (types.IntersectionResponse, error) {
	ret := _m.Called(_a0)

	var r0 *pb.Path
	if rf, ok := ret.Get(0).(func(types.IntersectionQuery) *pb.Path); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Path)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(types.IntersectionQuery) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return types.IntersectionResponse{Path: r0}, r1
}

// StoreAtlasTraceroute provides a mock function with given fields: _a0
func (_m *TRStore) StoreAtlasTraceroute(_a0 *datamodel.Traceroute) (int64,  error) {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.Traceroute) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return 0, r0
}

// GetAtlasSources provides a mock function with given fields: _a0, _a1
func (_m *TRStore) GetAtlasSources(_a0 uint32, _a1 time.Duration) ([]uint32, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []uint32
	if rf, ok := ret.Get(0).(func(uint32, time.Duration) []uint32); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]uint32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, time.Duration) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *TRStore) GetAvailableHopAtlasPerSource() (map[uint32]map[uint32]struct{}, error) {
	return nil, nil 
}