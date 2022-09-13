package reversetraceroute

import (
	"errors"

	"github.com/NEU-SNS/ReverseTraceroute/environment"
	// "context"
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"

	// "strconv"
	"sync"
	"time"

	stringutil "github.com/NEU-SNS/ReverseTraceroute/util/string"
	"github.com/golang/protobuf/ptypes"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	apb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"

	// vpservicepb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	rs "github.com/NEU-SNS/ReverseTraceroute/rankingservice/client"
	rspb "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	networkutil "github.com/NEU-SNS/ReverseTraceroute/util"
)

var (
	dstMustBeReachable = false
	batchInitRRVPs     = true
	maxUnresponsive    = 10
	rrVPsByCluster     bool
	tsAdjsByCluster    bool
)

// StopReason is the reason a reverse traceroute stopped
type StopReason string

const (
	// Failed is the StopReason when no technique could add a hop
	// or an unrecoverable error occured
	Failed StopReason = "FAILED"
	// Trivial is the StopReason when a revtr was trivial to accomplish
	Trivial StopReason = "TRIVIAL"
	// Canceled is the StopReason when a revtr is canceled
	Canceled StopReason = "CANCELED"
	// Reaches is the StopReason when a revtr has reached its destination
	Reaches StopReason = "REACHES"

	// Non spoofed serves to first test a non spoofed RR before a spoofed RR
	NON_SPOOFED string = "non_spoofed"
	
	// Max ingress vps is the maximum number of VPs to be tested for an ingress
	MAX_INGRESS_VPS = 5

)

// ReverseTraceroute is a reverse traceroute
// Paths is a stack of ReversePaths
// DeadEnd is a has of ip -> bool of paths (stored as lasthop) we know don't work
// and shouldn't be tried again
//		rrhop2ratelimit is from cluster to max number of probes to send to it in a batch
//		rrhop2vpsleft is from cluster to the VPs that haven't probed it yet,
//		in prioritized order
// tshop2ratelimit is max number of adjacents to probe for at once
// tshop2adjsleft is from cluster to the set of adjacents to try in prioritized order
// [] means we've tried them all. if it is missing the key, that means we still need to
// initialize it
type ReverseTraceroute struct {
	ID                      uint32
	LogStr                  string
	Stats                   Stats
	Paths                   *[]*ReversePath
	DeadEnd                 map[string]bool
	RRHop2RateLimit         map[string]int
	RRHop2VPSLeft           map[string][]rspb.Source // Map of remaining spoofers per hop per ingress
	TSHop2RateLimit         map[string]int
	TSHop2AdjsLeft          map[string][]string
	Src, Dst                string
	StopReason              StopReason
	StartTime, EndTime      time.Time
	BackoffEndhost          bool
	FailReason              string
	Label                   string // a label for evaluation purposes
	mu                      sync.Mutex // protects running
	hnCacheInit             bool
	hostnameCache           map[string]string
	rttCache                map[string]float32
	Tokens                  []*apb.IntersectionResponse
	TSDstToStampsZero       map[string]bool
	TSSrcToHopToSendSpoofed map[string]map[string]bool
	// whether this hop is thought to be responsive at all to this src
	// Since I can't intialize to true, I'm going to use an int and say 0 is true
	// anythign else will be false
	tsHopResponsive         map[string]int
	ErrorDetails            bytes.Buffer
	rrSpoofRRResponsive     map[string]int
	OnAdd                   OnAddFunc
	OnReach                 OnReachFunc
	OnFail                  OnFailFunc
	OnReachOnce, OnFailOnce sync.Once
	// RRHop2IssuedProbes		map[string][]int64 // NEW: list of , used for rankingservice API calls.
	RRHop2VPSIngress        map[string]map[string] string // Ingress per VP per hop
	FailedIngress           map[string] map[string] int // Keep track of which ingress has been tried for each hop
	RRVPSelectionTechnique  environment.RRVPsSelectionTechnique // Algorithm selection for performing spoofed RR 
	RankedSpoofersByHop 	map[uint32]map[uint32]*RankedSpooferMeasurement // Keep track of which vps were used  for the evaluation
	MaxSpoofers             uint32
	TracerouteID            int64 // If we intersected with a traceroute, keep track of the corresponding traceroute ID.
	IsRunForwardTraceroute  bool
	ForwardTracerouteID     int64 // If we run a forward traceroute, keep track of the corresponding traceroute. 
	IsRunRTTPings           bool 
	RTTByHop				map[uint32]RTTMeasurement					
	// Options of the atlas
	AtlasOpts  			    AtlasOptions
	// Options for checking symmetry
	SymmetryOpts            SymmetryOptions
	// Options for checking destination based routing
	CheckDBasedRoutingOpts         CheckDestinationBasedRoutingOptions
	DestinationBasedRoutingByHop   map[string] types.DestinationBasedRoutingViolation // Keep track of which hops violates destination based routing and cant be used as destination
	NextHopsegmentsbyHop           map[string] Segment // A segment containing a next hop by hop to allow fast check of destination based routing violaton.
	RRHeuristicsOpt         RRHeuristicsOptions
	// Whether to use timestamping or not. 
	UseTimestamp bool 
	// Whether to use cache or not.
	UseCache bool 
}

type RRHeuristicsOptions struct {
	UseDoubleStamp bool 
}

type AtlasOptions struct {
	UseAtlas bool
	UseRRPings bool 
	IgnoreSource bool
	IgnoreSourceAS bool 
	Staleness int64 // In minutes 
	StalenessBeforeRefresh int64 // In minutes.  
	Platforms  [] string // Platforms that are authorized to intersect traceroutes.
	
}

type SymmetryOptions struct {
	IsAllowInterdomainSymmetry bool
}

type CheckDestinationBasedRoutingOptions struct {
	CheckTunnel bool 
}

func (ao *AtlasOptions) ToString() string {
	aoStr := fmt.Sprintf("%t_%t_%t", ao.UseAtlas, ao.IgnoreSource, ao.UseAtlas)
	return aoStr
}

// Stats are the stats collected
// while running a reverse traceroute
type Stats struct {
	RRProbes                   int
	SpoofedRRProbes            int
	TSProbes                   int
	SpoofedTSProbes            int
	DestBasedRoutRRPobes       int
	DestBasedRoutSpoofedRRPobes       int
	RRRoundCount               int
	RRDuration                 time.Duration
	TSRoundCount               int
	TSDuration                 time.Duration
	TRToSrcRoundCount          int
	TRToSrcDuration            time.Duration
	AssumeSymmetricRoundCount  int
	AssumeSymmetricDuration    time.Duration
	BackgroundTRSRoundCount    int
	BackgroundTRSDuration      time.Duration
	CheckDestinationBasedRoutingRoundCount int
	CheckDestinationBasedRoutingDuration  time.Duration
}

// RankedSpooferMeasurement keeps track of the spoofed measurement and their results
type RankedSpooferMeasurement struct{
	Rank uint32
	IP   uint32
	MeasurementID int64 // ID of the measurement relative to this
	// SameIngress bool // Whether the ingress was the same of not
	// Ingress uint32 // The ingress of the spoofed RR
	// UncoveredHops uint32 // The number of uncovered reverse hops
	// PredictedDistanceIngress uint32 // The predicted distance to the ingress
	// DistanceIngress uint32 // Distance to the ingress
}

type RTTMeasurement struct {
	RTT uint32
	MeasurementID int64 // ID of the ping measuremennt related to this
}
// OnAddFunc is the type of the callback that can be called
// when a segment is added to the ReverseTraceroute
type OnAddFunc func(*ReverseTraceroute)

// OnReachFunc is called when a reverse traceroute reaches its dst
type OnReachFunc func(*ReverseTraceroute)

// OnFailFunc is called when a reverse traceroute fails
type OnFailFunc func(*ReverseTraceroute)

var initOnce sync.Once

// NewReverseTraceroute creates a new reverse traceroute
func NewReverseTraceroute(src, dst, label string, id uint32, 
	rrVPSelectionTechnique environment.RRVPsSelectionTechnique, maxSpoofers uint32, useTimestamp bool, useCache bool,  
	atlasOptions AtlasOptions, symmetryOptions SymmetryOptions, rrHeuristicsOptions RRHeuristicsOptions,
	checkDestBasedRoutingOptions CheckDestinationBasedRoutingOptions,
	isRunForwardTraceroute bool,
	isRunRTTPings bool ) *ReverseTraceroute {
	if id == 0 {
		id = rand.Uint32()
	}
	ret := ReverseTraceroute{
		ID:                      id,
		LogStr:                  fmt.Sprintf("ID: %d :", id),
		Src:                     src,
		Dst:                     dst,
		Label:                   label,
		Paths:                   &[]*ReversePath{NewReversePath(src, dst, nil)},
		DeadEnd:                 make(map[string]bool),
		tsHopResponsive:         make(map[string]int),
		TSDstToStampsZero:       make(map[string]bool),
		TSSrcToHopToSendSpoofed: make(map[string]map[string]bool),
		RRHop2RateLimit:         make(map[string]int),
		RRHop2VPSLeft:           make(map[string][]rspb.Source),
		TSHop2RateLimit:         make(map[string]int),
		TSHop2AdjsLeft:          make(map[string][]string),
		StartTime:               time.Now(),
		rrSpoofRRResponsive:     make(map[string]int),
		// RRHop2IssuedProbes: 	 make(map[string][]int64),
		RRVPSelectionTechnique:  rrVPSelectionTechnique,
		RRHop2VPSIngress:        make(map[string]map[string]string),
		RankedSpoofersByHop:     make(map[uint32]map[uint32]*RankedSpooferMeasurement),
		RTTByHop:                make(map[uint32]RTTMeasurement),
		MaxSpoofers:             maxSpoofers, // Upper bound of the number of spoofer sites 
		FailedIngress:           make(map[string]map[string]int),
		AtlasOpts: atlasOptions,
		SymmetryOpts: symmetryOptions,
		RRHeuristicsOpt: rrHeuristicsOptions,
		CheckDBasedRoutingOpts: checkDestBasedRoutingOptions,
		DestinationBasedRoutingByHop: make(map[string]types.DestinationBasedRoutingViolation),
		NextHopsegmentsbyHop:    make(map[string]Segment),
		IsRunForwardTraceroute:  isRunForwardTraceroute,
		IsRunRTTPings: isRunRTTPings,
		UseTimestamp : useTimestamp,
		UseCache: useCache, 

	}
	return &ret
}

func logRevtr(revtr *ReverseTraceroute) log.Logger {
	return log.WithFieldDepth(log.Fields{
		"Revtr": revtr.LogStr,
	}, 1)
}

// func (rt *ReverseTraceroute) AddMeasurementsIDs(hop string, measurementIDs []int64){
// 	for _, mID := range measurementIDs {
// 		rt.RRHop2IssuedProbes[hop] = append(rt.RRHop2IssuedProbes[hop], mID)
// 	}
// }

// func (rt *ReverseTraceroute) AddMeasurementsID(hop string, measurementID int64){
// 	rt.RRHop2IssuedProbes[hop] = append(rt.RRHop2IssuedProbes[hop], measurementID)
// }

// TSSetUnresponsive sets the dst as unresponsive to ts probes
func (rt *ReverseTraceroute) TSSetUnresponsive(dst string) {
	logRevtr(rt).Debug("Setting ", dst, " unresponsive")
	rt.tsHopResponsive[dst] = 1
}

// TSIsResponsive checks if the dst is responsive to ts probes
func (rt *ReverseTraceroute) TSIsResponsive(dst string) bool {
	return rt.tsHopResponsive[dst] == 0
}

// AddUnresponsiveRRSpoofer marks target dst as unresponsive.
// if the pair already exists the value is incremented by cnt.
// if  the value is already marked Responsive, this will noop
func (rt *ReverseTraceroute) AddUnresponsiveRRSpoofer(dst string, cnt int) {
	if _, ok := rt.rrSpoofRRResponsive[dst]; ok {
		if rt.rrSpoofRRResponsive[dst] == -1 {
			return
		}
		rt.rrSpoofRRResponsive[dst] = rt.rrSpoofRRResponsive[dst] + cnt
		return
	}
	rt.rrSpoofRRResponsive[dst] = cnt
}

// MarkResponsiveRRSpoofer makes the dst as responsive
func (rt *ReverseTraceroute) MarkResponsiveRRSpoofer(dst string) {
	rt.rrSpoofRRResponsive[dst] = -1
}

func (rt *ReverseTraceroute) len() int {
	return len(*(rt.Paths))
}

// SymmetricAssumptions returns the number of symmetric
// assumptions of the last path
func (rt *ReverseTraceroute) SymmetricAssumptions() int {
	return (*rt.Paths)[rt.len()-1].SymmetricAssumptions()
}

// Deadends returns the ips of all the deadends
func (rt *ReverseTraceroute) Deadends() []string {
	var keys []string
	for k := range rt.DeadEnd {
		keys = append(keys, k)
	}
	return keys
}

func (rt *ReverseTraceroute) rrVPSInitializedForHop(hop string) bool {
	_, ok := rt.RRHop2VPSLeft[hop]
	return ok
}

func (rt *ReverseTraceroute) setRRVPSForHop(hop string, vps []rspb.Source) {
	rt.RRHop2VPSLeft[hop] = vps
}

// CurrPath gets the last Path in the Paths "stack"
func (rt *ReverseTraceroute) CurrPath() *ReversePath {
	return (*rt.Paths)[rt.len()-1]
}

// Hops gets the hops from the last path
func (rt *ReverseTraceroute) Hops() []string {
	if rt.len() == 0 {
		return []string{}
	}
	return (*rt.Paths)[rt.len()-1].Hops()
}

// LastHop gets the last hop from the last path
func (rt *ReverseTraceroute) LastHop() string {
	return (*rt.Paths)[rt.len()-1].LastHop()
}

// Reaches checks if the last path reaches
func (rt *ReverseTraceroute) Reaches(cm clustermap.ClusterMap) bool {
	// Assume that any path reaches if and only if the last one reaches
	if len(*rt.Paths) == 0 {
		return false
	}
	reach := (*rt.Paths)[rt.len()-1].Reaches(cm)
	if reach {
		rt.EndTime = time.Now()
		rt.StopReason = Reaches
	}
	if reach && rt.OnReach != nil {
		rt.OnReachOnce.Do(func() {
			logRevtr(rt).Debug("Calling onReach")
			rt.OnReach(rt)
		})
	}
	return reach
}

// Failed returns whether we have any options lefts to explore
func (rt *ReverseTraceroute) Failed() bool {
	failed := rt.len() == 0 || (rt.BackoffEndhost && rt.len() == 1 &&
		(*rt.Paths)[0].len() == 1 && reflect.TypeOf((*(*rt.Paths)[0].Path)[0]) == reflect.TypeOf(&DstRevSegment{}))
	if failed {
		rt.EndTime = time.Now()
		rt.StopReason = Failed
	}
	if failed && rt.OnFail != nil {
		rt.OnFailOnce.Do(func() {
			rt.OnFail(rt)
		})
	}
	return failed
}

// FailCurrPath fails the current path
func (rt *ReverseTraceroute) FailCurrPath() {
	rt.DeadEnd[rt.LastHop()] = true
	// keep popping until we find something that is either on a path
	// we are assuming symmetric (we know it started at src so goes to whole way)
	// or is not known to be a deadend
	for !rt.Failed() && rt.DeadEnd[rt.LastHop()] && reflect.TypeOf(rt.CurrPath().LastSeg()) !=
		reflect.TypeOf(&DstSymRevSegment{}) {
		// Pop
		*rt.Paths = (*rt.Paths)[:rt.len()-1]
	}
	// When we cail the curr path we'll do an onadd to print the curr path
	if rt.OnAdd != nil {
		rt.OnAdd(rt)
	}
}

// AddAndReplaceSegment adds a new path, equal to the current one but with the last
// segment replaced by the new one
// returns a bool of whether it was added
// might not be added if it is a deadend
func (rt *ReverseTraceroute) AddAndReplaceSegment(s Segment) bool {
	if rt.DeadEnd[s.LastHop()] {
		return false
	}
	basePath := rt.CurrPath().Clone()
	basePath.Pop()
	basePath.Add(s)
	*rt.Paths = append(*rt.Paths, basePath)
	if rt.OnAdd != nil {
		rt.OnAdd(rt)
	}
	return true
}

/*
TODO
I'm not entirely sure that this sort will match  the ruby one
It will need to be tested and verified
*/
type magicSort struct {
	s  []Segment
	cm clustermap.ClusterMap
}

func (ms magicSort) Len() int           { return len(ms.s) }
func (ms magicSort) Swap(i, j int)      { ms.s[i], ms.s[j] = ms.s[j], ms.s[i] }
func (ms magicSort) Less(i, j int) bool { return ms.s[i].Order(ms.s[j], ms.cm) < 0 }

// AddSegments returns a bool of whether any were added
// might not be added if they are deadends
// or if all hops would cause loops
func (rt *ReverseTraceroute) AddSegments(segs []Segment, cm clustermap.ClusterMap) bool {
	var added bool
	// sort based on the magic compare
	// or how long the path is?
	sort.Sort(magicSort{s: segs, cm: cm})
	basePath := rt.CurrPath().Clone()
	for _, s := range segs {
		if !rt.DeadEnd[s.LastHop()] {
			// update the structure for destination based routing checks
			// all the hops except the last one
			for _, hop := range(s.Hops()[:len(s.Hops())-1]) {
				if _, ok := rt.NextHopsegmentsbyHop[hop]; !ok {
					rt.NextHopsegmentsbyHop[hop] = s
				}
			}
			// add loop removal here
			// if we intersected a traceroute, do not remove loops as we want to keep the intersection first
			sType := s.Type()
			if pb.RevtrHopType(sType) != pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT {
				err := s.RemoveHops(basePath.Hops(), cm)
				if err != nil {
					logRevtr(rt).Error(err)
					return false
				}
			}
			if s.Length(false) == 0 {
				logRevtr(rt).Debug("Skipping loop-causing segment ", s)
				continue
			}
			added = true
			cl := basePath.Clone()
			cl.Add(s)
			*rt.Paths = append(*rt.Paths, cl)
		}
	}
	
	if added && rt.OnAdd != nil {
		// Print result stuff
		rt.OnAdd(rt)
	}
	return added
}

// ToStorable returns a storble form of a ReverseTraceroute
func (rt *ReverseTraceroute) ToStorable(atlasClient at.Atlas) pb.ReverseTraceroute {
	var ret pb.ReverseTraceroute
	ret.Id = rt.ID
	ret.Src = rt.Src
	ret.Dst = rt.Dst
	ret.FailReason = rt.FailReason
	ret.Runtime = rt.EndTime.Sub(rt.StartTime).Nanoseconds()
	ret.StartTime = rt.StartTime.Unix()
	ret.StopReason = string(rt.StopReason)
	ret.Label = rt.Label
	ret.ForwardTracerouteId = rt.ForwardTracerouteID
	stats := pb.Stats{
		TsDuration:                ptypes.DurationProto(rt.Stats.TSDuration),
		RrDuration:                ptypes.DurationProto(rt.Stats.RRDuration),
		TrToSrcDuration:           ptypes.DurationProto(rt.Stats.TRToSrcDuration),
		AssumeSymmetricDuration:   ptypes.DurationProto(rt.Stats.AssumeSymmetricDuration),
		BackgroundTrsDuration:     ptypes.DurationProto(rt.Stats.BackgroundTRSDuration),
		RrProbes:                  int32(rt.Stats.RRProbes),
		SpoofedRrProbes:           int32(rt.Stats.SpoofedRRProbes),
		DestBasedCheckRrProbes:    int32(rt.Stats.DestBasedRoutRRPobes),
		DestBasedCheckSpoofedRrProbes: int32(rt.Stats.DestBasedRoutSpoofedRRPobes),
		TsProbes:                  int32(rt.Stats.TSProbes),
		SpoofedTsProbes:           int32(rt.Stats.SpoofedTSProbes),
		RrRoundCount:              int32(rt.Stats.RRRoundCount),
		TsRoundCount:              int32(rt.Stats.TSRoundCount),
		TrToSrcRoundCount:         int32(rt.Stats.TRToSrcRoundCount),
		AssumeSymmetricRoundCount: int32(rt.Stats.AssumeSymmetricRoundCount),
		BackgroundTrsRoundCount:   int32(rt.Stats.BackgroundTRSRoundCount),
		DestBasedCheckDuration:    ptypes.DurationProto(rt.Stats.CheckDestinationBasedRoutingDuration),
		DestBasedCheckRoundCount:  int32(rt.Stats.CheckDestinationBasedRoutingRoundCount),
		
	}
	ret.Stats = &stats
	rankedSpoofersByHop := make(map[uint32]*pb.RankedSpoofers)
	for hop, rankedSpoofers := range rt.RankedSpoofersByHop {
		rankedSpoofersPB := pb.RankedSpoofers{
			RankedSpoofers: []*pb.RankedSpoofer{},
		}
		for _, rankedSpoofer := range rankedSpoofers {
			rankedSpoofersPB.RankedSpoofers = append(rankedSpoofersPB.RankedSpoofers, 
				&pb.RankedSpoofer{
					Rank: rankedSpoofer.Rank,
					Ip:   rankedSpoofer.IP,
					MeasurementId: rankedSpoofer.MeasurementID,
					RankingTechnique: string(rt.RRVPSelectionTechnique), 
				},
			)
		}
		rankedSpoofersByHop[hop] = &rankedSpoofersPB
	}
	ret.RankedSpoofersByHop = rankedSpoofersByHop

	if rt.StopReason != "" {
		ret.Status = pb.RevtrStatus_COMPLETED
	} else {
		ret.Status = pb.RevtrStatus_RUNNING
	}
	hopsSeen := make(map[string]bool)
	if !rt.Failed() && rt.FailReason == "" {
		for _, s := range *rt.CurrPath().Path {
			ty := s.Type()
			hops := s.Hops()
			sType := pb.RevtrHopType(ty)
			measurementID := s.GetMeasurementID()
			fromCache := s.FromCache()
			for i, hi := range hops {
				if hopsSeen[hi] {
					continue
				}

				

				hopsSeen[hi] = true
				var h pb.RevtrHop
				h.Hop = hi
				if sType == pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT {
					// Look the type of each hop (exact or between)
					intersectHopType := s.(*TRtoSrcRevSegment).IntersectHopTypes[i]
					if intersectHopType == apb.IntersectHopType_BETWEEN {
						h.Type = pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT_BETWEEN
					} else if intersectHopType == apb.IntersectHopType_EXACT {
						h.Type = pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT
					} else {
						panic("Unknown intersecting type...?")
					}
					// h.Type = pb.RevtrHopType()
				} else {
					h.Type = sType 
				}
				h.MeasurementId = measurementID
				h.FromCache = fromCache
				if destinationBasedRoutingType, ok := rt.DestinationBasedRoutingByHop[hi]; ok {
					h.DestBasedRoutingType = pb.DestinationBasedRoutingType(destinationBasedRoutingType)
				} else  {
					h.DestBasedRoutingType = pb.DestinationBasedRoutingType_NO_CHECK
				}
				hopI, _ := util.IPStringToInt32(hi)

				rttInfos, ok := rt.RTTByHop[hopI]
				if !ok {
					h.Rtt = 0
					h.RttMeasurementId = 0
				} else {
					h.Rtt = rttInfos.RTT
					h.RttMeasurementId = rttInfos.MeasurementID
				}
				
				
				// Do not know if this check might be done here, but it seems a huge refactoring to include
				// it in the runner.  
				// if we intersected a traceroute, check if it was a direct intersection (exact or alias) or a
				// guess intersection (we intersected between two hops but we do not know which one)
				// if h.Type == pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT {
				// 	srcI, _ := util.IPStringToInt32(rt.Src)
				// 	dstI, _ := util.IPStringToInt32(rt.Dst)
				// 	log.Debugf("Revtr from %d to %d intersected traceroute %d of the atlas", srcI, dstI, rt.TracerouteID)
				// 	hopI, _ := util.IPStringToInt32(h.Hop)
				// 	path, err := atlasClient.CheckIntersectingPath(context.Background(), rt.TracerouteID, hopI)
				// 	if err != nil {
				// 		// Should never have an error
				// 		log.Error(err)
				// 		panic(err)
				// 	}
				// 	// Add the tr to src hops and return 
				// 	for _, hop := range(path.Hops){
				// 		hopS, _ := util.Int32ToIPString(hop.Ip)
				// 		if hop.IntersectHopType == apb.IntersectHopType_BETWEEN {
				// 			ret.Path = append(ret.Path, &pb.RevtrHop{
				// 				Hop:  hopS,
				// 				Type: pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT_BETWEEN,
				// 				MeasurementID: measurementID,
				// 			})
				// 		} else if hop.IntersectHopType == apb.IntersectHopType_EXACT {
				// 			ret.Path = append(ret.Path, &pb.RevtrHop{
				// 				Hop:  hopS,
				// 				Type: pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT,
				// 				MeasurementID: measurementID,
				// 				FromCache: fromCache,
				// 			})
				// 		}
				// 	}
				// 	return ret
				// }

				ret.Path = append(ret.Path, &h)
			}
		}
	}
	return ret
}

// InitializeTSAdjacents ...
func (rt *ReverseTraceroute) InitializeTSAdjacents(cls string, as types.AdjacencySource) error {
	adjs, err := getAdjacenciesForIPToSrc(cls, rt.Src, as)
	if err != nil {
		return err
	}
	rt.TSHop2AdjsLeft[cls] = adjs
	return nil
}

type byCount []types.AdjacencyToDest

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

type aByCount []types.Adjacency

func (b aByCount) Len() int           { return len(b) }
func (b aByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b aByCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

func getAdjacenciesForIPToSrc(ip string, src string, as types.AdjacencySource) ([]string, error) {
	ipint, _ := util.IPStringToInt32(ip)
	srcint, _ := util.IPStringToInt32(src)
	dest24 := srcint & 0xFFFFFF00

	ips1, err := as.GetAdjacenciesByIP1(ipint)
	if err != nil {
		return nil, err
	}
	ips2, err := as.GetAdjacenciesByIP2(ipint)
	if err != nil {
		return nil, err
	}
	adjstodst, err := as.GetAdjacencyToDestByAddrAndDest24(dest24, ipint)
	if err != nil {
		return nil, err
	}
	// Sort in descending order
	sort.Sort(sort.Reverse(byCount(adjstodst)))
	var atjs []string
	for _, adj := range adjstodst {
		ip, _ := util.Int32ToIPString(adj.Adjacent)
		atjs = append(atjs, ip)
	}
	combined := append(ips1, ips2...)
	sort.Sort(sort.Reverse(aByCount(combined)))
	var combinedIps []string
	for _, a := range combined {
		if a.IP1 == ipint {
			ips, _ := util.Int32ToIPString(a.IP2)
			combinedIps = append(combinedIps, ips)
		} else {
			ips, _ := util.Int32ToIPString(a.IP1)
			combinedIps = append(combinedIps, ips)
		}
	}
	ss := stringutil.StringSliceMinus(combinedIps, atjs)
	ss = stringutil.StringSliceMinus(ss, []string{ip})
	ret := append(atjs, ss...)
	if len(ret) < 30 {
		num := len(ret)
		return ret[:num], nil
	}
	return ret[:30], nil
}

// GetTSAdjacents get the set of adjacents to try for a hop
// for revtr:s,d,r, the set of R' left to consider, or if there are none
// will return the number that we want to probe at a time
func (rt *ReverseTraceroute) GetTSAdjacents(hop string, as types.AdjacencySource) []string {
	if _, ok := rt.TSHop2AdjsLeft[hop]; !ok {
		err := rt.InitializeTSAdjacents(hop, as)
		if err != nil {
			logRevtr(rt).Error(err)
		}
	}
	logRevtr(rt).Debug(rt.Src, " ", rt.Dst, " ", rt.LastHop(), " ", len(rt.TSHop2AdjsLeft[hop]), " TS adjacents left to try")

	// CASES:
	// 1. no adjacents left return nil
	if len(rt.TSHop2AdjsLeft[hop]) == 0 {
		return nil
	}
	// CAN EVENTUALLY MOVE TO REUSSING PROBES TO THIS DST FROM ANOTHER
	// REVTR, BUT NOT FOR NOW
	// 2. For now, we just take the next batch and send them
	var min int
	var rate int
	if val, ok := rt.TSHop2RateLimit[hop]; ok {
		rate = val
	} else {
		rate = 1
	}
	if rate > len(rt.TSHop2AdjsLeft[hop]) {
		min = len(rt.TSHop2AdjsLeft[hop])
	} else {
		min = rate
	}
	adjacents := rt.TSHop2AdjsLeft[hop][:min]
	rt.TSHop2AdjsLeft[hop] = rt.TSHop2AdjsLeft[hop][min:]
	return adjacents
}

// GetTimestampSpoofers gets spoofers to use for timestamp probes
func (rt *ReverseTraceroute) GetTimestampSpoofers(src, dst string, vpsource vpservice.VPSource) []string {
	var spoofers []string
	vps, err := vpsource.GetTSSpoofers(0)
	if err != nil {
		logRevtr(rt).Error(err)
		return nil
	}
	for _, vp := range vps {
		ips, _ := util.Int32ToIPString(vp.Ip)
		spoofers = append(spoofers, ips)
	}
	if len(spoofers) > 5 {
		return spoofers[:5]
	}
	return spoofers
}

const (
	// RateLimit is the number of Probes to send at once
	RateLimit = 5
)

// InitializeRRVPs initializes the rr vps for a cluster
func (rt *ReverseTraceroute) InitializeRRVPs(cls string, rkc rs.RankingSource, rrVPSelectionTechnique environment.RRVPsSelectionTechnique) error {
	logRevtr(rt).Debug("Initializing RR VPs individually for spoofers for ", cls)
	rt.RRHop2RateLimit[cls] = RateLimit // Number of spoofers at a time
	spoofersForTarget := []rspb.Source{
		{
			Ip: NON_SPOOFED,
		},
	} 
	// clsi, _ := util.IPStringToInt32(cls)
	if rkc == nil {
		// For backward compatibility
		err := "Ancient version of revtr running."
		log.Errorf(err)
		return errors.New(err)
	}
	vps, err := rkc.GetRRSpoofers(cls, rt.MaxSpoofers, rrVPSelectionTechnique)
	if err != nil {
		logRevtr(rt).Debug("error?")
		return err
	}
	rt.FailedIngress[cls] = make(map[string]int)
	rt.RRHop2VPSIngress[cls] = make(map[string]string)
	for _, vp := range vps {
		// ips, _ := util.Int32ToIPString(vp.Ip)
		spoofersForTarget = append(spoofersForTarget, *vp)
		rt.FailedIngress[cls][vp.Ingress] = 0
		rt.RRHop2VPSIngress[cls][vp.Ip] = vp.Ingress
	}
	rt.RRHop2VPSLeft[cls] = spoofersForTarget
	logRevtr(rt).Debug("Spoofers are:", spoofersForTarget)
	return nil
}

func stringSliceIndexWithClusters(ss []string, seg string, cm clustermap.ClusterMap) int {
	// Return the max because otherwise we have an index error in AddBackgroundTRSegment function
	maxIndex := -1
	for i, s := range ss {
		if cm.Get(s) == cm.Get(seg) {
			maxIndex = i
		}

	}
	return maxIndex
}

// AddBackgroundTRSegment need a different function because a TR segment might intersect
// at an IP back up the TR chain, want to delete anything that has been added along the way
func (rt *ReverseTraceroute) AddBackgroundTRSegment(trSeg Segment, cm clustermap.ClusterMap) bool {
	logRevtr(rt).Debug("Adding Background trSegment ", trSeg)
	var found *ReversePath
	// iterate through the paths, trying to find one that contains
	// intersection point, chunk is a ReversePath
	for _, chunk := range *rt.Paths {
		var index int
		logRevtr(rt).Debug("Looking for ", trSeg.Hops()[0], " in ", chunk.Hops())
		chunkHops := chunk.Hops()
		if index = stringSliceIndexWithClusters(chunkHops, trSeg.Hops()[0], cm); index != -1 {
			logRevtr(rt).Debug("Intersected: ", trSeg.Hops()[0], " in ", chunk)
			chunk = chunk.Clone()
			found = chunk
			// Iterate through all the segments until you find the hop
			// where they intersect. After reaching the hop where
			// they intersect, delete any subsequent hops within
			// the same segment. Then delete any segments after.
			var k int // Which IP hop we're at in the chunk
			var j int // Which segment we're at
			lenChunk := len(chunkHops)-1
			for ; lenChunk > index ; lenChunk = len(chunk.Hops())-1 {
				// get current segment
				seg := (*chunk.Path)[j]
				// if we're past the intersection point then delete the whole segment
				if k > index {
					*chunk.Path, (*chunk.Path)[j] = append((*chunk.Path)[:j], (*chunk.Path)[j+1:]...), nil
				} else if k+len(seg.Hops())-1 > index {
					l := stringutil.StringSliceIndex(seg.Hops(), trSeg.Hops()[0]) + 1
					for k+len(seg.Hops())-1 > index {
						seg.RemoveAt(l)
					}
				} else {
					j++
					k += len(seg.Hops())
				}
			}
			break
		}
	}
	if found == nil {
		logRevtr(rt).Debug(trSeg)
		logRevtr(rt).Debug("Tried to add traceroute to Reverse Traceroute that didn't share an IP... what happened?!")
		return false
	}
	*rt.Paths = append(*rt.Paths, found)
	// Now that the traceroute is cleaned up, add the new segment
	// this sequence slightly breaks how add_segment normally works here
	// we append a cloned path (with any hops past the intersection trimmed).
	// then, we call add_segment. Add_segment clones the last path,
	// then adds the segemnt to it. so we end up with an extra copy of found,
	// that might have soem hops trimmed off it. not the end of the world,
	// but something to be aware of
	success := rt.AddSegments([]Segment{trSeg}, cm)
	if !success {
		for i, s := range *rt.Paths {
			if found == s {
				*rt.Paths, (*rt.Paths)[rt.len()-1] = append((*rt.Paths)[:i], (*rt.Paths)[i+1:]...), nil
			}
		}
	}
	return success
}

// AddSpoofResults update the trace structure of the reverse traceroute
// to be able to evaluate choices of spoofers
func (rt *ReverseTraceroute) AddRankedSpoofers(target string, vps []string ) error {
	targetI, err := networkutil.IPStringToInt32(target)
	if err != nil {
		log.Errorln(err)
		return err
	}
	if _, ok := rt.RankedSpoofersByHop[targetI]; !ok {
		rt.RankedSpoofersByHop[targetI] = make(map[uint32]*RankedSpooferMeasurement)
	}
	previousMaxRank := len(rt.RankedSpoofersByHop[targetI]) + 1
	for rank, vp := range vps{
		vpI, err := networkutil.IPStringToInt32(vp)
		if err != nil {
			log.Errorln(err)
			return err
		}
		rt.RankedSpoofersByHop[targetI][vpI] = &RankedSpooferMeasurement{
			IP: vpI,
			Rank: uint32(rank) + uint32(previousMaxRank),
		}
	}
	return nil
}

func (rt *ReverseTraceroute) MatchSpoofedResponses(target string, rrs map[uint32]types.SpoofRRHops, vps [] string ) error {
	targetI, err := networkutil.IPStringToInt32(target)
	if err != nil {
		log.Errorln(err)
		return err
	}

	// At this point the target AND the spooferVP MUST be in the map (otherwise it is an error)
	for _, spooferVP := range(vps) {
		spooferVPI, _ := networkutil.IPStringToInt32(spooferVP)
		if results, ok := rrs[spooferVPI]; ok {
			rt.RankedSpoofersByHop[targetI][spooferVPI].MeasurementID = results.MeasurementID
		} else {
			rt.RankedSpoofersByHop[targetI][spooferVPI].MeasurementID = 0
		}
	}
	return nil
}


// GetRRVPsNoUpdate returns spoofers for destination based routing checks (do not update internal revtr structure)
func (rt *ReverseTraceroute) GetRRVPsNoUpdate(dst string, 
	rkc rs.RankingSource,
	rrVPSelectionTechnique environment.RRVPsSelectionTechnique) ([]string, error) {
	vps, err := rkc.GetRRSpoofers(dst, 250, rrVPSelectionTechnique)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	spoofers := [] string{}
	for _, vp := range(vps) {
		spoofers = append(spoofers, vp.Ip)
	}
	return spoofers, nil
}

// GetRRVPs returns the next set of VPs to probe from, plus the next destination to probe
// nil means none left, already probed from anywhere
// the first time, initialize the set of VPs
// if any exist that have already probed the dst but haven't been used in
// this reverse traceroute, return them, otherwise return [:non_spoofed] first
// then set of spoofing VPs on subsequent calls
func (rt *ReverseTraceroute) GetRRVPs(dst string, 
	// vps vpservice.VPSource, 
	rkc rs.RankingSource, 
	rrVPSelectionTechnique environment.RRVPsSelectionTechnique) ([]string, string) {
	// logRevtr(rt).Debug("GettingRRVPs for ", dst)
	// we either use destination or cluster, depending on how flag is set
	hops := rt.CurrPath().LastSeg().Hops()
	for _, hop := range hops {
		if _, ok := rt.RRHop2VPSLeft[hop]; !ok {
			// Initialize all the ingress as being tested 0 time.
			rt.InitializeRRVPs(hop, rkc, rrVPSelectionTechnique)
		}
	}
	// CASES:
	segHops := stringutil.CloneStringSlice(rt.CurrPath().LastSeg().Hops())
	// logRevtr(rt).Debug("segHops: ", segHops)
	var target, cls *string
	target = new(string)
	cls = new(string)
	var foundVPs bool
	for !foundVPs && len(segHops) > 0 {
		*target, segHops = segHops[len(segHops)-1], segHops[:len(segHops)-1]
		*cls = *target
		// logRevtr(rt).Debug("Sending RR probes to: ", *cls)
		// logRevtr(rt).Debug("RR VPS: ", rt.RRHop2VPSLeft[*cls])
		// 0. destination seems to be unresponsive
		if rt.rrSpoofRRResponsive[*cls] != -1 &&
			rt.rrSpoofRRResponsive[*cls] >= maxUnresponsive {
			// this may not match exactly but I think it does
			// log.Debug("GetRRVPs: unresponsive for: ", *cls)
			continue
		}
		// 1. no VPs left, return nil
		if len(rt.RRHop2VPSLeft[*cls]) == 0 {
			// log.Debug("GetRRVPs: No VPs left for: ", *cls)
			continue
		}
		foundVPs = true
	}
	if !foundVPs {
		return nil, ""
	}
	// logRevtr(rt).Debug(rt.Src, " ", rt.Dst, " ", *target, " ", len(rt.RRHop2VPSLeft[*cls]), " RR VPs left to try")
	// 2. send non-spoofed version if it is in the next batch
	// min := rt.RRHop2RateLimit[*cls]
	// if len(rt.RRHop2VPSLeft[*cls]) < min {
	// 	min = len(rt.RRHop2VPSLeft[*cls])
	// }
	// logRevtr(rt).Debug("Getting vps for: ", *cls, " min: ", min)

	remainingSpoofersIP := [] string {}
	for _, vp := range rt.RRHop2VPSLeft[*cls] {
		remainingSpoofersIP = append(remainingSpoofersIP, vp.Ip)
	}

	if stringutil.InArray(remainingSpoofersIP, NON_SPOOFED) {
		rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][1:]
		return []string{NON_SPOOFED}, *target
	}

	// 3. use unused spoofing VPs

	// notEmpty := rt.len() > 0
	// var isRRRev *bool
	// isRRRev = new(bool)
	// spoofer := new(string)
	// if rrev, ok := rt.CurrPath().LastSeg().(*SpoofRRRevSegment); ok {
	// 	*isRRRev = true
	// 	*spoofer = rrev.SpoofSource
	// }
	// if the current last hop was discovered with spoofed, and it
	// hasn't been used yet, use it
	// Why are we doing this?
	// if notEmpty && *isRRRev {
	// 	// logRevtr(rt).Debug("Found recent spoofer to use ", *spoofer)
	// 	var newleft []string
	// 	for _, s := range rt.RRHop2VPSLeft[*cls] {
	// 		if s == *spoofer {
	// 			continue
	// 		}
	// 		newleft = append(newleft, s)
	// 	}
	// 	rt.RRHop2VPSLeft[*cls] = newleft
	// 	min := rt.RRHop2RateLimit[*cls] - 1
	// 	if len(rt.RRHop2VPSLeft[*cls]) < min {
	// 		min = len(rt.RRHop2VPSLeft[*cls])
	// 	}
	// 	vps := append([]string{*spoofer}, rt.RRHop2VPSLeft[*cls][:min]...)
	// 	rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][min:]
	// 	return vps, *target
	// }
	min := rt.RRHop2RateLimit[*cls]
	if len(rt.RRHop2VPSLeft[*cls]) < min {
		min = len(rt.RRHop2VPSLeft[*cls])
	}

	i := 0
	touse := [] string {}
	for _, vp := range rt.RRHop2VPSLeft[*cls] {
		i++
		if nFailedSpoofers, ok := rt.FailedIngress[*cls][vp.Ingress] ; ok {
			if nFailedSpoofers >= MAX_INGRESS_VPS && vp.Ingress != "0.0.0.0" {
				// The ingress has failed, so skip this VP, 0.0.0.0 comes from global ranking
				continue
			}
		} 
		// Enqueue the vp otherwise
		touse = append(touse, vp.Ip)
		if len(touse) == min {
			break
		}
	}
	// Do not use these VPs anymore
	rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][i:]
	
	// logRevtr(rt).Debug("Returning VPS for spoofing: ", touse)
	return touse, *target
}

// CreateReverseTraceroute creates a reverse traceroute for the web interface
func CreateReverseTraceroute(revtr pb.RevtrMeasurement, cs types.ClusterSource,
	onAdd OnAddFunc, onFail OnFailFunc, onReach OnReachFunc) *ReverseTraceroute {
	if revtr.AtlasOptions == nil {
		// Default parameters
		revtr.AtlasOptions = &pb.AtlasOptions{
			UseAtlas: true,
			UseRrPings: true,
			IgnoreSource: false,
			IgnoreSourceAs: false,
			Platforms: []string{"mlab", "ripe", fmt.Sprintf("mlab_initial_%s", revtr.Src)},
			Staleness: 60 * 24,
			StalenessBeforeRefresh: 60 * 24,
		}
		revtr.IsRunRttPings = true
		revtr.UseCache = true
		revtr.MaxSpoofers = 10
		revtr.RrVpSelectionAlgorithm = string(environment.IngressCover)
	}
	if revtr.HeuristicsOptions == nil {
		revtr.HeuristicsOptions = &pb.RRHeuristicsOptions{
			UseDoubleStamp: true,
		}
	}
	if revtr.CheckDestBasedRoutingOptions == nil {
		revtr.CheckDestBasedRoutingOptions = &pb.CheckDestBasedRoutingOptions{
			CheckTunnel: false,
		}
	}

	if revtr.SymmetryOptions == nil {
		revtr.SymmetryOptions = &pb.SymmetryOptions{
			IsAllowInterdomainSymmetry: false,
		}
	}

	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Label, revtr.Id, 
		 environment.RRVPsSelectionTechnique(revtr.RrVpSelectionAlgorithm), 
		 revtr.MaxSpoofers,
		 revtr.UseTimestamp, 
		 revtr.UseCache, 
		 AtlasOptions{
			 UseAtlas: revtr.AtlasOptions.UseAtlas,
			 UseRRPings: revtr.AtlasOptions.UseRrPings,
			 IgnoreSource: revtr.AtlasOptions.IgnoreSource,
			 IgnoreSourceAS: revtr.AtlasOptions.IgnoreSourceAs,
			 Platforms: revtr.AtlasOptions.Platforms,
			 StalenessBeforeRefresh: revtr.AtlasOptions.StalenessBeforeRefresh,
			 Staleness: revtr.AtlasOptions.Staleness,
		 },
		 SymmetryOptions{
			IsAllowInterdomainSymmetry: revtr.SymmetryOptions.IsAllowInterdomainSymmetry, 
		 },
		 RRHeuristicsOptions{
			UseDoubleStamp: revtr.HeuristicsOptions.UseDoubleStamp,
		 },
		 CheckDestinationBasedRoutingOptions{
			 CheckTunnel: revtr.CheckDestBasedRoutingOptions.CheckTunnel,
		 },
		 revtr.IsRunForwardTraceroute, 
		 revtr.IsRunRttPings,
		)
	rt.BackoffEndhost = revtr.BackoffEndhost
	rt.OnAdd = onAdd
	rt.OnFail = onFail
	rt.OnReach = onReach
	return rt
}
