package reversetraceroute

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	mocks "github.com/NEU-SNS/ReverseTraceroute/revtr/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	vpm "github.com/NEU-SNS/ReverseTraceroute/vpservice/mocks"
	vpt "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var vp = []*vpt.VantagePoint{
	&vpt.VantagePoint{
		Hostname:    "test1.fake.com",
		Ip:          1239139955,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake.com",
	},
	&vpt.VantagePoint{
		Hostname:    "test2.fake.com",
		Ip:          1239139958,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake.com",
	},
	&vpt.VantagePoint{
		Hostname:    "test1.fake1.com",
		Ip:          1239139959,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake1.com",
	},
	&vpt.VantagePoint{
		Hostname:    "test2.fake1.com",
		Ip:          1239139956,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake1.com",
	},
	&vpt.VantagePoint{
		Hostname:    "test1.fake2.com",
		Ip:          1239139960,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake2.com",
	},
	&vpt.VantagePoint{
		Hostname:    "test2.fake2.com",
		Ip:          1239139957,
		Timestamp:   true,
		RecordRoute: true,
		Spoof:       true,
		RecSpoof:    true,
		Site:        "fake2.com",
	},
}

var cs = &mocks.ClusterSource{}
var cm = clustermap.New(cs)
var mvps = &vpm.VPSource{}

func initTests() {
	cs.On("GetClusterIDByIP", mock.AnythingOfType("uint32")).Return(0, fmt.Errorf("None found"))
	mvps.On("GetVPs").Return(&vpt.VPReturn{Vps: vp}, nil)
	mvps.On("GetRRSpoofers",
		mock.AnythingOfType("uint32"),
		mock.AnythingOfType("uint32")).Return(vp, nil)
}

func TestMain(m *testing.M) {
	flag.Parse()
	initTests()
	os.Exit(m.Run())
}

type vpSourceMock struct {
}

func (vps vpSourceMock) GetOneVPPerSite() (*vpt.VPReturn, error) {
	vp := []*vpt.VantagePoint{
		&vpt.VantagePoint{
			Hostname:    "test1.fake.com",
			Ip:          1239139955,
			Timestamp:   true,
			RecordRoute: true,
			Spoof:       true,
			RecSpoof:    true,
			Site:        "fake.com",
		},
		&vpt.VantagePoint{
			Hostname:    "test2.fake1.com",
			Ip:          1239139956,
			Timestamp:   true,
			RecordRoute: true,
			Spoof:       true,
			RecSpoof:    true,
			Site:        "fake1.com",
		},
		&vpt.VantagePoint{
			Hostname:    "test2.fake2.com",
			Ip:          1239139957,
			Timestamp:   true,
			RecordRoute: true,
			Spoof:       true,
			RecSpoof:    true,
			Site:        "fake2.com",
		},
	}
	return &vpt.VPReturn{
		Vps: vp,
	}, nil
}

var myIP = "129.10.113.189"

func TestCreateReverseTraceroute(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]types.Adjacency{
		types.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)

	for _, test := range []struct {
		rm        pb.RevtrMeasurement
		expectNil bool
	}{
		{rm: pb.RevtrMeasurement{
			Src:       myIP,
			Dst:       "8.8.8.8",
			Id:        1,
			Staleness: 60,
		},
			expectNil: false},
	} {

		revtr := CreateReverseTraceroute(test.rm, cs, nil, nil, nil)
		if test.expectNil {
			if revtr != nil {
				t.Fatalf("CreateReverseTraceroute(%v): Expected[<nil>], Got[!<nil>]", test.rm)
			}
		} else {
			if revtr == nil {
				t.Fatalf("CreateReverseTraceroute(%v): Expected[!<nil>], Got[<nil>]", test.rm)
			}
		}
	}
}

func TestSymmetricAssumptions(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]types.Adjacency{
		types.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	for _, test := range []struct {
		rm       pb.RevtrMeasurement
		expected int
		add      []Segment
	}{
		{rm: pb.RevtrMeasurement{
			Src:       myIP,
			Dst:       "8.8.8.8",
			Id:        1,
			Staleness: 60,
		}, expected: 0},
	} {
		revtr := CreateReverseTraceroute(test.rm, cs, nil, nil, nil)
		if revtr == nil {
			t.Fatalf("Failed to create ReverseTraceroute")
		}
		revtr.AddSegments(test.add, cm)
		if revtr.SymmetricAssumptions() != test.expected {
			t.Fatalf("SymmetricAssumptions, Expected: %d, Got: %d", test.expected, revtr.SymmetricAssumptions())
		}
	}
}

func TestDeadends(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]types.Adjacency{
		types.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	for _, test := range []struct {
		rm     pb.RevtrMeasurement
		expect []string
	}{
		{rm: pb.RevtrMeasurement{
			Src:       myIP,
			Dst:       "8.8.8.8",
			Id:        1,
			Staleness: 60,
		}, expect: nil,
		},
	} {
		revtr := CreateReverseTraceroute(test.rm, cs, nil, nil, nil)
		if revtr == nil {
			t.Fatalf("Failed to create ReverseTraceroute")
		}
		deadends := revtr.Deadends()
		assert.Equal(t, deadends, test.expect)
	}

}

func TestRRVPSInitializedForHop(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	testIP := "192.168.1.1"
	out := revtr.rrVPSInitializedForHop(testIP)
	if out != false {
		t.Fatalf("Expected false, got true")
	}
	revtr.setRRVPSForHop(testIP, []string{"test", "test1"})
	out = revtr.rrVPSInitializedForHop(testIP)
	if out != true {
		t.Fatalf("Expected true, got false")
	}
}

func TestCurrPath(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	expected := NewReversePath(myIP, "8.8.8.8", nil)
	currPath := revtr.CurrPath()
	if currPath.Dst != expected.Dst || currPath.Src != expected.Src {
		t.Fatalf("Src or dst not equal")
	}
}

func TestAddSegments(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)

	for _, test := range []struct {
		rm       pb.RevtrMeasurement
		add      []Segment
		expected bool
	}{
		{rm: pb.RevtrMeasurement{
			Src:       myIP,
			Dst:       "8.8.8.8",
			Id:        1,
			Staleness: 60,
		}, add: []Segment{NewTrtoSrcRevSegment([]string{"8.8.8.8", myIP}, "8.8.8.8", myIP)},
			expected: true,
		},
	} {
		revtr := CreateReverseTraceroute(test.rm, cs, nil, nil, nil)
		if revtr == nil {
			t.Fatalf("Failed to create ReverseTraceroute")
		}
		added := revtr.AddSegments(test.add, cm)
		if added != test.expected {
			t.Fatalf("Failed to added segments: Got[%v] Expected[%v]", added, test.expected)
		}
	}
}

func TestReaches(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if revtr.Reaches(cm) {
		t.Fatal("Reaches, reaches on creation of ReverseTraceroute")
	}
	segs := []Segment{NewTrtoSrcRevSegment([]string{"8.8.8.8", "10.0.0.2", myIP}, myIP, "8.8.8.8")}
	added := revtr.AddSegments(segs, cm)
	if !added {
		t.Fatal("Failed to add Segments")
	}
	reaches := revtr.Reaches(cm)
	if !reaches {
		t.Fatal("Failed to reach after adding reaching segment LastHop: ", revtr.LastHop(), " Got ", revtr.CurrPath().LastSeg())
	}
	t.Log(revtr.CurrPath())
}

var adjs = []types.Adjacency{
	types.Adjacency{
		IP1: 167772161,
		IP2: 167772162,
		Cnt: 2,
	},
	types.Adjacency{
		IP1: 167772161,
		IP2: 167772163,
		Cnt: 4,
	},
	types.Adjacency{
		IP1: 167772161,
		IP2: 167772164,
		Cnt: 3,
	},
	types.Adjacency{
		IP1: 167772165,
		IP2: 167772161,
		Cnt: 2,
	},
	types.Adjacency{
		IP1: 167772166,
		IP2: 134744072,
		Cnt: 9,
	},
	types.Adjacency{
		IP1: 167772167,
		IP2: 167772161,
		Cnt: 8,
	},
}

var adjstodst = []types.AdjacencyToDest{
	types.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772166,
		Adjacent: 167772161,
		Cnt:      7,
	},
	types.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772167,
		Adjacent: 167772161,
		Cnt:      6,
	},
	types.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  134744072,
		Adjacent: 167772162,
		Cnt:      3,
	},
	types.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772161,
		Adjacent: 167772164,
		Cnt:      2,
	},
}

func TestInitializeRRVPs(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	testIP := "8.8.8.8"
	err := revtr.InitializeRRVPs(testIP, mvps)
	if err != nil {
		t.Fatalf("Failed to initialize RRVPs Expected[<nil>], Got[%v]", err)
	}
	if rl, ok := revtr.RRHop2RateLimit[testIP]; ok {
		if rl != RateLimit {
			t.Fatalf("Failed to init RR Rate limit, Epected %d, Got %d", RateLimit, rl)
		}
	} else {
		t.Fatalf("Failed to initalize for %s", testIP)
	}
	if ss, ok := revtr.RRHop2VPSLeft[testIP]; ok {
		if len(ss) < 1 {
			t.Fatalf("Failed to initialize RR vps, Got: %v", ss)
		}
		t.Log(ss)
	} else {
		t.Fatalf("Failed to initalize for %s, no vps in map", testIP)
	}
}

func TestGetRRVPs(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	vps, target := revtr.GetRRVPs("8.8.8.8", mvps)
	if vps, ok := revtr.RRHop2VPSLeft["8.8.8.8"]; ok {
		t.Log(vps)
	} else {
		t.Fatal("VP not initialized")
	}
	if vps[0] != "non_spoofed" || target != "8.8.8.8" {
		t.Fatal("Failed to get non_spoofed on first call")
	}
	vps, target = revtr.GetRRVPs("8.8.8.8", mvps)
	if vps[0] == "non_spoofed" || target != "8.8.8.8" {
		t.Fatal("Failed, got non_spoofed on second call")
	}
}

func TestInitializeTSAdjacents(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	err := revtr.InitializeTSAdjacents("8.8.8.8", as)
	if err != nil {
		log.Fatal(err)
	}
	if adjs, ok := revtr.TSHop2AdjsLeft["8.8.8.8"]; !ok {
		log.Fatal("Failed to init TS Adjs ", adjs)
	}
}

func TestGetTSAdjacents(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	adjs := revtr.GetTSAdjacents("8.8.8.8", as)
	if len(adjs) == 0 {
		t.Fatalf("Got no Adjs %d", len(adjs))
	}
	t.Log(adjs)
}

func TestLength(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	length := revtr.CurrPath().Length()
	if length == 0 {
		t.Fatalf("Got len: %d, Expected: %d", length, 1)
	}
	t.Log(length)
}

func TestPop(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	length := revtr.CurrPath().len()
	revtr.CurrPath().Pop()
	if length != revtr.CurrPath().len()+1 {
		t.Fatal("Failed to pop an item")
	}
	t.Log(length)
}

func TestHops(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	hops := revtr.Hops()
	if len(hops) == 0 {
		t.Fatal("Hops failed, expected 1, got 0")
	}
}

func TestFailed(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if revtr.Failed() {
		t.Fatal("Brand new revtr Failed")
	}
}

func TestFailCurrPath(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	hop := revtr.LastHop()
	revtr.FailCurrPath()
	if !revtr.DeadEnd[hop] {
		t.Fatal("LastHop not added to deadends")
	}
}

func TestAddAndReplaceSegment(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	startLen := revtr.len()
	seg := NewTrtoSrcRevSegment([]string{"8.8.8.8", "10.0.0.1", "10.0.0.2"}, "8.8.8.8", myIP)
	res := revtr.AddAndReplaceSegment(seg)
	if !res {
		t.Fatalf("Failed to add Seg")
	}
	endLen := revtr.len()
	if startLen >= endLen {
		t.Fatal("Failed to add path")
	}
	if revtr.CurrPath().len() != 1 {
		t.Fatal("Failed replacement ", revtr.CurrPath().len())
	}
}

func TestAddBackgroundTRSegment(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := pb.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, cs, nil, nil, nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	seg1 := NewRRRevSegment([]string{"8.8.8.8", "10.0.0.1"}, myIP, "8.8.8.8")
	seg2 := NewRRRevSegment([]string{"10.0.0.1", "10.0.0.4"}, myIP, "8.8.8.8")
	revtr.AddSegments([]Segment{seg1, seg2}, cm)
	seg := NewTrtoSrcRevSegment([]string{"10.0.0.1", "10.0.0.2", myIP}, myIP, "10.0.0.1")
	res := revtr.AddBackgroundTRSegment(seg, cm)
	if !res {
		t.Fatal("Failed to add background tr seg")
	}
	if !revtr.Reaches(cm) {
		t.Fatal("Adding reaching trsegment didn't reach")
	}
	t.Log(revtr.CurrPath())
}
