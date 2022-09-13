/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package server

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/types"
	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	radix "github.com/NEU-SNS/ReverseTraceroute/radix"
	rkc "github.com/NEU-SNS/ReverseTraceroute/rankingservice/client"
	"github.com/NEU-SNS/ReverseTraceroute/ripe"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

var (
	nameSpace     = "atlas"
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
	tracerouteGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "measurements",
		Name:      "traceroutes",
		Help:      "The current number of runninsg traceroutes",
	})
)

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(tracerouteGauge)
}

// AtlasServer is the interface for the atlas
type AtlasServer interface {
	GetIntersectingPath(*pb.IntersectionRequest) (*pb.IntersectionResponse, error)
	GetPathsWithToken(*pb.TokenRequest) (*pb.TokenResponse, error)
	InsertTraceroutes(*pb.InsertTraceroutesRequest) (*pb.InsertTraceroutesResponse, error)
	RunAtlasRRPings(*pb.RunAtlasRRPingsRequest) (*pb.RunAtlasRRPingsResponse, error)
	CheckIntersectingPath(*pb.CheckIntersectionRequest) (*pb.CheckIntersectionResponse, error)
	MarkTracerouteStale(req *pb.MarkTracerouteStaleRequest) (*pb.MarkTracerouteStaleResponse, error)
	MarkTracerouteStaleSource(req *pb.MarkTracerouteStaleSourceRequest) (*pb.MarkTracerouteStaleSourceResponse, error)
	RunTracerouteAtlasToSource(req *pb.RunTracerouteAtlasToSourceRequest)  (*pb.RunTracerouteAtlasToSourceResponse, error)
	GetAvailableHopAtlasPerSource(req *pb.GetAvailableHopAtlasPerSourceRequest) (*pb.GetAvailableHopAtlasPerSourceResponse, error)

}

// Config is the config for the atlas
type Config struct {
	DB       repo.Configs
	RootCA   string `flag:"root-ca"`
	CertFile string `flag:"cert-file"`
	KeyFile  string `flag:"key-file"`
	IP2AS    string `flag:"ip2as"`
	RIPE RIPEConfig `flag:"ripe"`
}

type RIPEConfig struct {
	Email string `flag:"email"`
	Key string `flag:"key"`
}

type server struct {
	donec chan struct{}
	curr  runningTraces
	opts  serverOptions
	tc    *tokenCache
	limit *rate.Limiter
	// Mapping between the source IP addresses of the RIPE probes and their probe ID. 
	mapRIPEProbeID map[string]int
	// Limiting the number of queries going into the database
	flyingIntersectQueries chan int
	traceroutesToRefreshChan chan types.RefreshTracerouteIDChan // Chanel to allow revtrs to ask for traceroute refresh. 
	// staleness float64 
	ripeAccount string
	ripeKey string

	// Specific for staleness evaluation, should be removed in production
	ripeEvaluationAccount string
	ripeEvaluationKey string  
	traceroutesToRefreshEvaluationChan chan types.RefreshTracerouteIDChan // Chanel to allow revtrs to ask for traceroute refresh. 
	useRefresh bool 
	useRefreshEvaluation bool
	home string 
	atlasHopsPerSource map[uint32]map[uint32]struct{}
	atlasHopsPerSourceLock sync.RWMutex
}

type serverOptions struct {
	cl  cclient.Client
	vps client.VPSource
	trs types.TRStore
	ca  Cache
	rkcl rkc.RankingSource
	// Mapping between the source IP addresses of the RIPE probes and their AS. 
	rt  radix.BGPRadixTree
	
}

const (
	ATLAS_PING_LABEL = "atlas_rr_ping"
)

// Cache is the cache used for the atlas
type Cache interface {
	Get(interface{}) (interface{}, bool)
	Add(interface{}, interface{}) bool
	Remove(interface{})
}

// Option sets an option to configure the server
type Option func(*serverOptions)

// WithClient configures the server with client c
func WithClient(c cclient.Client) Option {
	return func(opts *serverOptions) {
		opts.cl = c
	}
}

// WithVPS configures the server with the given VPSource
func WithVPS(vps client.VPSource) Option {
	return func(opts *serverOptions) {
		opts.vps = vps
	}
}

// WithTRS configures the server with the given TRStore
func WithTRS(trs types.TRStore) Option {
	return func(opts *serverOptions) {
		opts.trs = trs
	}
}

// WithCache configures the server to use the cache ca
func WithCache(ca Cache) Option {
	return func(opts *serverOptions) {
		opts.ca = ca
	}
}

// WithRanking configures the server to use the ranking source
func WithRanking(rks rkc.RankingSource) Option {
	return func(opts *serverOptions) {
		opts.rkcl = rks
	}
}

// WithRadix configures the server to use radix
func WithRadix(rt radix.BGPRadixTree) Option {
	return func(opts *serverOptions) {
		opts.rt = rt
	}
}

// NewServer creates a server
func NewServer(conf Config ,opts ...Option) AtlasServer {
	atlas := &server{
		curr: newRunningTraces(),
	}
	for _, opt := range opts {
		opt(&atlas.opts)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	atlas.home = home

	atlas.tc = newTokenCache(atlas.opts.ca)
	atlas.limit = rate.NewLimiter(rate.Every(time.Millisecond*12), 5000)
	atlas.flyingIntersectQueries = make(chan int, 500)
	// _, atlas.mapRIPEProbeID = util.GetRIPEProbesIP(true, home + "/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/resources/ripe_probes_atlas.json")
	// _, ripeAnchorID := util.GetRIPEAnchorsIP()
	// for ip, probeID := range(ripeAnchorID) {
	// 	atlas.mapRIPEProbeID[ip] = probeID
	// }


	atlas.ripeAccount = conf.RIPE.Email
	atlas.ripeKey = conf.RIPE.Key

	log.Infof("Number of RIPE probes available: %d", len(atlas.mapRIPEProbeID))
	atlas.traceroutesToRefreshChan = make(chan types.RefreshTracerouteIDChan, 10000000)

	atlas.traceroutesToRefreshEvaluationChan = make(chan types.RefreshTracerouteIDChan, 10000000)
	atlas.useRefresh = false
	atlas.useRefreshEvaluation = false
	// Periodically fill that mapping has not changed 
	
	atlas.atlasHopsPerSourceLock = sync.RWMutex{}
	atlas.GetAvailableHopAtlasPerSource(&pb.GetAvailableHopAtlasPerSourceRequest{})
	// go  atlas.GetAvailableHopAtlasPerSourcePeriodic()

	// Refresh the traceroutes every X hours
	// go atlas.refreshTraceroutes()
	if atlas.useRefreshEvaluation {
		go atlas.refreshTraceroutesEvaluation()
	}
	// go atlas.MapRIPEProbeIDPeriodic()
	go atlas.opts.trs.MonitorDB()
	return atlas
}

func (a * server) GetAvailableHopAtlasPerSourcePeriodic() {

	t := time.NewTicker(time.Hour * 12)

	log.Infof("Loading available hops in memory...")
	var err error 
	a.atlasHopsPerSource, err = a.opts.trs.GetAvailableHopAtlasPerSource()
	if err != nil {
		panic(err)
	}

	log.Infof("Loading available hops in memory...Done")
	log.Infof("Found atlas traceroutes for %d sources", len(a.atlasHopsPerSource))

	for {
		select{
		case <-t.C:
			a.atlasHopsPerSourceLock.Lock()
			a.atlasHopsPerSource, err = a.opts.trs.GetAvailableHopAtlasPerSource()
			if err != nil {
				panic(err)
			}
			log.Infof("Loading available hops in memory...Done")
			log.Infof("Found atlas traceroutes for %d sources", len(a.atlasHopsPerSource))
			a.atlasHopsPerSourceLock.Unlock()
		}
	}
}

func (a * server) MapRIPEProbeIDPeriodic() {
	t := time.NewTicker(12 * time.Hour)
	for {
		select{
		case <-t.C:
			_, newMapping := util.GetRIPEProbesIP(true, a.home + "/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/resources/ripe_probes.json")
			_, ripeAnchorID := util.GetRIPEAnchorsIP()
			for ip, probeID := range(ripeAnchorID) {
				newMapping[ip] = probeID
			}
			a.mapRIPEProbeID = newMapping
		}
	}
}

func (a * server) runRRIntersectionFromIds(idsFile string, ids [] int64) error {
	// Now run the atlas_rr_intersection
	atlas.ProbeIDsToFile(idsFile, ids)
	cmd := fmt.Sprintf(`cd %s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice && 
	python3 -m atlas.record_route --mode=selection --ids-file=%s`, a.home, idsFile) 
	stdout, err := util.RunCMD(cmd)
	if err != nil {
		log.Error(stdout)
		log.Error(err)
		return err
	}
	return nil
}


func (a * server) runRRIntersectionFromLabel(label string) error {
	// Now run the atlas_rr_intersection
	cmd := fmt.Sprintf(`cd %s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice && 
	python3 -m atlas.record_route --mode=label --label=%s`, a.home, label) 
	stdout, err := util.RunCMD(cmd)
	if err != nil {
		log.Error(stdout)
		log.Error(err)
		return err
	}
	return nil 
}

func (a * server) updateChangedTraceroutes(traceroutes [] *dm.Traceroute, 
	tracerouteIDPerMeta map[types.RefreshTracerouteMeta]int64,
	metaPerTracerouteID map[int64]types.RefreshTracerouteMeta, 
	tracerouteIds []int64 ) (map[int64] bool, error) {
	batchSize := 50000
	changedTraceroutes := []*dm.Traceroute {}
	changedTraceroutesIds := []int64 {} 
	stillIntersectIds := map[int64] bool {}
	for i := 0; i < len(traceroutes); i+= batchSize {
		maxIndex := i+batchSize
		if maxIndex > len(traceroutes) {
			maxIndex = len(traceroutes)
		}
		resp, err := a.InsertTraceroutes(&pb.InsertTraceroutesRequest{
			Traceroutes: traceroutes[i:maxIndex],
			IsRunRrPings: false,
		})
		if err != nil {
			log.Error(err)
			return nil, err 
		}
		for j, id := range(resp.Ids) {
			meta := types.RefreshTracerouteMeta {
				Dst: traceroutes[i+j].Dst,
				Src: traceroutes[i+j].Src,
			}
			oldTracerouteID := tracerouteIDPerMeta[meta]
			oldMeta := metaPerTracerouteID[oldTracerouteID]
			// Check if the traceroute has changed or not.
			hasChanged, stillIntersect, err := a.opts.trs.CompareOldAndNewTraceroute(oldMeta.IntersectionIP, oldTracerouteID, id)
			if err != nil {
				log.Error(err)
				return nil, err
			}
			if !hasChanged {
				a.opts.trs.UpdateRRIntersectionTracerouteNonStale(oldTracerouteID, id)
			} else {
				changedTraceroutes = append(changedTraceroutes, traceroutes[i+j])
				changedTraceroutesIds = append(changedTraceroutesIds, id)
			}
			if stillIntersect {
				stillIntersectIds[oldTracerouteID] = true
			} else {
				stillIntersectIds[oldTracerouteID] = false
			}
		}
	}
	a.RunAtlasRRPings(&pb.RunAtlasRRPingsRequest{
		Traceroutes: changedTraceroutes,
	})
	if len(changedTraceroutesIds) > 0 {
		uuid := util.GenUUID()
		idsFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/atlas/resources/refresh/atlas_refresh_mlab_%s.ids", 
		a.home, uuid)
		a.runRRIntersectionFromIds(idsFile, changedTraceroutesIds)
	}
	// Mark the refreshed traceroutes as stale 
	err := a.opts.trs.MarkTraceroutesStale(tracerouteIds)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return stillIntersectIds, nil 
} 

func (a * server) refreshMLabTraceroutes(traceroutesToRefresh map[int64]struct{}, 
	metaPerTracerouteID map[int64]types.RefreshTracerouteMeta,
	tracerouteIDPerMeta map[types.RefreshTracerouteMeta]int64) {
	
	traceroutesMeasurements := [] *dm.TracerouteMeasurement {}
	traceroutesIds := []int64 {}
	platform := ""
	for id, _ := range(traceroutesToRefresh) {
		meta := metaPerTracerouteID[id]
		traceroutesMeasurements = append(traceroutesMeasurements, &dm.TracerouteMeasurement{
			// Src:        3589431500,
			// Dst:        1357876940,
			Src: 		meta.Src,
			Dst: 		meta.Dst,
			Timeout:    60,
			Wait:       "2",
			Attempts:   "1",
			LoopAction: "1",
			Loops:      "3",
			CheckCache: false,
			SaveDb:     false,
			Staleness:  60,
		})
		platform = meta.Platform
		traceroutesIds = append(traceroutesIds, id)
	}

	_, err := atlas.RunTraceroutes(a.opts.cl, traceroutesMeasurements, platform)
	if err != nil {
		log.Error(err)
	}

	// TODO refresh all the MLab traceroutes?

	// stillIntersect, err := a.updateChangedTraceroutes(traceroutes, tracerouteIDPerMeta, metaPerTracerouteID, traceroutesIds, )
	// if err != nil {
	// 	panic(err)
	// }
	// // Finally tell all the intersection request that we're done refreshing
	// for id, channels := range (traceroutesToRefresh) {
	// 	for _, channel := range(channels) {
	// 		// To avoid overloading the database with queries 
	// 		time.Sleep(time.Duration(rand.Intn(10))* time.Millisecond)
	// 		channel <- types.RefreshStillIntersect{
	// 			IsRefreshed: true,
	// 			IsStillIntersect: stillIntersect[id],
	// 		}
	// 	}
	// }
	return 
}

func (a * server) refreshRIPETraceroutes(nTraceroutesPerSource int,
	 traceroutesToRefresh map[int64]struct{},
	 metaPerTracerouteID map[int64]types.RefreshTracerouteMeta,
	 tracerouteIDPerMeta map[types.RefreshTracerouteMeta]int64) {


	// Struct to sort the traceroutes per weight
	// type kv struct {
	// 	Key int64
	// 	Value int 
	// }
	// // Sort the traceroutes by their weight
	// sortedTraceroutesByWeight := [] kv {}
	// for tracerouteID, weight := range(weightPerTracerouteID) {
	// 	sortedTraceroutesByWeight = append(sortedTraceroutesByWeight, kv{tracerouteID, weight})
	// }
	// sort.Slice(sortedTraceroutesByWeight, func(i, j int) bool {
	// 	return sortedTraceroutesByWeight[i].Value > sortedTraceroutesByWeight[j].Value
	// })
	// Select the best traceroutes
	// bestSortedTraceroutes := [] atlas.SrcProbeID {}
	// refreshedTracerouteIds := map[int64] struct {} {}
	// for _, traceroute := range(sortedTraceroutesByWeight) {
	// 	if len(refreshedTracerouteIds) == budgetPerStaleness {
	// 		break
	// 	}
	// 	meta := metaPerTracerouteID[traceroute.Key]
	// 	dstS, _ := util.Int32ToIPString(meta.Dst)
	// 	srcS, _ := util.Int32ToIPString(meta.Src)
	// 	if probeID, ok := a.mapRIPEProbeID[srcS]; ok {
	// 		refreshedTracerouteIds[traceroute.Key] = struct{}{}
	// 		bestSortedTraceroutes = append(bestSortedTraceroutes, atlas.SrcProbeID{
	// 			Src: dstS,
	// 			ProbeID: probeID,
	// 		})
	// 	} 
	// 	// else {
	// 	// 	log.Infof("src %s not in RIPE probes", srcS)
	// 	// }
	// }
	

	// traceroutesToRefresh are traceroutes that intersected

	// Group them per source
	traceroutesPerSource := map[uint32]map[int] struct {}{}
	for tracerouteID, _ := range(traceroutesToRefresh) {
		meta := metaPerTracerouteID[tracerouteID]
		if _, ok := traceroutesPerSource[meta.Src]; !ok{
			traceroutesPerSource[meta.Src] = make(map[int]struct{})
		}
		srcS, _ := util.Int32ToIPString(meta.Src)
		if probeID, ok := a.mapRIPEProbeID[srcS]; ok {
			if len(traceroutesPerSource[meta.Src]) < nTraceroutesPerSource {
				traceroutesPerSource[meta.Src][probeID] = struct{}{}
			}	
		}
	}

	//  

	for _, traceroutesSource := range(traceroutesPerSource) {
		// Select some more random traceroutes 
		traceroutesCandidates := [] int {}
		for _, probeID := range(a.mapRIPEProbeID) {
			if _, ok := traceroutesSource[probeID]; !ok {
				traceroutesCandidates = append(traceroutesCandidates, probeID)
			}
		}

		// TODO should remove from the traceroute candidates the traceroutes we already issued but did not intersect... 
		// Otherwise those ones could be re-issued and (potentially) make the convergence slower
		// Find an optimization so that the query does not take a lot of time.   


		// Randomly select the traceroute candidates
		perm  := rand.Perm(len(traceroutesCandidates))
		for i := 0; i < len(perm) && len(traceroutesSource) < nTraceroutesPerSource ; i ++ {
			traceroutesSource[traceroutesCandidates[perm[i]]] = struct{}{}
		}
	}
	traceroutes := [] atlas.SrcProbeID{}
	for src, traceroutesSrc := range(traceroutesPerSource) {
		srcS, _ := util.Int32ToIPString(src)
		for probeID, _ := range(traceroutesSrc) {
			traceroutes = append(traceroutes, atlas.SrcProbeID{
				Src: srcS,
				ProbeID: probeID,
			})
		}
	}
	// Write the traceroutes to refresh in a file 
	description := "Tracerouter atlas refresh, uuid "
	uuid := atlas.RunRIPETraceroutesPython(traceroutes, a.ripeKey, a.ripeAccount, description)
	// Wait time function of the window of refresh (number or traceroutes is a function of that window)
	// minTimeWait := 10 // 10 second minimum
	// maxTimeWait := 300 // 5 minutes maximum
	// waitTime := int(60 * a.staleness / 5) // 1/5 of the window of refreshment (in seconds)
	// if waitTime < minTimeWait {
	// 	waitTime = minTimeWait
	// }
	// if waitTime > maxTimeWait {
	// 	waitTime = maxTimeWait
	// }
	waitTime := 600 // 10 minutes to wait before collecting RIPE results. 
	//1000 tr to one source takes approx 7 minutes...
	time.Sleep(time.Duration(waitTime) * time.Second)
	dumpFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/atlas/resources/refresh/atlas_refresh_%s.json", 
	a.home, uuid)
	atlas.DumpRIPETraceroutes(uuid, dumpFile, a.ripeKey, a.ripeAccount)
	// Now fetch the results and convert them to traceroutes
	ripeAtlasSourceASNFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/cmd/ripeatlascontroller/resources/sources_asn.csv",
	 a.home)
	ripeTraceroutes, err := ripe.FetchRIPEMeasurementsToTraceroutes(dumpFile, ripeAtlasSourceASNFile, "ripe")
	if err != nil {
		log.Error(err)
		panic(err)
	}
	_, err = a.InsertTraceroutes(&pb.InsertTraceroutesRequest{
		Traceroutes: ripeTraceroutes,
		IsRunRrPings: true,
		IsRunRrIntersections: true,
	})

	if err != nil {
		log.Error(err)
	}
	return 
}

func (a * server) refreshTraceroutesEvaluation() {
	refreshWindow := time.Second * time.Duration(60 * 15) // Refresh every 15 minutes
	// refreshWindow := time.Second * time.Duration(10)
	refreshTicker := time.NewTicker(refreshWindow)

	intersectedTraceroutesRIPE := map[int64] types.RefreshTracerouteMeta {}
	intersectedTraceroutesMLab := map[int64] types.RefreshTracerouteMeta {}

	for {
		select {
		case <-refreshTicker.C:
			
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				if len(intersectedTraceroutesMLab) > 0 {
					traceroutesMeasurements := []*dm.TracerouteMeasurement {}
					platform := ""
					for _, meta := range(intersectedTraceroutesMLab) {
						platform = "staleness_evaluation_" + meta.Platform
						traceroutesMeasurements = append(traceroutesMeasurements, &dm.TracerouteMeasurement{
							// Src:        3589431500,
							// Dst:        1357876940,
							Src: 		meta.Src,
							Dst: 		meta.Dst,
							Timeout:    60,
							Wait:       "2",
							Attempts:   "1",
							LoopAction: "1",
							Loops:      "3",
							CheckCache: false,
							SaveDb:     false,
							Staleness:  60,
						})
					}
					log.Infof("Running refresh evaluation for %d MLab traceroutes", len(traceroutesMeasurements))
					traceroutes, err := atlas.RunTraceroutes(a.opts.cl, traceroutesMeasurements, platform)
					if err != nil {
						log.Error(err)
					}
					_, err = a.InsertTraceroutes(&pb.InsertTraceroutesRequest{
						Traceroutes: traceroutes,
						IsRunRrPings: false,
					})
					if err != nil {
						panic(err)
					}
				}		
			} ()

			wg.Add(1)
			go func() {

				defer wg.Done()
				// Run the intersected traceroutes. 
				if len(intersectedTraceroutesRIPE) > 0 {
					traceroutesSrcProbeIDToRefresh := [] atlas.SrcProbeID{}
					for _, meta := range(intersectedTraceroutesRIPE) {
						dstS, _ := util.Int32ToIPString(meta.Dst)
						srcS, _ := util.Int32ToIPString(meta.Src)
						if probeID, ok :=  a.mapRIPEProbeID[srcS]; ok {
							if probeID != 0 {
								traceroutesSrcProbeIDToRefresh = append(traceroutesSrcProbeIDToRefresh, atlas.SrcProbeID{
									Src: dstS,
									ProbeID: probeID,
								})
							}
							
						}
						
					}
					log.Infof("Running refresh evaluation for %d RIPE traceroutes", len(traceroutesSrcProbeIDToRefresh))
					description := "Tracerouter atlas refresh evaluation, uuid "
					atlas.RunRIPETraceroutesPython(traceroutesSrcProbeIDToRefresh, 
						a.ripeEvaluationKey, a.ripeEvaluationAccount, description)
				}
			} ()
			wg.Wait()
			// Reset the intersected traceroutes. 
			intersectedTraceroutesRIPE = make(map[int64]types.RefreshTracerouteMeta)
			intersectedTraceroutesMLab = make(map[int64]types.RefreshTracerouteMeta)
			break
		case tracerouteIntersected := <- a.traceroutesToRefreshEvaluationChan:
			// We just want to run one instance of each traceroute we intersected
			log.Infof("Making refresh request evaluation for traceroute %d", tracerouteIntersected.TracerouteID)
			if tracerouteIntersected.Meta.Platform == "ripe" {
				intersectedTraceroutesRIPE[tracerouteIntersected.TracerouteID] = tracerouteIntersected.Meta 
			} else {
				intersectedTraceroutesMLab[tracerouteIntersected.TracerouteID] = tracerouteIntersected.Meta 
			}
			
		}
	}
}

func (a * server) refreshTraceroutes(){
 
	refreshWindowMinutes := 24 *  60 // 24 hours 
	refreshWindowSeconds := 60 * refreshWindowMinutes // in seconds
	// refreshWindow := time.Second * time.Duration(60 * a.staleness)
	
	refreshWindow := time.Second * time.Duration(refreshWindowSeconds)
	refreshTicker := time.NewTicker(refreshWindow)

	done := make(chan bool)

	ripeTraceroutesToRefresh := make(map[int64]struct{})
	traceroutesToRefresh := make(map[int64]struct{})

	// tracerouteIDsPerSource := make(map[uint32][]int64)
	metaPerTracerouteID := make(map[int64]types.RefreshTracerouteMeta)
	tracerouteIDPerMeta := make(map[types.RefreshTracerouteMeta]int64)

	type RefreshedTimeIntersected struct {
		RefreshTime time.Time
		StillIntersect bool 
	}
	type TracerouteIDsChan struct {
		Chan chan bool 
		Refreshed map[int64] RefreshedTimeIntersected
	}
	// TODO find a mechanism so that a reverse traceroute 
	// avoids asking for a traceroute that is being refreshed.

	nTraceroutesPerSource := 1000
	for {
		select {
		case <-done:
			return

		case <-refreshTicker.C:
			// Copy the data structures to clear it afterwards
			ripeTraceroutesToRefreshCopy := make(map[int64]struct{})
			traceroutesToRefreshCopy := make(map[int64]struct{})

			for k, v := range(ripeTraceroutesToRefresh) {
				ripeTraceroutesToRefreshCopy[k] = v
			}

			for k, v := range(traceroutesToRefresh) {
				traceroutesToRefreshCopy[k] = v
			}

			// tracerouteIDsPerSource := make(map[uint32][]int64)
			metaPerTracerouteIDCopy := make(map[int64]types.RefreshTracerouteMeta)
			tracerouteIDPerMetaCopy := make(map[types.RefreshTracerouteMeta]int64)
			for k, v := range(metaPerTracerouteID) {
				metaPerTracerouteIDCopy[k] = v
			}
			for k, v := range(tracerouteIDPerMeta) {
				tracerouteIDPerMetaCopy[k] = v
			}
			// First refresh MLab traceroutes
			go func() {
				if len(traceroutesToRefresh) > 0 {
					log.Infof("Refreshing MLab traceroutes %d ", len(traceroutesToRefresh))

					a.refreshMLabTraceroutes(traceroutesToRefreshCopy, metaPerTracerouteIDCopy, tracerouteIDPerMetaCopy)
				}
			} ()
			
			go func() {
				if len(ripeTraceroutesToRefresh) > 0 {
					log.Infof("Refreshing RIPE traceroutes %d ", len(ripeTraceroutesToRefresh))
					a.refreshRIPETraceroutes(nTraceroutesPerSource, 
						ripeTraceroutesToRefreshCopy, metaPerTracerouteIDCopy, tracerouteIDPerMetaCopy)
				}
				
			} ()
			
			// Reset the data structures
			traceroutesToRefresh = make(map[int64]struct{})
			ripeTraceroutesToRefresh = make(map[int64]struct{})
			metaPerTracerouteID = make(map[int64]types.RefreshTracerouteMeta)
			// weightPerTracerouteID = make(map[int64]int)
			 
			
		case tracerouteToRefresh := <- a.traceroutesToRefreshChan:
			tracerouteID := tracerouteToRefresh.TracerouteID
			log.Infof("Making refresh request for traceroute %d", tracerouteID)
			if tracerouteToRefresh.Meta.Platform == "ripe" {
				ripeTraceroutesToRefresh[tracerouteID] = struct{}{}
			} else {
				traceroutesToRefresh[tracerouteID] = struct{}{}
			} 
			metaPerTracerouteID[tracerouteID] = tracerouteToRefresh.Meta
			tracerouteIDPerMeta[types.RefreshTracerouteMeta{
				Src: tracerouteToRefresh.Meta.Src,
				Dst: tracerouteToRefresh.Meta.Dst,
			}] = tracerouteID
		}
	}
}

func (a * server) CheckIntersectingPath(req *pb.CheckIntersectionRequest) (*pb.CheckIntersectionResponse, error) {
	path, err := a.opts.trs.CheckIntersectingPath(req.TracerouteId, req.HopIntersection)
	return &pb.CheckIntersectionResponse{Path: path}, err
}

func (a *server) MarkTracerouteStale(req *pb.MarkTracerouteStaleRequest) (*pb.MarkTracerouteStaleResponse, error) {
	err := a.opts.trs.MarkTracerouteStale(req.IntersectIP, req.OldTracerouteId, req.NewTracerouteId)
	return &pb.MarkTracerouteStaleResponse{}, err
}

func (a *server) MarkTracerouteStaleSource(req *pb.MarkTracerouteStaleSourceRequest) (*pb.MarkTracerouteStaleSourceResponse, error) {
	err := a.opts.trs.MarkTracerouteStaleSource(req.Source)
	return &pb.MarkTracerouteStaleSourceResponse{}, err
}

func (a * server) InsertTraceroutes(req *pb.InsertTraceroutesRequest) (*pb.InsertTraceroutesResponse, error){

	
	for _, t := range(req.Traceroutes) {
		// Add source ASN
		t.SourceAsn = int32(a.opts.rt.GetI(t.Src))
	}

	// Bulk insert the traceroutes but can not store more than 20000 at a time
	log.Infof("Inserting %d traceroutes for the atlas...", len(req.Traceroutes))
	batchInsertSize := 5000
	retIds := []int64{}
	for k := 0; k < len(req.Traceroutes); k+=batchInsertSize {
		maxIndex := k + batchInsertSize
		if maxIndex > len(req.Traceroutes) {
			maxIndex = len(req.Traceroutes)
		}
		ids, err := a.opts.trs.StoreAtlasTracerouteBulk(req.Traceroutes[k:maxIndex])
		if err != nil {
			log.Error(err)
			return nil, err
		}
		for i, traceroute := range(req.Traceroutes[k:maxIndex]) {
			traceroute.Id = ids[i]
		}
		retIds = append(retIds, ids...)
	}

	if req.IsRunRrPings {
		log.Infof("Running RR pings for the atlas for %d traceroutes", len(req.Traceroutes))
		a.RunAtlasRRPings(&pb.RunAtlasRRPingsRequest{Traceroutes: req.Traceroutes, OnlyIntersections: req.OnlyIntersections})
		if req.IsRunRrIntersections {
			uuid := util.GenUUID()
			idsFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/atlas/resources/refresh/atlas_traceroutes_%s.ids", a.home, uuid)
			a.runRRIntersectionFromIds(idsFile, retIds)
		}
	}

	// Then refresh the atlas cache
	a.GetAvailableHopAtlasPerSource(&pb.GetAvailableHopAtlasPerSourceRequest{})
	return &pb.InsertTraceroutesResponse{Ids: retIds}, nil
}

// GetPathsWithToken satisfies the server interface
func (a *server) GetPathsWithToken(tr *pb.TokenRequest) (*pb.TokenResponse, error) {
	log.Debug("Looking for intersection from token: ", tr)
	req, err := a.tc.Get(tr.Token)
	if err != nil {
		log.Error(err)
		return &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_ERROR,
			Error: err.Error(),
		}, nil
	}
	a.tc.Remove(tr.Token)
	// Look up the source AS
	asn := a.opts.rt.GetI(req.Src)
	iq := types.IntersectionQuery{
		Addr:         req.Address,
		Dst:          req.Dest,
		Src:          req.Src,
		Stale:        req.Staleness, // In minutes
		IgnoreSource: req.IgnoreSource,
		IgnoreSourceAS: req.IgnoreSourceAs,
		SourceAS: asn,
		Alias:        req.UseAliases,
		UseAtlasRR: req.UseAtlasRr,
		Platforms: req.Platforms,
	}
	log.Debug("Looking for intesection for: ", req)


	// intersectionResponseChan := make(chan types.IntersectionResponseWithError, 1)
	// a.queryQueue <- intersectionQueryChannel{
	// 	query: ir,
	// 	response: intersectionResponseChan,
	// }

	// intersectionResponseWithError := <- intersectionResponseChan
	// intersectionResponse := intersectionResponseWithError.Ir
	// err = intersectionResponseWithError.Err
	// a.flyingIntersectQueries <- 0
	intersectionResponse, err := a.opts.trs.FindIntersectingTraceroute(iq)
	// <- a.flyingIntersectQueries 
	log.Debug("FindIntersectingTraceroute resp: ", intersectionResponse.Path)
	if err != nil {
		log.Debug("Found no intersection")
		if err != repo.ErrNoIntFound {
			log.Error(err)
			return nil, err
		}
		return &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_NONE_FOUND,
		}, nil
	}
	log.Debug("Got path: ", intersectionResponse.Path, " for token ", tr.Token)

	// Inform atlas that we have intersected
	refreshRequest := types.RefreshTracerouteIDChan {
		TracerouteID: intersectionResponse.TracerouteID,
		Channel: nil,
		Meta : types.RefreshTracerouteMeta {
			Src: intersectionResponse.Src,
			Dst: iq.Dst,
			DistanceToSource: intersectionResponse.DistanceToSource,
			IntersectionIP: intersectionResponse.Path.Address,
			Platform: intersectionResponse.Platform,	
			Timestamp: time.Unix(intersectionResponse.Timestamp, 0),	
		},
	}

	a.traceroutesToRefreshChan <- refreshRequest

	hasToBeRefreshed := false
	if a.useRefresh {
	// Check if the traceroute needs to be refreshed 
	stalenessSeconds := int(req.StalenessBeforeRefresh * 60)
	elapsedSinceMeasured := time.Now().Unix() - intersectionResponse.Timestamp
	if int(elapsedSinceMeasured) > stalenessSeconds {
		hasToBeRefreshed = true
		refreshStillIntersect := a.refreshTraceroute(intersectionResponse, iq)
		if refreshStillIntersect.IsRefreshed && refreshStillIntersect.IsStillIntersect {
			// Check for the new traceroute
			intersectionResponse, err = a.opts.trs.FindIntersectingTraceroute(iq)
			log.Debug("FindIntersectingTraceroute resp ", intersectionResponse.Path)
			if err != nil {
				log.Debug("Found no intersection")
				if err != repo.ErrNoIntFound {
					log.Error(err)
					return nil, err
				}
				return &pb.TokenResponse{
					Token: tr.Token,
					Type:  pb.IResponseType_NONE_FOUND,
				}, nil
			}
		} else {
			// If the traceroute has not been refreshed because it was not selected, 
			// just return a non exisiting intersection. 
			return &pb.TokenResponse{
				Token: tr.Token,
				Type:  pb.IResponseType_NONE_FOUND,
			}, nil
		}
	} 
	}
	if !hasToBeRefreshed {
		// The traceroute is still considered as non stale, so put it in the evaluation queue
		a.traceroutesToRefreshEvaluationChan <- types.RefreshTracerouteIDChan {
			TracerouteID: intersectionResponse.TracerouteID,
			Meta : types.RefreshTracerouteMeta {
				Src: intersectionResponse.Src,
				Dst: iq.Dst,
				DistanceToSource: intersectionResponse.DistanceToSource,
				IntersectionIP: intersectionResponse.Path.Address,
				Platform: intersectionResponse.Platform,	
				Timestamp: time.Unix(intersectionResponse.Timestamp, 0),	
			},
		}

	}
	
	

	var intr *pb.TokenResponse 
	if intersectionResponse.Platform == "mlab" {
		intr = &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_PATH,
			Path:  intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	} else if intersectionResponse.Platform == "ripe" {
		srcS , _ := util.Int32ToIPString(intersectionResponse.Src)
		intr = &pb.TokenResponse{
			Type: pb.IResponseType_PATH,
			Path: intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			SourceProbeId: int64(a.mapRIPEProbeID[srcS]),
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	} else {
		intr = &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_PATH,
			Path:  intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	}
	return intr, nil
}

// GetIntersectingPath satisfies the server interface
func (a *server) GetIntersectingPath(ir *pb.IntersectionRequest) (*pb.IntersectionResponse, error) {
	if ir.Staleness == 0 {
		ir.Staleness = 60
	}

	// Check if the hop if available in the atlas in memory. If it's the case, query the DB to retrieve it. 
	a.atlasHopsPerSourceLock.RLock()
	if _, ok := a.atlasHopsPerSource[ir.Dest]; !ok {
		a.atlasHopsPerSourceLock.RUnlock()
		iresp := &pb.IntersectionResponse{
			Type:  pb.IResponseType_ERROR,
			Error: repo.ErrNoIntFound.Error(),
		}
		
		return iresp, nil 
	}	

	if _, ok := a.atlasHopsPerSource[ir.Dest][ir.Address]; !ok {
		a.atlasHopsPerSourceLock.RUnlock()
		iresp := &pb.IntersectionResponse{
			Type:  pb.IResponseType_ERROR,
			Error: repo.ErrNoIntFound.Error(),
		}
		return iresp, nil 
	}

	a.atlasHopsPerSourceLock.RUnlock()


	asn := a.opts.rt.GetI(ir.Src)
	iq := types.IntersectionQuery{
		Addr:         ir.Address,
		Dst:          ir.Dest,
		Src:          ir.Src,
		Stale:        ir.Staleness, // In minutes
		Alias:        ir.UseAliases,
		IgnoreSource: ir.IgnoreSource,
		IgnoreSourceAS: ir.IgnoreSourceAs,
		SourceAS: asn,
		UseAtlasRR: ir.UseAtlasRr,
		Platforms: ir.Platforms, 
	}

	// intersectionResponseChan := make(chan types.IntersectionResponseWithError, 1)
	// a.queryQueue <- intersectionQueryChannel{
	// 	query: iq,
	// 	response: intersectionResponseChan,
	// }

	// intersectionResponseWithError := <- intersectionResponseChan
	// intersectionResponse := intersectionResponseWithError.Ir
	// err := intersectionResponseWithError.Err
	a.flyingIntersectQueries <- 0
	intersectionResponse, err := a.opts.trs.FindIntersectingTraceroute(iq)
	<- a.flyingIntersectQueries 
	log.Debug("FindIntersectingTraceroute resp ", intersectionResponse.Path)
	if err != nil {
		if err != repo.ErrNoIntFound {
			log.Error(err)
			return nil, err
		}
		token, err := a.tc.Add(ir)
		var iresp *pb.IntersectionResponse
		if err != nil {
			log.Error(err)
			iresp = &pb.IntersectionResponse{
				Type:  pb.IResponseType_ERROR,
				Error: err.Error(),
			}
		} else {
			iresp = &pb.IntersectionResponse{
				Type:  pb.IResponseType_TOKEN,
				Token: token,
			}
		}

		// go a.fillAtlas(ir.Address, ir.Dest, ir.Staleness)
		return iresp, nil
	}

	// Inform atlas that we have intersected
	refreshRequest := types.RefreshTracerouteIDChan {
		TracerouteID: intersectionResponse.TracerouteID,
		Channel: nil,
		Meta : types.RefreshTracerouteMeta {
			Src: intersectionResponse.Src,
			Dst: iq.Dst,
			DistanceToSource: intersectionResponse.DistanceToSource,
			IntersectionIP: intersectionResponse.Path.Address,
			Platform: intersectionResponse.Platform,	
			Timestamp: time.Unix(intersectionResponse.Timestamp, 0),	
		},
	}

	a.traceroutesToRefreshChan <- refreshRequest

	// Check if the traceroute needs to be refreshed 
	
	
	// now := time.Now().Unix()
	// fmt.Printf("%d\n", now)
	hasToBeRefreshed := false
	if a.useRefresh {
	stalenessSeconds := int(ir.StalenessBeforeRefresh * 60)
	elapsedSinceMeasured := time.Now().Unix() - intersectionResponse.Timestamp
	if int(elapsedSinceMeasured) > stalenessSeconds {
		hasToBeRefreshed = true
		refreshStillIntersect := a.refreshTraceroute(intersectionResponse, iq)
		if refreshStillIntersect.IsRefreshed && refreshStillIntersect.IsStillIntersect {
			// Check for the new traceroute
			intersectionResponse, err = a.opts.trs.FindIntersectingTraceroute(iq)
			log.Debug("FindIntersectingTraceroute resp ", intersectionResponse.Path)
			if err != nil {
				if err != repo.ErrNoIntFound {
					log.Error(err)
					return nil, err
				}
				token, err := a.tc.Add(ir)
				var iresp *pb.IntersectionResponse
				if err != nil {
					log.Error(err)
					iresp = &pb.IntersectionResponse{
						Type:  pb.IResponseType_ERROR,
						Error: err.Error(),
					}
				} else {
					iresp = &pb.IntersectionResponse{
						Type:  pb.IResponseType_TOKEN,
						Token: token,
					}
				}
				// go a.fillAtlas(ir.Address, ir.Dest, ir.Staleness)
				return iresp, nil
			}
		} else {
			// if the traceroute has not been refreshed, return intersection not found.
			token, err := a.tc.Add(ir)
			var iresp *pb.IntersectionResponse
			if err != nil {
				log.Error(err)
				iresp = &pb.IntersectionResponse{
					Type:  pb.IResponseType_ERROR,
					Error: err.Error(),
				}
			} else {
				iresp = &pb.IntersectionResponse{
					Type:  pb.IResponseType_TOKEN,
					Token: token,
				}
			}
			// go a.fillAtlas(ir.Address, ir.Dest, ir.Staleness)
			return iresp, nil
		}
	} 
	}
	if !hasToBeRefreshed {
		if a.useRefreshEvaluation {
			// The traceroute is still considered as non stale, so put it in the evaluation queue
			a.traceroutesToRefreshEvaluationChan <- types.RefreshTracerouteIDChan {
				TracerouteID: intersectionResponse.TracerouteID,
				Meta : types.RefreshTracerouteMeta {
					Src: intersectionResponse.Src,
					Dst: iq.Dst,
					DistanceToSource: intersectionResponse.DistanceToSource,
					IntersectionIP: intersectionResponse.Path.Address,
					Platform: intersectionResponse.Platform,	
					Timestamp: time.Unix(intersectionResponse.Timestamp, 0),	
				},
			}
		}
	}

	var intr *pb.IntersectionResponse
	if intersectionResponse.Platform == "mlab" {
		intr = &pb.IntersectionResponse{
			Type: pb.IResponseType_PATH,
			Path: intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			Timestamp: intersectionResponse.Timestamp,
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	} else if intersectionResponse.Platform == "ripe" {
		// srcS , _ := util.Int32ToIPString(intersectionResponse.Src)
		intr = &pb.IntersectionResponse{
			Type: pb.IResponseType_PATH,
			Path: intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			// SourceProbeId: int64(a.mapRIPEProbeID[srcS]),
			Timestamp: intersectionResponse.Timestamp,
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	} else {
		intr = &pb.IntersectionResponse{
			Type: pb.IResponseType_PATH,
			Path: intersectionResponse.Path,
			TracerouteId: intersectionResponse.TracerouteID,
			Src: intersectionResponse.Src,
			Platform: intersectionResponse.Platform,
			Timestamp: intersectionResponse.Timestamp,
			DistanceToSource: intersectionResponse.DistanceToSource,
		}
	}

	return intr, nil
}

func (a * server) refreshTraceroute(intersectionResponse types.IntersectionResponse, 
	intersectionRequest types.IntersectionQuery,
	) types.RefreshStillIntersect {
	
		// Has to refresh the traceroute
		refreshChan := make(chan types.RefreshStillIntersect, 1) 
		defer close(refreshChan)
		refreshRequest := types.RefreshTracerouteIDChan {
			TracerouteID: intersectionResponse.TracerouteID,
			Channel: refreshChan,
			Meta : types.RefreshTracerouteMeta {
				Src: intersectionResponse.Src,
				Dst: intersectionRequest.Dst,
				DistanceToSource: intersectionResponse.DistanceToSource,
				IntersectionIP: intersectionResponse.Path.Address,
				Platform: intersectionResponse.Platform,	
				Timestamp: time.Unix(intersectionResponse.Timestamp, 0),	
			},
		}
		a.traceroutesToRefreshChan <- refreshRequest
		isRefreshed := <-refreshChan
		return isRefreshed
}

func runRIPETraceroutes(sources [] string, nTraceroutesPerSource int, ripeAccount string, ripeKey string, waitMinutes int) (string, error) {
	// Run traceroutes to these sources using python script (easier to deal with rate limits)
	// Divide the credits across the different M-Lab sources
	// nTraceroutesPerSource := 1000 // 146 sites in theory atm
	home, err := os.UserHomeDir()
	uuid := util.GenUUID()
	log.Infof("UUID for RIPE traceroutes atlas %s", uuid)

	// Write the sources in a file so that they can be retrieved by the python atlas_service
	sourcesFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/algorithms/evaluation/resources/sources_%s.csv", home, uuid)
	f, err := os.Create(sourcesFile)
	if err != nil {
        return "", err
	}
	defer f.Close()
	for _, source := range(sources) {
		line := []byte(fmt.Sprintf("%s\n", source))
		_, err := f.Write(line)
		if err != nil {
			log.Error(err)
			return "", err
		}
	}
	f.Sync()
	cmd := fmt.Sprintf("cd %s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/ && python3 -m " + 
	"atlas.atlas_service --action=trace --n-traceroutes-per-source=%d --sources-file=%s --uuid=%s --ripe-key=%s --ripe-account=%s",
	home, nTraceroutesPerSource, sourcesFile, uuid, ripeKey, ripeAccount, 
	)

	_, err = util.RunCMD(cmd)
	// log.Info(stdout)
	if err != nil {
		log.Error(err)
		return "", err
	}
	time.Sleep(time.Duration(waitMinutes) * time.Minute)
	return uuid, nil  
}

func (a *server) fillAtlas(srcs [] uint32, req *pb.RunTracerouteAtlasToSourceRequest) error {
	// if there are none to run, don't
	if len(srcs) == 0 {
		return errors.New("No vantage points available to run traceroutes atlas") 
	}

	// The revtr source to which we send the traceroutes
	dest :=  req.Source
	destS, _ := util.Int32ToIPString(dest)
	req.WithRipe = true
	if req.WithRipe {
		go func () {
			if req.RipeAccount == "" && req.RipeKey == "" {
				// TODO change this into an error
				req.RipeAccount = a.ripeAccount
				req.RipeKey = a.ripeKey
			} 
			// Run RIPE Atlas traceroutes 
			uuid, err := runRIPETraceroutes([]string {destS}, 2000, req.RipeAccount, req.RipeKey, 10) // this wait 10 minutes to let RIPE traceroutes run on the platform. 
			// Try to fetch the results every X minutes after 10 with a hard deadline of 2 hours. 
			
			t := time.NewTicker(10 * time.Second)
			maxTryFetchTime := 30 // 30 minutes max to wait
			defer t.Stop()
			dumpFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/atlas/resources/atlas_%s.json",
			a.home, uuid)
			
			startFetch := time.Now()

			tryFetch:
			for {
				select {
				case <- t.C:
					if time.Now().Sub(startFetch).Minutes() > float64(maxTryFetchTime) {
						break tryFetch
					} 

					stderr, err := atlas.DumpRIPETraceroutes(uuid, dumpFile, req.RipeKey, req.RipeAccount)
					if err != nil {
						log.Infof("Results for RIPE traceroute %s not yet available", uuid)
						log.Errorf(stderr)
						continue
						
					} else {
						
						break tryFetch
					}
				}

			}
			// Now fetch the results and convert them to traceroutes
			ripeAtlasSourceASNFile := fmt.Sprintf("%s/go/src/github.com/NEU-SNS/ReverseTraceroute/cmd/ripeatlascontroller/resources/sources_asn.csv", a.home)
			traceroutes, err := ripe.FetchRIPEMeasurementsToTraceroutes(dumpFile,
				 ripeAtlasSourceASNFile, 
				 "ripe")
			if err != nil {
				log.Error(err)
			}

			a.InsertTraceroutes(&pb.InsertTraceroutesRequest{
				Traceroutes: traceroutes,
				IsRunRrPings: true,
				IsRunRrIntersections: true,
			})


		} ()
	}
	
	log.Debug("Sources to fill atlas for ", dest, " ", srcs, " ", len(srcs), " new sources.")
	var traces []*dm.TracerouteMeasurement
	for _, src := range srcs {
		if src == dest {
			continue
		}
		curr := &dm.TracerouteMeasurement{
			Src:        src,
			Dst:        dest,
			Timeout:    60,
			Wait:       "2",
			Attempts:   "1",
			LoopAction: "1",
			Loops:      "3",
			CheckCache: false,
			SaveDb: true, // This will be saved into the traceroute atlas DB
			Staleness:  60 * 24, // One day
			FromRevtr: true,
		}
		traces = append(traces, curr)
	}
	log.Debug("Running ", len(traces), " traces")
	// res := a.limit.ReserveN(time.Now(), len(traces))
	// if !res.OK() {
	// 	log.Error("Burst too high for atlas traceroutes")
	// 	return 
	// }
	// delay := res.Delay()
	// if delay == rate.InfDuration {
	// 	a.curr.Remove(dest, srcs)
	// 	log.Error("Cant run traces to fill atlas")
	// 	return 
	// }
	// log.Debug("Sleeping for ", delay)
	// time.Sleep(delay)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	defer cancel()
	tracerouteGauge.Add(float64(len(traces)))
	st, err := a.opts.cl.Traceroute(ctx, &dm.TracerouteArg{Traceroutes: traces})
	if err != nil {
		log.Error(err)
		// a.curr.Remove(dest, srcs)
		return err
	}
	traceroutes := [] * dm.Traceroute{}
	for {
		t, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		hopsWithSrc := [] * dm.TracerouteHop{{
			ProbeTtl: 0,
			Addr: t.Src,
		}}
		hops := t.GetHops()
		if len(hops) == 0 {
			continue
		}
		if hops[len(hops)-1].Addr != t.Dst {
			srcS := util.Int32ToIP(t.Src)
			dstS := util.Int32ToIP(t.Dst)
			log.Infof("Traceroute from %s to %s did not reach destination", srcS, dstS)
			continue
		}
		hopsWithSrc = append(hopsWithSrc, hops...)
		t.Hops = hopsWithSrc
		t.Platform = "mlab"
		// Setting the source AS of the M-Lab vantage point
		asn := a.opts.rt.GetI(t.Src)
		t.SourceAsn = int32(asn)
		traceroutes = append(traceroutes, t)
	}
	if len(traceroutes) > 0 {
		_, err := a.InsertTraceroutes(&pb.InsertTraceroutesRequest{
			Traceroutes: traceroutes,
			IsRunRrPings: true,
			IsRunRrIntersections: true,
		})
		if err != nil {
			log.Error(err)
			return err
		}
		ids, err := a.opts.trs.StoreAtlasTracerouteBulk(traceroutes)
		for i, t := range(traceroutes) {
			t.Id = ids[i]
		}
	}

	return nil
}



func (a *server) getSrcs() []uint32 {
	// This returns the current M-Lab VPs available for running traceroutes 
	vps, err := a.opts.vps.GetVPs()
	if err != nil {
		return nil
	}
	sites := make(map[string]*vppb.VantagePoint)
	for _, vp := range vps.GetVps() {
		if vp.Trace {
			sites[vp.Site] = vp
		}
	}
	var srcs []uint32
	for _, vp := range sites {
		srcs = append(srcs, vp.Ip)
	}
	return srcs
}

// SrcDst is a struct to match ips with their traceroutes
type SrcDst struct {
	src uint32 
	dst uint32
}

func (a *server) RunTracerouteAtlasToSource(req *pb.RunTracerouteAtlasToSourceRequest)  (*pb.RunTracerouteAtlasToSourceResponse, error) {
	// This runs M-Lab traceroutes to the source when an external source has just been added to the system. 

	// Get the M-Lab sources.
	atlasMLabVPs := a.getSrcs()
	// Hack for the peering experiment
	// peering anycast source is 
	// sourceS, _ := util.Int32ToIPString(req.Source)
	// Run traceroutes + RR Pings
	err := a.fillAtlas(atlasMLabVPs, req)
	if err != nil {
		return nil, err
	}

	return &pb.RunTracerouteAtlasToSourceResponse{}, nil
}


func (a *server) RunAtlasRRPings(req *pb.RunAtlasRRPingsRequest) (*pb.RunAtlasRRPingsResponse, error) {
	// All the traceroutes must be traceroutes to sources (M-Lab nodes)
	// First create data structure to be able to 
	// (1) Easily find the src, dst pair associated to an IP
	// (2) The potential aliases for this IP. 
	traceroutes := req.Traceroutes
	srcDstByHop := map[uint32][]SrcDst{}
	srcByHop := map[uint32]map[uint32]struct{}{}
	hopsBySrcDst := map[SrcDst][]uint32{}
	tracerouteIDBySrcDst := map[SrcDst]int64{}
	log.Infof("Preparing running RR pings for %d traceroutes", len(req.Traceroutes))
	for i, traceroute := range(traceroutes) {
		if i % 10000 == 0 {
			log.Infof("Prepared RR pings for %d traceroutes", i)
		}
		// Destinations of these traceroutes are revtr sources.
		hops := []uint32{}
		for _, hop := range(traceroute.Hops) {
			hops = append(hops, hop.Addr)
			if hop.Addr == traceroute.Dst {
				continue
			}
			if _, ok :=  srcDstByHop[hop.Addr] ; !ok {
				srcDstByHop[hop.Addr] = []SrcDst{{
					src: traceroute.Src,
					dst: traceroute.Dst,
				}}	
			} else {
				srcDstByHop[hop.Addr] = append(srcDstByHop[hop.Addr], SrcDst{
					src: traceroute.Src,
					dst: traceroute.Dst,
				})	
			}

			if _, ok := srcByHop[hop.Addr] ; !ok {
				srcByHop[hop.Addr] = make(map[uint32]struct{})
			}
			srcByHop[hop.Addr][traceroute.Dst] = struct{}{}
		}
		srcDstKey := SrcDst{src: traceroute.Src, dst: traceroute.Dst}
		hopsBySrcDst[srcDstKey] = hops
		tracerouteIDBySrcDst[srcDstKey] = traceroute.Id
	}

	// Get the ip clusters of these aliases, 
	// there is no point probing those not belonging to any alias set.
	ipsQuery := []uint32{}
	for ip, _ := range(srcDstByHop) {
		ipsQuery = append(ipsQuery, ip)
	}
	// aliasSetByIP, _, err := a.opts.trs.GetIPAliases(ipsQuery)
	// if err != nil {
	// 	log.Error(err)
	// 	panic(err)
	// }
	
	
	// Run the atlas RR pings. 
	// Because we are assuming destination based routing, only run one RR ping per source. 
	pmsChan := make(chan *dm.PingMeasurement, 1000000000)
	var wg sync.WaitGroup
	log.Infof("Finding RR spoofers for %d ip addresses", len(srcByHop))
	for ip, srcs := range(srcByHop) {
		wg.Add(1)
		
		// log.Debugf("Getting spoofers for %s", ipS)
		go func(ipP uint32, srcsP map[uint32]struct{}) {
			defer wg.Done()

			// if _, ok := aliasSetByIP[ip]; !ok {
			// 	// The IP is not in any alias set, so we will not be able to find any alias, so skip.
			// 	continue
			// }

			traceroutesContainingHop := srcDstByHop[ipP]
			if req.OnlyIntersections && len(traceroutesContainingHop) < 2 {
				// Not an intersection point
				return 
			}
			ipS, _ := util.Int32ToIPString(ipP)
			rrSpoofers, err := a.opts.rkcl.GetRRSpoofers(ipS, 5, environment.IngressCover)
			if err != nil {
				log.Error(err)
				return
			}
			
			for src, _ := range(srcsP) {
				pm := &dm.PingMeasurement{
					Src:        src, // src is the M-Lab node
					Dst:        ipP,
					Sport: strconv.Itoa(20000),
					Timeout:    20,
					Count:      "1",
					Staleness:  7200,
					CheckCache: true,
					Spoof:      false,
					RR:         true,
					Label:      ATLAS_PING_LABEL,
					SaveDb:     false,
					FromRevtr: true,
					// SpoofTimeout: 10,
					// IsAbsoluteSpoofTimeout: true,
				}
				pmsChan <- pm
				srcS, _ := util.Int32ToIPString(src)
				for _, rrSpoofer := range(rrSpoofers) {
					spooferI, _ := util.IPStringToInt32(rrSpoofer.Ip)
					
					sPort := rand.Intn(environment.MaxSport - environment.MinSport) + environment.MinSport
					// Send spoofed packet from spooferI to ip received by dstS (M-Lab node)
					pm := &dm.PingMeasurement{
						Src:        spooferI,
						Dst:        ipP,
						SAddr:      srcS,
						Sport: strconv.Itoa(sPort),
						Timeout:    20,
						Count:      "1",
						Staleness:  7200,
						CheckCache: true,
						Spoof:      true,
						RR:         true,
						Label:      ATLAS_PING_LABEL, 
						SaveDb:     false,
						SpoofTimeout: 30,
						FromRevtr: true,
						// IsAbsoluteSpoofTimeout: true,
					}
					pmsChan <- pm
				}
			}

		} (ip, srcs)
		
	} 
	wg.Wait()
	close(pmsChan)
	pms := []*dm.PingMeasurement {}
	for pm := range(pmsChan) {
		pms = append(pms, pm)
	}

	log.Infof("Shuffling %d RR ping to send", len(pms))

	rand.Shuffle(len(pms), func(i, j int) { pms[i], pms[j] = pms[j], pms[i] })
	type HopsTimestamp struct {
		hops [] uint32
		date time.Time
	}
	rrSegmentsTrIDHop := map[SrcDst]map[uint32]map[uint32]HopsTimestamp{}
	// Form batch of pings to not overload the controller 
	batchesRR := [][]*dm.PingMeasurement{}
	batch := []*dm.PingMeasurement{}
	batchSize := 500000
	for _, pm := range(pms) {
		batch = append(batch, pm)
		if len(batch) == batchSize {
			batchesRR = append(batchesRR, batch)
			batch = nil
		}
	}
	if len(batch) > 0 {
		batchesRR = append(batchesRR, batch)
	}

	

	for i, batch := range(batchesRR) {
		// TODO compute the number of pings to be sent by vp to adjust the timeout
		pingsByVP := map[uint32]uint32 {}
		for _, p := range(batch) {
			var vpI uint32
			if p.Spoof {
				vp := p.SAddr
				vpI, _ = util.IPStringToInt32(vp)
			} else {
				vpI = p.Src
			}
			if _, ok := pingsByVP[vpI]; !ok {
				pingsByVP[vpI] = 0
			}
			pingsByVP[vpI] += 1 
		}
		
		// Compute maximum number of pings per VP to adjust timeout
		maxNPings := uint32(0)
		for _, nPings := range(pingsByVP) {
			if nPings > maxNPings {
				maxNPings = nPings
			}
		}
		theoreticalSeconds := maxNPings / 50 // 100 is the number of pps so, be conservative if the node is busy

		if len(batch) > 100 {
			// Small hack here to determine a back of the enveloppe estimation of the measurement time.
			// Basically, check whether we are in an huge batch of pings of just a small RR pings from the revtr
			// system 
			batch[0].IsAbsoluteSpoofTimeout = true
			batch[0].SpoofTimeout = theoreticalSeconds +  10 // time to gather last responses + processing time of the system 	
		}
		
		// Find the maximum number of packets per  
		log.Infof("Sending %d batch of %d (spoofed) RR pings out of %d with timeout %d, absolute %t", i, 
		len(batch), len(batchesRR), batch[0].SpoofTimeout, batch[0].IsAbsoluteSpoofTimeout)
		st, err :=  a.opts.cl.Ping(context.Background(), &dm.PingArg{
			Pings: batch,
		})
		if err != nil {
			log.Error(err)
		}
		
	
		// insertTrIDRRHopAlias := map[types.TrIDRRHopAlias] int{}
	
		
	
		for {
			p, err := st.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				break
			}
	
			if len(p.GetResponses()) > 0 {
				// Look if we got some RR hops after the destination
				rr := p.GetResponses()[0].RR
				rrAfterDst := rr
				hasDstStamp := false
				for i, hop := range(rr) {
					if hop == p.Dst {
						// Attach the RR IPs after the destination to the traceroute  
						rrAfterDst = rrAfterDst[i:]
						hasDstStamp = true
						break
					}
				}
				if !hasDstStamp {
					continue
				}
				// Search if the RR hop after the destination is an alias of the traceroute
				for _, srcDstKey := range(srcDstByHop[p.Dst]) {
					vp := srcDstKey.dst // dst is the M-Lab node 
					if p.Src != vp {
						// Find the corresponding traceroute for the RR ping.
						// The RR ping was not for that source, so skip
						continue
					}
					if _, ok := rrSegmentsTrIDHop[srcDstKey]; !ok {
						rrSegmentsTrIDHop[srcDstKey] = map[uint32]map[uint32]HopsTimestamp{}
					}
					if _, ok := rrSegmentsTrIDHop[srcDstKey][p.Dst]; !ok {
						rrSegmentsTrIDHop[srcDstKey][p.Dst] = map[uint32]HopsTimestamp{}
					}
					var start time.Time
					if p.Start == nil {
						start = time.Now()
					} else {
						start = time.Unix(p.Start.Sec, p.Start.Usec*1000)
					}
					rrSegmentsTrIDHop[srcDstKey][p.Dst][p.SpoofedFrom] = HopsTimestamp{
						hops: rrAfterDst,
						date: start,
					}
					
					// trHops := hopsBySrcDst[srcDstKey] 
					// for _, trHop := range(trHops) {
					// 	// Look if the trHop
					// 	if trAliasID, ok := aliasSetByIP[trHop]; ok {
					// 		for rrHopIndex, rrHop := range(rrAfterDst) {
					// 			if trHop == rrHop {
					// 				continue
					// 			}
					// 			if rrAliasID, ok:= aliasSetByIP[rrHop]; ok {
					// 				if trAliasID == rrAliasID {
					// 					insertTrIDRRHopAlias[types.TrIDRRHopAlias{
					// 						ID: tracerouteIDBySrcDst[srcDstKey],
					// 						RRHop: rrHop,
					// 						Alias: trHop,
					// 						RRHopIndex: rrHopIndex,
					// 					}] = 1
					// 				}
					// 			}
					// 		}	
					// 	}
							
				}	 
			}
		}
	}
	
	insertTrIDRRHopList := []types.TrIDRRHop {}
	for srcDst, trHops := range(hopsBySrcDst) {
		for _, trHop := range(trHops) {
			for spoofer, rrHops := range(rrSegmentsTrIDHop[srcDst][trHop]) {
				for rrHopIndex, rrHop := range(rrHops.hops) {
					insertTrIDRRHopList = append(insertTrIDRRHopList, types.TrIDRRHop{
						ID: tracerouteIDBySrcDst[srcDst],
						PingID: int64(spoofer),
						RRHop: rrHop,
						TrHop: trHop,
						RRHopIndex: rrHopIndex,
						Date: rrHops.date,
					})
				}
				
			} 
		}
	}

	// for srcDst, hopRRHops := range(rrSegmentsTrIDHop){
	// 	trIDRRHops := rebuildTraceFromSegments(hopRRHops, hopsBySrcDst[srcDst])
	// 	for i, trIDRRHop :=range(trIDRRHops) {
	// 		trIDRRHops[i] = types.TrIDRRHop{
	// 			ID: tracerouteIDBySrcDst[srcDst],
	// 			RRHop: trIDRRHop.RRHop,
	// 			TrHopIntersection: trIDRRHop.TrHopIntersection,
	// 			TrHopIndex: trIDRRHop.TrHopIndex,
	// 		}
	// 		insertTrIDRRHopList = append(insertTrIDRRHopList, trIDRRHops[i])
	// 	}
	// }
	sort.Stable(types.ByTracerouteID(insertTrIDRRHopList))
	// insertTrIDRRHopList = types.RemoveDuplicateValues(insertTrIDRRHopList)
	if len(insertTrIDRRHopList) > 0 {

		// Batch the number of inserts to have a reasonable query length
		for i := 0; i < len(insertTrIDRRHopList) ; i += 20000 {
			var nextIndex int
			if i + 20000 > len(insertTrIDRRHopList) {
				nextIndex = len(insertTrIDRRHopList)
			} else {
				nextIndex = i+20000
			}
			err := a.opts.trs.StoreTrRRHop(insertTrIDRRHopList[i:nextIndex])
			if err != nil {
				panic(err)
			}
		}
		
	}

	return &pb.RunAtlasRRPingsResponse{}, nil
}



func (a *server) GetAvailableHopAtlasPerSource(req *pb.GetAvailableHopAtlasPerSourceRequest) (*pb.GetAvailableHopAtlasPerSourceResponse, error) {
	
	log.Infof("Loading available hops in memory...")
	hopsPerSource, err := a.opts.trs.GetAvailableHopAtlasPerSource()
	if err != nil {
		return nil, err
	}
	log.Infof("Loading available hops in memory...done")
	a.atlasHopsPerSourceLock.Lock()
	defer a.atlasHopsPerSourceLock.Unlock()
	a.atlasHopsPerSource = hopsPerSource
	log.Infof("Found atlas traceroutes for %d sources", len(a.atlasHopsPerSource))
	hopsPerSourceResponse := map[uint32]*pb.AtlasHops{}
	 
	for dst, hops := range(hopsPerSource) {
		atlasHops := []uint32{}
		for hop, _ := range(hops){
			atlasHops = append(atlasHops, hop)
		}
		hopsPerSourceResponse[dst] = &pb.AtlasHops{
			Hops: atlasHops,
		}
	}
	return &pb.GetAvailableHopAtlasPerSourceResponse{
		HopsPerSource:hopsPerSourceResponse, 
	}, nil 
}



type runningTraces struct {
	mu        *sync.Mutex
	dstToSrcs map[uint32][]uint32
}

func newRunningTraces() runningTraces {
	return runningTraces{
		mu:        &sync.Mutex{},
		dstToSrcs: make(map[uint32][]uint32),
	}
}

func (rt runningTraces) Check(ip uint32) ([]uint32, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	srcs, ok := rt.dstToSrcs[ip]
	return srcs, ok
}

func (rt runningTraces) Remove(ip uint32, done []uint32) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if running, ok := rt.dstToSrcs[ip]; ok {
		checked := make(map[uint32]bool)
		for _, r := range running {
			checked[r] = true
		}
		for _, d := range done {
			checked[d] = false
		}
		var new []uint32
		for k, v := range checked {
			if v {
				new = append(new, k)
			}
		}
		if len(new) > 0 {
			rt.dstToSrcs[ip] = new
		} else {
			delete(rt.dstToSrcs, ip)
		}
	}
}

// UInt32Slice is for sorting uint32s
type UInt32Slice []uint32

func (u UInt32Slice) Len() int           { return len(u) }
func (u UInt32Slice) Less(i, j int) bool { return u[i] < u[j] }
func (u UInt32Slice) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

func (rt runningTraces) TryAdd(ip uint32, dsts []uint32) []uint32 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var merged []uint32
	var added []uint32
	if old, ok := rt.dstToSrcs[ip]; ok {
		sort.Sort(UInt32Slice(old))
		sort.Sort(UInt32Slice(dsts))
		var i, j int
		for i < len(old) && j < len(dsts) {
			switch {
			case old[i] < dsts[j]:
				merged = append(merged, old[i])
				i++
			case old[i] > dsts[j]:
				merged = append(merged, dsts[j])
				added = append(added, dsts[j])
				j++
			default:
				merged = append(merged, old[i])
				i++
				j++
			}
		}
		for i < len(old) {
			merged = append(merged, old[i])
			i++
		}
		for j < len(dsts) {
			merged = append(merged, dsts[j])
			added = append(added, dsts[j])
			j++
		}
		rt.dstToSrcs[ip] = merged
		return added
	}
	rt.dstToSrcs[ip] = dsts
	return dsts
}

type tokenCache struct {
	ca Cache
	// Should only be accessed atomicaly
	nextID uint32
}

func (tc *tokenCache) Add(ir *pb.IntersectionRequest) (uint32, error) {
	if ir == nil {
		return 0, fmt.Errorf("Cannot cache nil")
	}
	new := atomic.AddUint32(&tc.nextID, 1)
	tc.ca.Add(fmt.Sprintf("%d", new), *ir)
	return new, nil
}

func (tc *tokenCache) Get(id uint32) (*pb.IntersectionRequest, error) {
	it, ok := tc.ca.Get(fmt.Sprintf("%d", id))
	if !ok {
		return nil, fmt.Errorf("Failed to get cached token for id: %v", id)
	}

	if ir, ok := it.(pb.IntersectionRequest); ok {
		return &ir, nil
	}
	return nil, fmt.Errorf("Untknown type cached in token cache")
}

func (tc *tokenCache) Remove(id uint32) {
	tc.ca.Remove(fmt.Sprintf("%d", id))
}

type cacheError struct {
	id uint32
}

func (ce cacheError) Error() string {
	return fmt.Sprintf("No token registered for id: %d", ce.id)
}

func newTokenCache(ca Cache) *tokenCache {
	tc := &tokenCache{
		ca: ca,
	}
	return tc
}
