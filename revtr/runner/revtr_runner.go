package runner

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	apb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	ipoptions "github.com/NEU-SNS/ReverseTraceroute/ipoptions"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/radix"
	rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/client"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	iputil "github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	rt "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	stringutil "github.com/NEU-SNS/ReverseTraceroute/util/string"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"golang.org/x/net/context"
)

type optionSet struct {
	ctx context.Context
	cm  clustermap.ClusterMap
	cl  client.Client
	at  at.Atlas
	vps vpservice.VPSource
	as  types.AdjacencySource
	ca  types.Cache
	rkc rk.RankingSource
	// Spoofers selection technique
	// rrvpselect environment.RRVPsSelectionTechnique

	// In minutes 
	cacheTime int

	ip2as *radix.BGPRadixTree
}

// RunOption configures how run will behave
type RunOption func(*optionSet)

// WithCacheTime runs revtrs with the cacheTime ca
func WithCacheTime(cacheTime int) RunOption {
	return func(os *optionSet){
		os.cacheTime=cacheTime
	}
}

// WithCache runs revtrs with the cache ca
func WithCache(ca types.Cache) RunOption {
	return func(os *optionSet) {
		os.ca = ca
	}
}

// WithContext runs revtrs with the context c
func WithContext(c context.Context) RunOption {
	return func(os *optionSet) {
		os.ctx = c
	}
}

// WithClusterMap runs the revts with the clustermap cm
func WithClusterMap(cm clustermap.ClusterMap) RunOption {
	return func(os *optionSet) {
		os.cm = cm
	}
}

// WithClient runs the revts with the client cl
func WithClient(cl client.Client) RunOption {
	return func(os *optionSet) {
		os.cl = cl
	}
}

// WithAtlas runs the revtrs with the atlas at
func WithAtlas(at at.Atlas) RunOption {
	return func(os *optionSet) {
		os.at = at
	}
}

// WithVPSource runs the revtrs with the vpservice vps
func WithVPSource(vps vpservice.VPSource) RunOption {
	return func(os *optionSet) {
		os.vps = vps
	}
}

// WithAdjacencySource runs the revtrs with the adjacnecy source as
func WithAdjacencySource(as types.AdjacencySource) RunOption {
	return func(os *optionSet) {
		os.as = as
	}
}

func WithRKClient(rkc rk.RankingSource) RunOption {
	return func (os *optionSet) {
		os.rkc = rkc
	}
}

func WithIP2AS(ip2as *radix.BGPRadixTree) RunOption {
	return func (os *optionSet) {
		os.ip2as = ip2as
	}
}

func logRevtr(revtr *rt.ReverseTraceroute) log.Logger {
	return log.WithFieldDepth(log.Fields{
		"Revtr": revtr.LogStr,
	}, 1)
}

type runner struct {
}

// Runner is the interface for running reverse traceroutes
type Runner interface {
	Run([]*rt.ReverseTraceroute, ...RunOption) <-chan *rt.ReverseTraceroute
}

// New creates a new Runner
func New() Runner {
	return new(runner)
}

func (r *runner) Run(revtrs []*rt.ReverseTraceroute,
	opts ...RunOption) <-chan *rt.ReverseTraceroute {
	optset := &optionSet{}
	for _, opt := range opts {
		opt(optset)
	}
	if optset.ctx == nil {
		optset.ctx = context.Background()
	}
	rc := make(chan *rt.ReverseTraceroute, len(revtrs))
	batch := &rtBatch{}
	batch.opts = optset
	batch.wg = &sync.WaitGroup{}
	batch.wg.Add(len(revtrs))

	for _, revtr := range revtrs {
		logRevtr(revtr).Debug("Running ", revtr)
		go batch.run(revtr, rc)
	}
	go func() {
		batch.wg.Wait()
		close(rc)
	}()
	return rc
}

type step func(*rt.ReverseTraceroute) step

type rtBatch struct {
	opts *optionSet
	wg   *sync.WaitGroup
}

func (b *rtBatch) initialStep(revtr *rt.ReverseTraceroute) step {
	if revtr.BackoffEndhost {
		return b.backoffEndhost
	}
	// Hack for debugging timestamp
	// return b.timestamp
	return b.trToSource
}

func (b *rtBatch) backoffEndhost(revtr *rt.ReverseTraceroute) step {
	next := b.assumeSymmetric(revtr)
	if revtr.Reaches(b.opts.cm) {
		revtr.StopReason = rt.Trivial
		revtr.EndTime = time.Now()
	}
	if next == nil {
	}
	logRevtr(revtr).Debug("Done backing off")
	return next
}

func (b *rtBatch) CheckTracerouteStaleness(revtr *rt.ReverseTraceroute, it intersectingTR) (int64, error) {
	// DEPRECATED
	
	// Re-build the measurement from intersetingTR
	var tracerouteMeasurement datamodel.TracerouteMeasurement
	if it.platform == "ripe" { 
		tracerouteMeasurement = datamodel.BuildRIPETraceroute(revtr.Src, []int{int(it.ripeProbeID)}, "", "")
	} else if it.platform == "mlab" {
		src, _ := util.IPStringToInt32(it.src)
		dst, _ := util.IPStringToInt32(revtr.Src)
		tracerouteMeasurement = datamodel.TracerouteMeasurement{
			Src:        src,
			Dst:        dst,
			CheckCache: true,
			CheckDb:    false,
			SaveDb:     false,
			Staleness:  revtr.AtlasOpts.StalenessBeforeRefresh,
			Timeout:    60,
			Wait:       "2",
			Attempts:   "1",
			LoopAction: "1",
			Loops:      "3",
			FromRevtr:  true,
		}
	}
	trToInsert := [] *datamodel.Traceroute {}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	st, err := b.opts.cl.Traceroute(ctx, &datamodel.TracerouteArg{
		Traceroutes: [] *datamodel.TracerouteMeasurement{&tracerouteMeasurement},
	})
	if err != nil {
		log.Error(err)
		return 0, err
	}
	
	defer cancel()
	if err != nil {
		log.Error(err)
		return 0, fmt.Errorf("Failed to run traceroute: %v", err)
	}
	for {
		trace, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			log.Errorf("Error running traceroute from %s to %s", it.src, revtr.Src)
			return  0, fmt.Errorf("Error running traceroute: %v", err)
		}
		if trace.Error != "" {
			return 0, tracerouteError{trace: trace}
		}
		trToInsert = append(trToInsert, trace)
		
	}
	// Insert the traceroute into the atlas and run RR pings if needed
	// For now, always re-run the RR pings.
	ids, err := b.opts.at.InsertTraceroutes(ctx, trToInsert, true, false)
	if err != nil {
		log.Error(err)
	}
	id := ids[0]
	if len(ids) > 1{
		// More than one traceroute inserted, impossible
		err := "More than one rerun traceroute inserted"
		log.Error(err)
		return 0, fmt.Errorf(err)
	}

	// Run the python algorithm that fills up the between intersections.
	// Better would be to have in go, but... painful to write.
	cmd := fmt.Sprintf("cd /home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/ && python3 -m " + 
	"atlas.record_route --mode=single --id=%d",
	 ids[0])
	_, err = util.RunCMD(cmd)	
	if err != nil {
		log.Error(err)
		return 0, err
	}

	return id, nil
}

func (b *rtBatch) trToSource(revtr *rt.ReverseTraceroute) step {
	if revtr.AtlasOpts.UseAtlas {
		revtr.Stats.TRToSrcRoundCount++
		start := time.Now()
		defer func() {
			done := time.Now()
			dur := done.Sub(start)
			revtr.Stats.TRToSrcDuration += dur
		}()
		var addrs []uint32
		for _, hop := range revtr.CurrPath().LastSeg().Hops() {
			// [A, B, C, D_sym]
			addr, _ := util.IPStringToInt32(hop)
			if iputil.IsPrivate(net.ParseIP(hop)) {
				continue
			}
			addrs = append(addrs, addr)
		}
		
			hops, tokens, err := intersectingTraceroute(revtr.Src, revtr.Dst, addrs,
				b.opts.at, b.opts.cm, revtr.AtlasOpts)
			if err != nil {
				// and error occured trying to find intersecting traceroutes
				// move on to the next step
				return b.recordRoute
			}
			if tokens != nil {
				revtr.Tokens = tokens
				// received tokens, move on to next step and try later
				return b.recordRoute
			}
			if len(hops.hops) == 0 {
				revtr.Tokens = tokens
				// no hops found move on
				return b.recordRoute
			}
			// Check if traceroute is stale
			// if hops.date.Add(time.Duration(revtr.AtlasOpts.IntersectWithoutRerun) * time.Minute).Before(time.Now()) {
				// The traceroute is stale, so must re-run it before confirming 
				// Mark the traceroute as stale (and all the ones that were sharing IPs before the intersection)
				
				// The logic is the following:
				// Re-run the traceroute. Check if it still intersects. 
				// If it is the case, return 
				// If not, put this traceroute as stale and check another 
				// intersecting traceroute exists
				// newTracerouteID, err := b.CheckTracerouteStaleness(revtr, hops)
				// if err != nil {
				// 	log.Error(err)
				// 	// Could not rerun traceroute, so something bad happened?
				// 	return nil
				// }
				// intersectIP , _ := util.IPStringToInt32(hops.hops[0])
				// b.opts.at.MarkTracerouteStale(context.Background(), intersectIP, hops.tracerouteID, newTracerouteID)

				// // We should have either a fresh traceroute or no traceroute at all, so no risk of infinite loops
				// return b.trToSource

			// }

			logRevtr(revtr).Debug("Creating TRToSrc seg: ", hops, " ", revtr.Src, " ", hops.addr)
			revtr.TracerouteID = hops.tracerouteID
			segment := rt.NewTrtoSrcRevSegment(hops.hops, hops.hopTypes, revtr.Src, hops.addr, hops.tracerouteID)
			if !revtr.AddBackgroundTRSegment(segment, b.opts.cm) {
				// Ouput failed revtr in a file to be able to debug afterwards
				log.Errorf("Failed to add TR segment for revtr src: %s dst: %s, traceroute ID %d",
				 revtr.Src, revtr.Dst, revtr.TracerouteID)
				revtr.FailReason = "Failed to add a TR segment"
				return nil  
			}
			if revtr.Reaches(b.opts.cm) {
				return nil
			}
			srcI, _ := util.IPStringToInt32(revtr.Src)
			dstI, _ := util.IPStringToInt32(revtr.Dst)
			panic(fmt.Sprintf("Added a TR to source but the revtr didn't reach from %d to %d, traceroute ID %d", srcI, dstI, revtr.TracerouteID))
	} else {
		// Do not use the atlas
		return b.recordRoute
	}
	
}

type ipstr uint32

func (ips ipstr) String() string {
	s, _ := util.Int32ToIPString(uint32(ips))
	return s
}

func (b *rtBatch) spoofAndUpdate(target string, vps []string, revtr *rt.ReverseTraceroute) (step, bool){
	revtr.AddRankedSpoofers(target, vps)
	revtr.Stats.SpoofedRRProbes += len(vps)
	rrs, err := issueSpoofedRR(revtr, revtr.Src, target, vps, revtr.AtlasOpts.Staleness, b.opts.cl, b.opts.cm, revtr.UseCache)
	if err != nil {
		logRevtr(revtr).Error(err)
	}
	// Match responses with their IDs to be able to retrieve them in the DB
	revtr.MatchSpoofedResponses(target, rrs, vps)
	
	// we got some responses, even if there arent any addresses
	// in the responses make the target as responsive
	if len(rrs) > 0 {
		revtr.MarkResponsiveRRSpoofer(target)
	} else {
		// track unresponsiveness
		revtr.AddUnresponsiveRRSpoofer(target, len(vps))
		cacheKey := revtr.Src + "_" + target
		if revtr.UseCache {
			b.opts.ca.Set(cacheKey, "dummy", time.Duration(b.opts.cacheTime) * time.Minute)
		}
		
		// logRevtr(revtr).Debug("SET DUMMY VALUE !")
	}
	var segs []rt.Segment
	// create segments for all the hops in the responses
	for _, rr := range rrs {
		logRevtr(revtr).Debug("Creating segment for ", rr)
		for i, hop := range rr.Hops {
			if hop == "0.0.0.0" {
				continue
			}
			// rr = [A, B, C]
			// segs += [A], [A,B], [A,B,C].
			segs = append(segs,
				rt.NewSpoofRRRevSegment(rr.Hops[:i+1], revtr.Src, target, rr.VP, rr.MeasurementID, rr.FromCache))
		}
	}
	if len(segs) == 0 {
		// so segments created
		return nil, false
	}
	// try to add them
	if !revtr.AddSegments(segs, b.opts.cm) {
		//Couldn't add anything
		// continue
		return nil, false
	}



	// RR get up hops
	// test if it reaches we're done
	if revtr.Reaches(b.opts.cm) {
		return nil, true
	}

	// Got hops but we didn't reach SOURCE
	// try adding from the traceroutes
	// the atlas ran
	// if they don't complete the revtr
	// continue RR 
	return b.checkbgTRs(revtr, b.recordRoute), true
}


func violatesTunnelDestinationBasedRouting(revtr *rt.ReverseTraceroute, hop string, nextHop string, 
	tunnelSegments [][]string) bool {
	// If we arrived here, we are able to check if hopTest verifies destination based routing. 
	// We have a next hop (from a RR not from this hop), and some next hops from this RR hop for tunneling and LB.
	// First check is to find if hopTest violates destination based routing.
	// Send a RR packet to this hop, and spoofed RR if needed. 
	// Send N spoofed packets to with different sources.
	// Take random sources that can receive spoofing

	// If next hops are all the same AND different from the next hop found before, it might be in a tunnel.
	nextHopTunnel := [] string {}
	nextHopTunnelsSet := map[string]struct{}{}
	for _, seg := range(tunnelSegments) {
		if len(seg) > 0 {
			nextHopTunnelsSet[seg[0]] = struct{}{}
			nextHopTunnel = append(nextHopTunnel, seg[0])
		}
	}

	if len(nextHopTunnel) < 3 {
		// Not enough values to compare
		return false
	}

	if len(nextHopTunnelsSet) > 1 {
		// Several next hops so no default route, 
		revtr.DestinationBasedRoutingByHop[hop] = types.NoViolation
		return false
	}

	if len(nextHopTunnelsSet) == 0 {
		// No next hops, so can't verify. 
		return false
	}
	
	// Length 1, but go...
	for nextHopTunnel, _ := range(nextHopTunnelsSet) { 
		if nextHopTunnel != nextHop {
			revtr.DestinationBasedRoutingByHop[hop] = types.Tunnel
			return true
		}
	}
	return false
}

func (b *rtBatch) nextHopLastHop(revtr *rt.ReverseTraceroute) error {
	// return the last hop that respects destination based routing
	revtr.Stats.CheckDestinationBasedRoutingRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.CheckDestinationBasedRoutingDuration += dur
	}()

	// Check destination based routing until we find a last hop for which we have no evidence that 
	// it is not in a tunnel (and could bring a wrong path)
	for {
		hopTestDestinationBasedRouting := revtr.CurrPath().LastHop()
		// Check if we already have a RR segment containing a next hop 
		nextHop := map[string]struct{}{} // set
		if _, ok := revtr.NextHopsegmentsbyHop[hopTestDestinationBasedRouting]; ok {
			// nextHopSeg, _ := revtr.NextHopsegmentsbyHop[hopTestDestinationBasedRouting]
			// i, _ := util.Find((*nextHopSeg).Hops(), hopTestDestinationBasedRouting)
			return nil
		} else {
			hopsForCheckingDestinationBasedRouting := [] string {}
			// First filter the hops that are helpful to prove destination based routing
			for _, hop := range(revtr.CurrPath().LastSeg().Hops()) {
				if hop == hopTestDestinationBasedRouting {
					continue
				}
				if violationType, ok := revtr.DestinationBasedRoutingByHop[hop]; ok {
					// This hop has already been tested so check if we can use it to test other hops.
					if violationType == types.Tunnel {
						continue
					}
				}
				hopsForCheckingDestinationBasedRouting = append(hopsForCheckingDestinationBasedRouting, hop)
			}
			util.ReverseAny(hopsForCheckingDestinationBasedRouting)
			// For each hop in the segment before last hop, send a (spoofed) RR to uncover a next hop for the last hop
			// (Save it in the cache in case we have to fall back on that hop)
			for _, hop := range (hopsForCheckingDestinationBasedRouting) {
				// Issue one RR ping  
				rrSegment, err :=  issueRR(revtr, revtr.Src, hop, revtr.AtlasOpts.Staleness, b.opts.cl, b.opts.cm, true)
				if err != nil {
					log.Error(err)
				}
				revtr.Stats.DestBasedRoutRRPobes++
				// If this hop contains a next hop for the hop to test, we're done
				i, found := util.Find(rrSegment.Hops, hopTestDestinationBasedRouting)
				if found {
					// revtr.DestinationBasedRoutingByHop[hop] = types.NoViolation
					// Check if we have a next hop for hopTest
					if len(rrSegment.Hops) > i {
						nextHop[rrSegment.Hops[i+1]] = struct{}{}
						revtr.NextHopsegmentsbyHop[hopTestDestinationBasedRouting] = rt.NewRRRevSegment(rrSegment.Hops,
							 revtr.Src, hop, rrSegment.MeasurementID, rrSegment.FromCache)
						// We have a next hop so break
						break
					}
				}
				// Try to reveal with spoofed RR
				// Get VPs 
				vps, err := revtr.GetRRVPsNoUpdate(hop, b.opts.rkc, revtr.RRVPSelectionTechnique)
				if err != nil {
					log.Error(err)
					return err
				}
				if len(vps) == 0 {
					log.Infof("No spoofers for %s for checking destination based routing of %s", hop, hopTestDestinationBasedRouting)
					continue
				}
				vps = vps[:5] // Limit checks for spoofers at 5
				spoofedRRSegments, err := issueSpoofedRR(revtr, revtr.Src, hop, vps, revtr.AtlasOpts.Staleness, b.opts.cl, b.opts.cm, true)
				if err != nil {
					log.Error(err)
					return err 
				}
				// To be changed when developing cache retrieval 
				revtr.Stats.DestBasedRoutRRPobes += 5
				// Look if spoofers revealed a next hop 
				for _, spoofedRRSegment := range(spoofedRRSegments) {
					i, found := util.Find(rrSegment.Hops, hopTestDestinationBasedRouting)
					if found {
						// revtr.DestinationBasedRoutingByHop[hop] = types.NoViolation
						// Check if we have a next hop for hopTest
						if len(rrSegment.Hops) > i {
							revtr.NextHopsegmentsbyHop[hopTestDestinationBasedRouting] = rt.NewSpoofRRRevSegment(rrSegment.Hops,
								revtr.Src, hop, spoofedRRSegment.VP, rrSegment.MeasurementID, rrSegment.FromCache)
							nextHop[spoofedRRSegment.Hops[i+1]] = struct{}{}

							// We have a next hop so break
						}
					}
				}
				if len(nextHop) > 0 {
					break
				}
			}
		}
		if len(nextHop) == 0 {
			// We failed to find a next hop so we will not be able to check destination based routing for this hop, 
			// so fail this current hop
			revtr.FailCurrPath()
			// Try another hop in the segment
			continue
		}
		return nil
	}
	// We failed to find a next hop for any hop in the segment (should not happen if len(seg) > 1)
	// return fmt.Errorf("Failed to find a next hop for all hops in all paths") 
}

func randomRecSpoofSources(vps vpservice.VPSource, n int) ([]string, error){
	sources , err := vps.GetOneVPPerSite()
	if err != nil {
		// VP service is offline, so stop the measurements
		return nil, err 
	}
	recSpoofSources := [] string {}
	for _, source := range(sources.Vps)  {
		if source.RecSpoof {
			sourcei, _ := util.Int32ToIPString(source.Ip)
			recSpoofSources = append(recSpoofSources, sourcei)
		}
	}
	util.Shuffle(recSpoofSources)
	return recSpoofSources[:n], nil 
}

func (b *rtBatch) recordRoute(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.RRRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.RRDuration += dur
	}()
	for {
		// vps:= []string{}
		// target := ""
		if revtr.StopReason == rt.Failed {
			// This is a weird bug, sometimes we arrive here with a failed status
			return nil
		}

		if revtr.CheckDBasedRoutingOpts.CheckTunnel && revtr.LastHop() != revtr.Dst { 
			// do not check for first hop as we have no previous segment for it
			// Find the next last hop that respects destination based routing
			err := b.nextHopLastHop(revtr)
			if err != nil {
				log.Error(err)
				// Something wrong happened
				return nil 
			}
			if revtr.Failed() {
				revtr.FailReason = "No possibility to check destination based routing for any of the hops"
				return nil 
			}
		}
		vps, target := revtr.GetRRVPs(revtr.LastHop(), b.opts.rkc, revtr.RRVPSelectionTechnique)
		targetI, _ := util.IPStringToInt32(target)
		if iputil.IsPrivate(net.ParseIP(target)) && target != ""{
			// Here, we step back and try to find 
			// a better VP that would reveal a public IP address after the private one.
			revtr.FailCurrPath()
			continue
		}
		if len(vps) == 0 || len(revtr.RankedSpoofersByHop[targetI]) >= int(revtr.MaxSpoofers) {
			// // No vps left, so last try is to find an intersecting traceroute that we may have got 
			// // during spoofing measurements
			if revtr.UseTimestamp {
				// try timestamp
				return b.timestamp
			} else  {
				return b.backgroundTRS
			}
		}

		if revtr.CheckDBasedRoutingOpts.CheckTunnel && target != revtr.Dst{
			// If we have to check destination based routing, we'll have to run additional measurements, so run them now. 
			// Before adding the segments, check that these guys respect destination based routing.
			maxTries := 2
			// We know that this guy is able to reveal reverse hops for target
			workingSpoofers := [] string{}  
			spoofedRRSegments := [][]string{}
			for i := 0; i < maxTries && len(spoofedRRSegments) < 5; i++ {
				// Get some random sources 
				sources, err := randomRecSpoofSources(b.opts.vps, rt.RateLimit)
				if err != nil {
					return nil 
				}
				spoofers := [] string {}
				if len(workingSpoofers) == 0 {
					spoofers, err = revtr.GetRRVPsNoUpdate(target, b.opts.rkc, revtr.RRVPSelectionTechnique)
				} else {
					// Batch of 5 spoofers
					for k := 0 ; k < 5; k++ {
						spoofers = append(spoofers, workingSpoofers[k%len(workingSpoofers)]) 
					}
				}
				spoofedRRSegmentsBySpoofer, err := issueTunnelDestinationBasedCheckSpoofedRR(revtr, sources, target, spoofers, 
					revtr.AtlasOpts.Staleness, b.opts.cl, b.opts.cm, revtr.UseCache)
				if err != nil {
					// Something bad happened with spoofing, so skip 
					continue
				}
				for _, spoofedRRSegment := range(spoofedRRSegmentsBySpoofer) {
					if len(spoofedRRSegment.Hops) > 0 {
						// We found a spoofer that worked so add it to 
						workingSpoofers = append(workingSpoofers, spoofedRRSegment.VP)
						spoofedRRSegments = append(spoofedRRSegments, spoofedRRSegment.Hops)
					}
				}
			}
			// verify if we have enough segments 
			nextHopSeg, _ := revtr.NextHopsegmentsbyHop[target]
			i, _ := util.FindFirstAfterNotEqual(nextHopSeg.Hops(), target)
			nextHop := nextHopSeg.Hops()[i]
			if violatesTunnelDestinationBasedRouting(revtr,  target, nextHop, spoofedRRSegments) {
				revtr.FailCurrPath()
				continue
			}
		}
		// vps = ["non_spoofed", ip1, ip2, ...]
		// DONT NEED to put in NON SPOOFED VP
		// this is just the first step (issuing regular RR from the Source)
		// if this doesn't uncover anything, use other VPs and perform spoofed RR from each...
		if stringutil.InArray(vps, rt.NON_SPOOFED) {
			// rr is a rr segment after the destination
			rr, err := issueRR(revtr, revtr.Src, target,
				revtr.AtlasOpts.Staleness, b.opts.cl, b.opts.cm, revtr.UseCache)
			if err != nil {
				// Couldn't perform RR measurements
				// move on to try next group for now
				// If all the VPs fail and have no responses for example.
				continue
			}

			if !rr.FromCache {
				revtr.Stats.RRProbes++
			}
			
			var segs []rt.Segment
			for i, hop := range rr.Hops {
				if hop == "0.0.0.0" {
					continue
				}
				segs = append(segs, rt.NewRRRevSegment(rr.Hops[:i+1], revtr.Src, target, rr.MeasurementID, rr.FromCache))
			}

			if !revtr.AddSegments(segs, b.opts.cm) {
				// CAN PUT IN CACHE HERE
				// Failed to anything from the RR hops
				// move on to next group
				continue
			}
			// RR get up hops
			// test if it reaches we're done
			if revtr.Reaches(b.opts.cm) {
				return nil
			}
			// Got hops but we didn't reach
			// try adding from the traceroutes
			// the atlas ran
			// if they don't complete the revtr
			// we're back to the start with trToSource
			return b.checkbgTRs(revtr, b.trToSource)
		}
		log.Debug("vps is: ", vps)

		// if the code came here, the source RR didn't work, now we do spoofed RRs from VPs.
		// The cache is handled at the controller level, revtr does not have to check for responsiveness

		cacheKey := revtr.Src + "_" + target
		// logRevtr(revtr).Debug(cacheKey)
		res, ok := b.opts.ca.Get(cacheKey)
		// logRevtr(revtr).Debug(res)
		if ok && res == "dummy" {
			// Target is unresponsive, so continue
			logRevtr(revtr).Debug("SKIPPED A VP !")
			continue
		}
		next, mustDoNext := b.spoofAndUpdate(target, vps, revtr)
		if mustDoNext{
			return next
		} else {
			continue
		}
	}
}

const (
	dummyIP = "128.208.3.77"
)

func (b *rtBatch) timestamp(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.TSRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.TSDuration += dur
	}()
	logRevtr(revtr).Debug("Trying Timestamp")
	var receiverToSpooferToProbe = make(map[string]map[string][][]string)
	checkMapMagic := func(f, s string) {
		if _, ok := receiverToSpooferToProbe[f]; !ok {
			receiverToSpooferToProbe[f] = make(map[string][][]string)
		}
	}
	checksrctohoptosendspoofedmagic := func(f string) {
		if _, ok := revtr.TSSrcToHopToSendSpoofed[f]; !ok {
			revtr.TSSrcToHopToSendSpoofed[f] = make(map[string]bool)
		}
	}
	target := revtr.LastHop()
	if !revtr.TSIsResponsive(target) {
		// the target is not responsive to ts
		// move on to next step
		return b.backgroundTRS
	}
	for {
		adjs := revtr.GetTSAdjacents(target, b.opts.as)
		if len(adjs) == 0 {
			logRevtr(revtr).Debug("No adjacents for: ", target)
			// No adjacencies left, move on to the next step
			return b.backgroundTRS
		}
		var dstsDoNotStamp [][]string
		var tsToIssueSrcToProbe = make(map[string][][]string)
		if revtr.TSDstToStampsZero[target] {
			logRevtr(revtr).Debug("tsDstToStampsZero wtf")
			for _, adj := range adjs {
				dstsDoNotStamp = append(dstsDoNotStamp,
					[]string{revtr.Src, target, adj})
			}
		} else if !revtr.TSSrcToHopToSendSpoofed[revtr.Src][target] {
			logRevtr(revtr).Debug("Adding Spoofed TS to send")
			for _, adj := range adjs {
				tsToIssueSrcToProbe[revtr.Src] = append(
					tsToIssueSrcToProbe[revtr.Src],
					[]string{revtr.LastHop(),
						revtr.LastHop(),
						adj, 
						adj})
			}
		} else {
			logRevtr(revtr).Debug("TS Non of the above")
			spfs := revtr.GetTimestampSpoofers(revtr.Src, revtr.LastHop(), b.opts.vps)
			if len(spfs) == 0 {
				logRevtr(revtr).Debug("no spoofers left")
				return b.backgroundTRS
			}
			for _, adj := range adjs {
				for _, spf := range spfs {
					checkMapMagic(revtr.Src, spf)
					receiverToSpooferToProbe[revtr.Src][spf] = append(
						receiverToSpooferToProbe[revtr.Src][spf],
						[]string{revtr.LastHop(),
							revtr.LastHop(),
							adj, adj})
				}
			}
			// if we haven't already decided whether it is responsive,
			// we'll set it to false, then change to true if we get one
			revtr.TSSetUnresponsive(target)
		}
		type pair struct {
			src, dst string
		}
		type triplet struct {
			src, dst, vp string
		}
		type tripletTs struct {
			src, dst, tsip string
		}
		var revHopsSrcDstToRevSeg = make(map[pair][]rt.Segment)
		var linuxBugToCheckSrcDstVpToRevHops = make(map[triplet][]string)
		var destDoesNotStamp []tripletTs

		processTSCheckForRevHop := func(src, vp string, p *datamodel.Ping) {
			logRevtr(revtr).Debug("Processing TS: ", p)
			dsts, _ := util.Int32ToIPString(p.Dst)
			segClass := "SpoofTSAdjRevSegment"
			if vp == "non_spoofed" {
				checksrctohoptosendspoofedmagic(src)
				revtr.TSSrcToHopToSendSpoofed[src][dsts] = false
				segClass = "TSAdjRevSegment"
			} else {
				log.Debug("processing spoofed ts reply ", p)
			}
			revtr.TSSetUnresponsive(dsts)
			rps := p.GetResponses()
			if len(rps) > 0 {
				logRevtr(revtr).Debug("Response ", rps[0].Tsandaddr)
			}
			if len(rps) > 0 {
				var ts1, ts2, ts3 *datamodel.TsAndAddr
				ts1 = new(datamodel.TsAndAddr)
				ts2 = new(datamodel.TsAndAddr)
				ts3 = new(datamodel.TsAndAddr)
				switch len(rps[0].Tsandaddr) {
				case 0:
				case 1:
					ts1 = rps[0].Tsandaddr[0]
				case 2:
					ts1 = rps[0].Tsandaddr[0]
					ts2 = rps[0].Tsandaddr[1]
				case 3:
					ts1 = rps[0].Tsandaddr[0]
					ts2 = rps[0].Tsandaddr[1]
					ts3 = rps[0].Tsandaddr[2]
				}
				if ts3.Ts != 0 {
					ss, _ := util.Int32ToIPString(rps[0].Tsandaddr[2].Ip)
					var seg rt.Segment
					if segClass == "SpoofTSAdjRevSegment" {
						seg = rt.NewSpoofTSAdjRevSegment([]string{ss}, src, dsts, vp, false, int64(p.RevtrMeasurementId), p.FromCache)
					} else {
						seg = rt.NewTSAdjRevSegment([]string{ss}, src, dsts, false, int64(p.RevtrMeasurementId), p.FromCache)
					}
					revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{seg}
				} else if ts2.Ts != 0 {
					ts2ips, _ := util.Int32ToIPString(ts2.Ip)
					if ts2.Ts-ts1.Ts > 3 || ts2.Ts < ts1.Ts {
					// if ts2.Ts < ts1.Ts {
						// if 2nd slot is stamped with an increment from 1st, rev hop
						var seg rt.Segment
						if segClass == "SpoofTSAdjRevSegment" {
							seg = rt.NewSpoofTSAdjRevSegment([]string{ts2ips}, src, dsts, vp, false, int64(p.RevtrMeasurementId), p.FromCache)
						} else {
							seg = rt.NewTSAdjRevSegment([]string{ts2ips}, src, dsts, false, int64(p.RevtrMeasurementId), p.FromCache)
						}
						revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{seg}
					} else {
						// else, if 2nd stamp is clsoe to 1st, need to check for linux bug
						linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] = append(linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}], ts2ips)
					}
				} else if ts1.Ts == 0 {
					// if dst responds, does not stamp, can try advanced techniques
					ts2ips, _ := util.Int32ToIPString(ts2.Ip)
					revtr.TSDstToStampsZero[dsts] = true
					destDoesNotStamp = append(destDoesNotStamp, tripletTs{src: src, dst: dsts, tsip: ts2ips})
				} else {
					logRevtr(revtr).Debug("TS probe is ", vp, p, "no reverse hop found")
				}
			}
		}
		logRevtr(revtr).Debug("tsToIssueSrcToProbe ", tsToIssueSrcToProbe)
		if len(tsToIssueSrcToProbe) > 0 {
			// there should be a uniq thing here but I need to figure out how to do it
			for src, probes := range tsToIssueSrcToProbe {
				for _, probe := range probes {
					checksrctohoptosendspoofedmagic(src)
					if _, ok := revtr.TSSrcToHopToSendSpoofed[src][probe[0]]; ok {
						continue
					}
					// set it to true, then change it to false if we get a response
					revtr.TSSrcToHopToSendSpoofed[src][probe[0]] = true
				}
			}
			logRevtr(revtr).Debug("Issuing TS probes")
			revtr.Stats.TSProbes += len(tsToIssueSrcToProbe)
			err := issueTimestamps(revtr, tsToIssueSrcToProbe, processTSCheckForRevHop,
				revtr.AtlasOpts.Staleness, b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
			logRevtr(revtr).Debug("Done issuing TS probes ", tsToIssueSrcToProbe)
			for src, probes := range tsToIssueSrcToProbe {
				for _, probe := range probes {
					// if we got a reply, would have set sendspoofed to false
					// so it is still true, we need to try to find a spoofer
					checksrctohoptosendspoofedmagic(src)
					if revtr.TSSrcToHopToSendSpoofed[src][probe[0]] {
						mySpoofers := revtr.GetTimestampSpoofers(src, probe[0], b.opts.vps)
						for _, sp := range mySpoofers {
							logRevtr(revtr).Debug("Adding spoofed TS probe to send")
							checkMapMagic(src, sp)
							receiverToSpooferToProbe[src][sp] = append(receiverToSpooferToProbe[src][sp], probe)
						}
						// if we haven't already decided whether it is responsive
						// we'll set it to false, then change to true if we get one
						revtr.TSSetUnresponsive(probe[0])
					}
				}
			}
		}
		logRevtr(revtr).Debug("receiverToSpooferToProbe: ", receiverToSpooferToProbe)
		for _, val := range receiverToSpooferToProbe {
			revtr.Stats.SpoofedTSProbes += len(val)
		}
		if len(receiverToSpooferToProbe) > 0 {
			err := issueSpoofedTimestamps(revtr, receiverToSpooferToProbe,
				processTSCheckForRevHop, revtr.AtlasOpts.Staleness, b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
		}
		if len(linuxBugToCheckSrcDstVpToRevHops) > 0 {
			var linuxChecksSrcToProbe = make(map[string][][]string)
			var linuxChecksSpoofedReceiverToSpooferToProbe = make(map[string]map[string][][]string)
			for sdvp := range linuxBugToCheckSrcDstVpToRevHops {
				p := []string{sdvp.dst, sdvp.dst, dummyIP, dummyIP}
				if sdvp.vp == "non_spoofed" {
					linuxChecksSrcToProbe[sdvp.src] = append(linuxChecksSrcToProbe[sdvp.src], p)
				} else {
					if val, ok := linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src]; ok {
						val[sdvp.vp] = append(linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp], p)
					} else {
						linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src] = make(map[string][][]string)
						linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp] = append(linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp], p)
					}
				}
			}
			// once again leaving out a check for uniqness
			processTSCheckForLinuxBug := func(src, vp string, p *datamodel.Ping) {
				dsts, _ := util.Int32ToIPString(p.Dst)
				rps := p.GetResponses()
				if len(rps) ==  0 ||  1 >= len(rps[0].Tsandaddr){
					// This means we have only one stamp, so the one from the destination, so the 
					// ip that we tried is not on the path
					return
				} 
				ts2 := rps[0].Tsandaddr[1]

				segClass := "SpoofTSAdjRevSegment"
				// if I got a response, must not be filtering, so dont need to use spoofing
				if vp == "non_spoofed" {
					checksrctohoptosendspoofedmagic(src)
					revtr.TSSrcToHopToSendSpoofed[src][dsts] = false
					segClass = "TSAdjRevSegment"
				}
				if ts2.Ts != 0 {
					logRevtr(revtr).Debug("TS probe is ", vp, p, "linux bug")
					// TODO keep track of linux bugs
					// at least once, i observed a bug not stamp one probe, so
					// this is important, probably then want to do the checks
					// for revhops after all spoofers that are trying have tested
					// for linux bugs
				} else {
					logRevtr(revtr).Debug("TS probe is ", vp, p, "not linux bug")
					for _, revhop := range linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] {

						var seg rt.Segment
						if segClass == "TSAdjRevSegment" {
							seg = rt.NewTSAdjRevSegment([]string{revhop}, src, dsts, false, int64(p.RevtrMeasurementId), p.FromCache)
						} else {
							seg = rt.NewSpoofTSAdjRevSegment([]string{revhop}, src, dsts, vp, false, int64(p.RevtrMeasurementId), p.FromCache)
						}
						revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{seg}
					}
				}
			}
			revtr.Stats.TSProbes += len(tsToIssueSrcToProbe)
			err := issueTimestamps(revtr, linuxChecksSrcToProbe,
				processTSCheckForLinuxBug, revtr.AtlasOpts.Staleness, b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
			for _, val := range receiverToSpooferToProbe {
				revtr.Stats.SpoofedTSProbes += len(val)
			}
			err = issueSpoofedTimestamps(revtr, linuxChecksSpoofedReceiverToSpooferToProbe,
				processTSCheckForLinuxBug, revtr.AtlasOpts.Staleness, b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
		}
		receiverToSpooferToProbe = make(map[string]map[string][][]string)
		for _, probe := range destDoesNotStamp {
			spoofers := revtr.GetTimestampSpoofers(probe.src, probe.dst, b.opts.vps)
			for _, s := range spoofers {
				checkMapMagic(probe.src, s)
				receiverToSpooferToProbe[probe.src][s] = append(receiverToSpooferToProbe[probe.src][s], []string{probe.dst, probe.tsip, probe.tsip, probe.tsip, probe.tsip})
			}
		}
		// if I get the response, need to then do the non-spoofed version
		// for that, I can get everything I need to know from the probe
		// send the duplicates
		// then, for each of those that get responses but don't stamp
		// I can delcare it a revhop-- I just need to know which src to declare it
		// for
		// so really what I ened is a  map from VP,dst,adj to the list of
		// sources/revtrs waiting for it
		destDoesNotStampToVerifySpooferToProbe := make(map[string][][]string)
		vpDstAdjToInterestedSrcs := make(map[tripletTs][]string)
		processTSDestDoesNotStamp := func(src, vp string, p *datamodel.Ping) {
			dsts, _ := util.Int32ToIPString(p.Dst)
			rps := p.GetResponses()
			if len(rps) <= 0 {
				return
			}
			var ts1, ts2, ts4 *datamodel.TsAndAddr
			ts1 = new(datamodel.TsAndAddr)
			ts2 = new(datamodel.TsAndAddr)
			ts4 = new(datamodel.TsAndAddr)
			switch len(rps[0].Tsandaddr) {
			case 0:
			case 1:
				ts1 = rps[0].Tsandaddr[0]
			case 2, 3:
				ts1 = rps[0].Tsandaddr[0]
				ts2 = rps[0].Tsandaddr[1]
			case 4:
				ts1 = rps[0].Tsandaddr[0]
				ts2 = rps[0].Tsandaddr[1]
				ts4 = rps[0].Tsandaddr[3]
			}
			// if 2 stamps, we assume one was forward, one was reverse
			// if 1 or 4, we need to verify it was reverse
			// 3 should not happend according to justine?
			if ts2.Ts != 0 && ts4.Ts == 0 {
				// declare reverse hop
				ts2ips, _ := util.Int32ToIPString(ts2.Ts)
				revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{rt.NewSpoofTSAdjRevSegmentTSZeroDoubleStamp([]string{ts2ips}, src, dsts, vp, false, int64(p.RevtrMeasurementId), p.FromCache)}
				logRevtr(revtr).Debug("TS Probe is ", vp, p, "reverse hop from dst that stamps 0!")
			} else if ts1.Ts != 0 {
				logRevtr(revtr).Debug("TS probe is ", vp, p, "dst does not stamp, but spoofer ", vp, "got a stamp")
				ts1ips, _ := util.Int32ToIPString(ts1.Ip)
				destDoesNotStampToVerifySpooferToProbe[vp] = append(destDoesNotStampToVerifySpooferToProbe[vp], []string{dsts, ts1ips, ts1ips, ts1ips, ts1ips})
				// store something
				vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}] = append(vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}], src)
			} else {
				logRevtr(revtr).Debug("TS probe is ", vp, p, "no reverse hop for dst that stamps 0")
			}
		}
		if len(destDoesNotStamp) > 0 {
			err := issueSpoofedTimestamps(revtr, receiverToSpooferToProbe,
				processTSDestDoesNotStamp,
				revtr.AtlasOpts.Staleness, b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
		}

		// if you don't get a response, add it with false
		// then at the end
		if len(destDoesNotStampToVerifySpooferToProbe) > 0 {
			for vp, probes := range destDoesNotStampToVerifySpooferToProbe {
				probes = append(probes, probes...)
				probes = append(probes, probes...)
				destDoesNotStampToVerifySpooferToProbe[vp] = probes
			}
			maybeRevhopVPDstAdjToBool := make(map[tripletTs]bool)
			revHopsVPDstToRevSeg := make(map[pair][]rt.Segment)
			processTSDestDoesNotStampToVerify := func(src, vp string, p *datamodel.Ping) {
				dsts, _ := util.Int32ToIPString(p.Dst)
				rps := p.GetResponses()
				if len(rps) > 0 && len(rps[0].Tsandaddr) > 0{
					ts1 := rps[0].Tsandaddr[0]
					ts1ips, _ := util.Int32ToIPString(ts1.Ip)
					if ts1.Ts == 0 {
						logRevtr(revtr).Debug("Reverse hop! TS probe is ", vp, p, "dst does not stamp, but spoofer", vp, "got a stamp and didn't direclty")
						maybeRevhopVPDstAdjToBool[tripletTs{src: src, dst: dsts, tsip: ts1ips}] = true
					} else {
						del := tripletTs{src: src, dst: dsts, tsip: ts1ips}
						for key := range vpDstAdjToInterestedSrcs {
							if key == del {
								delete(vpDstAdjToInterestedSrcs, key)
							}
						}
						logRevtr(revtr).Debug("Can't verify reverse hop! TS probe is ", vp, p, "potential hop stamped on non-spoofed path for VP")
					}
				}
			}
			logRevtr(revtr).Debug("Issuing to verify for dest does not stamp")
			err := issueTimestamps(revtr, destDoesNotStampToVerifySpooferToProbe,
				processTSDestDoesNotStampToVerify,
				revtr.AtlasOpts.Staleness,
				b.opts.cl, revtr.UseCache)
			if err != nil {
				// logRevtr(revtr).Error(err)
			}
			for k := range maybeRevhopVPDstAdjToBool {
				for _, origsrc := range vpDstAdjToInterestedSrcs[tripletTs{src: k.src, dst: k.dst, tsip: k.tsip}] {
					// TODO Add the corresponding measurement ID
					revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}] =
						append(revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}],
							rt.NewSpoofTSAdjRevSegmentTSZeroDoubleStamp(
								[]string{k.tsip}, origsrc, k.dst, k.src, false, 0, false))
				}

			}
		}

		// Ping V/S->R:R',R',R',R'
		// (i think, but justine has it nested differently) if stamp twice,
		// declare rev hop, # else if i get one:
		// if i get responses:
		// n? times: Ping V/V->R:R',R',R',R'
		// if (never stamps) // could be a false positive, maybe R' just didn't
		// feel like stamping this time
		// return R'
		// if stamps more thane once, decl,
		if segments, ok := revHopsSrcDstToRevSeg[pair{src: revtr.Src, dst: revtr.LastHop()}]; ok {
			if revtr.AddSegments(segments, b.opts.cm) {
				// added a segment
				// if it reaches we're done
				if revtr.Reaches(b.opts.cm) {
					return nil
				}
				return b.checkbgTRs(revtr, b.trToSource)
			}
		}
		// continue to next set
	}
}

// this is used to check background traceroute.
// it is different than the step backgroundTRS
// This checks for background trs that were issued for
// current round. It is called after a step finds hops
// the step next is returned if the background trs dont
// reach
func (b *rtBatch) checkbgTRs(revtr *rt.ReverseTraceroute, next step) step {
	ns := b.backgroundTRS(revtr)
	// if ns is nil backgroundTRS reaches
	// and we're done
	if ns == nil {
		return ns
	}
	// if ns is not nil, we're not done
	// but the background traceroutes didn't
	// finish or they weren't useful
	// return the next step
	return next
}

func (b *rtBatch) backgroundTRS(revtr *rt.ReverseTraceroute) step {
	// hack here as we do not run background traceroutes, 
	// do not try to get a background traceroute, just go to symmetry
	return b.assumeSymmetric
	revtr.Stats.BackgroundTRSRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.BackgroundTRSDuration += dur
	}()
	tokens := revtr.Tokens
	revtr.Tokens = nil
	tr, err := retreiveTraceroutes(tokens, b.opts.at, b.opts.cm)
	if err != nil {
		logRevtr(revtr).Debugf(err.Error())
		// Failed to find a intersection
		return b.assumeSymmetric
	}
	logRevtr(revtr).Debug("Creating TRToSrc seg: ", tr.hops, " ", revtr.Src, " ", tr.addr)
	revtr.TracerouteID = tr.tracerouteID
	segment := rt.NewTrtoSrcRevSegment(tr.hops, tr.hopTypes, revtr.Src, tr.addr, tr.tracerouteID)
	if !revtr.AddBackgroundTRSegment(segment, b.opts.cm) {
		panic("Failed to add background TR segment. That's not possible")
	}
	if revtr.Reaches(b.opts.cm) {
		return nil
	}
	panic("Added a TR to source but the revtr didn't reach")
}

func isInterdomainSymmetryAssumption(ip2as *radix.BGPRadixTree, assumedLinkNearSide string, assumedLinkFarSide string) bool {
	assumedLinkNearSideASN := ip2as.Get(assumedLinkNearSide)
	assumedLinkFarSideASN := ip2as.Get(assumedLinkFarSide)

	isInterdomainSymmetry := assumedLinkNearSideASN != assumedLinkFarSideASN  || assumedLinkNearSideASN == -1 || assumedLinkFarSideASN == -1
	return isInterdomainSymmetry
}

func (b *rtBatch) assumeSymmetric(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.AssumeSymmetricRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.AssumeSymmetricDuration += dur
	}()

	// if last hop is assumed, add one more from that tr
	if reflect.TypeOf(revtr.CurrPath().LastSeg()) == reflect.TypeOf(&rt.DstSymRevSegment{}) {
		logRevtr(revtr).Debug("Backing off along current path for ", revtr.Src, " ", revtr.Dst)
		// need to not ignore the hops in the last segment, so can't just
		// call add_hops(revtr.hops + revtr.deadends)
		newSeg := revtr.CurrPath().LastSeg().Clone().(*rt.DstSymRevSegment)
		logRevtr(revtr).Debug("newSeg: ", newSeg)
		var allHops []string
		for i, seg := range *revtr.CurrPath().Path {
			// Skip the last one
			if i == len(*revtr.CurrPath().Path)-1 {
				continue
			}
			allHops = append(allHops, seg.Hops()...)
		}
		allHops = append(allHops, revtr.Deadends()...)
		logRevtr(revtr).Debug("all hops: ", allHops)
		err := newSeg.AddHop(allHops)
		if err != nil {
			logRevtr(revtr).Error(err)
		}
		if !revtr.SymmetryOpts.IsAllowInterdomainSymmetry {
			// Check here that the two last hops of new segs are intradomain and
			// that there is no interdomain assumption 
			dstSymmRevSegLen := len(newSeg.RevSegment.Segment)
			assumedLinkNearSide := newSeg.RevSegment.Segment[dstSymmRevSegLen-1]
			assumedLinkFarSide := newSeg.RevSegment.Segment[dstSymmRevSegLen-2]
			if isInterdomainSymmetryAssumption(b.opts.ip2as, assumedLinkNearSide, assumedLinkFarSide) {
				// we failed so we're done
				revtr.EndTime = time.Now()
				revtr.FailReason = "Assume symmetry on an interdomain link"
				revtr.StopReason = rt.Failed
				if revtr.OnFail != nil {
					revtr.OnFailOnce.Do(func() {
						revtr.OnFail(revtr)
					})
				}
				return nil
			}
		}
		

		logRevtr(revtr).Debug("New seg: ", newSeg)
		added := revtr.AddAndReplaceSegment(newSeg)
		if added {
			logRevtr(revtr).Debug("Added hop from another DstSymRevSegment")
			if revtr.Reaches(b.opts.cm) {
				return nil
			}
			return b.trToSource
		}
		panic("Should never get here")
	}
	trace, err := issueTraceroute(b.opts.cl, b.opts.cm,
		revtr.Src, revtr.LastHop(), revtr.AtlasOpts.Staleness, revtr.UseCache)
	if err != nil {
		logRevtr(revtr).Debug("Issue traceroute err: ", err)
		revtr.ErrorDetails.WriteString("Error running traceroute\n")
		revtr.ErrorDetails.WriteString(err.Error() + "\n")
		revtr.FailCurrPath()
		if revtr.Failed() {
			// we failed so we're done
			revtr.FailReason = "Traceroute failed when trying to assume symmetric"
			return nil
		}
		// move on to the top of the loop
		// We found a hop and will asume it to be symmetric, so send a times
		return b.trToSource
	}
	var hToIgnore []string
	hToIgnore = append(hToIgnore, revtr.Hops()...)
	hToIgnore = append(hToIgnore, revtr.Deadends()...)
	logRevtr(revtr).Debug("Attempting to add hop from tr ", trace.hops)
	
	newSeg := rt.NewDstSymRevSegment(revtr.Src,
			revtr.LastHop(),
			trace.hops, 1,
			hToIgnore,
			trace.measurementID,
			trace.fromCache)
	if !revtr.SymmetryOpts.IsAllowInterdomainSymmetry {

		// Check here that there is no interdomain assumption between the last hop of the revtr 
		// and the penultimate of the traceroute
		assumeSymmetricLinkFarSide := revtr.LastHop()
		assumeSymmetricLinkNearSide := trace.hops[len(trace.hops)-2]
		if isInterdomainSymmetryAssumption(b.opts.ip2as, assumeSymmetricLinkNearSide, assumeSymmetricLinkFarSide) {
			// we failed so we're done
			revtr.EndTime = time.Now()
			revtr.FailReason = "Assume symmetry on an interdomain link"
			revtr.StopReason = rt.Failed
			if revtr.OnFail != nil {
				revtr.OnFailOnce.Do(func() {
					revtr.OnFail(revtr)
				})
			}
			return nil
		}

	}
	
	if revtr.AddSegments([]rt.Segment{
			newSeg,
		},
		b.opts.cm) {
		
		if revtr.Reaches(b.opts.cm) {
			// done
			return nil
		}
		return b.trToSource
	}
	// everything failed
	revtr.FailCurrPath()
	if revtr.Failed() {
		revtr.FailReason = "Failed to find hops for any path."
		return nil
	}
	return b.trToSource
}

// This step in run at the very end of the revtr if we are in a survey case. 
func (b *rtBatch) forwardTraceroute(revtr *rt.ReverseTraceroute) {
	traceroute, err := issueTraceroute(b.opts.cl, b.opts.cm, revtr.Src, revtr.Dst, revtr.AtlasOpts.Staleness, true) 
	if err != nil {
		// log.Error(err)
		traceError, ok := err.(tracerouteError)
		if ok {
			revtr.ForwardTracerouteID = int64(traceError.trace.RevtrMeasurementId)
		}
	} else {
		revtr.ForwardTracerouteID = traceroute.measurementID
	}
}

// This step runs pings to the different reverse hops to get RTTs 
func (b * rtBatch) RTTPings(revtr * rt.ReverseTraceroute) {

	pms := []*datamodel.PingMeasurement{}
	hopsSeen := make(map[string]bool)
	srcI, _ := util.IPStringToInt32(revtr.Src)
	for _, s := range *revtr.CurrPath().Path {

		for _, hi := range s.Hops() {
			if hopsSeen[hi] {
				continue
			}
			hopsSeen[hi] = true
			dstHopI, _ := util.IPStringToInt32(hi)
			p := &datamodel.PingMeasurement{
				Src:        srcI,
				Dst:        dstHopI,
				Timeout:    10,
				Count:      "1",
				CheckCache: true,
				CheckDb:    false,
				Staleness:  15, // 15 minutes by default
				Label:      revtr.Label,
				SaveDb:     true,
				FromRevtr:  true,
			}
			pms = append(pms, p)
		}
		
	}
	prs, err :=  issuePings(pms, b.opts.cl)
	if err != nil {
		log.Error(err)
	}

	for _, pr := range(prs) {
		minRTT := uint32(0)
		
		if len(pr.Responses) > 0 {
			rtts := []uint32{}
			for _, resp := range(pr.Responses) {
				rtts = append(rtts, resp.Rtt) 
			}
			minRTT = util.Min(rtts)			
		}
		revtr.RTTByHop[pr.Dst] = rt.RTTMeasurement{MeasurementID: int64(pr.RevtrMeasurementId), RTT: minRTT}
	}
}


func issuePings(pms [] *datamodel.PingMeasurement, cl client.Client) ([]*datamodel.Ping, error) {
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{Pings: pms})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	prs := []*datamodel.Ping{}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// log.Error(err)
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func issueTimestamps(revtr *rt.ReverseTraceroute, issue map[string][][]string,
	fn func(string, string, *datamodel.Ping),
	staleness int64,
	cl client.Client,
	checkCache bool) error {

	log.Debug("Issuing timestamps")
	var pings []*datamodel.PingMeasurement
	for src, probes := range issue {
		srcip, _ := util.IPStringToInt32(src)
		for _, probe := range probes {
			dstip, _ := util.IPStringToInt32(probe[0])
			tss := ipoptions.TimestampAddrToString(probe)
			if iputil.IsPrivate(net.ParseIP(probe[0])) {
				continue
			}
			p := &datamodel.PingMeasurement{
				Src:        srcip,
				Dst:        dstip,
				TimeStamp:  tss,
				Timeout:    10,
				Count:      "1",
				CheckCache: checkCache,
				CheckDb:    false,
				Staleness:  staleness,
				Label:      revtr.Label,
				SaveDb:     true,
				FromRevtr:  true,
			}
			pings = append(pings, p)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		log.Error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// log.Error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		fn(srcs, "non_spoofed", pr)
	}
	return nil
}

func issueSpoofedTimestamps(revtr *rt.ReverseTraceroute, issue map[string]map[string][][]string,
	fn func(string, string, *datamodel.Ping),
	staleness int64,
	cl client.Client,
	checkCache bool) error {

	log.Debug("Issuing spoofed timestamps")
	var pings []*datamodel.PingMeasurement
	for reciever, spooferToProbes := range issue {
		recip, _ := util.IPStringToInt32(reciever)
		for spoofer, probes := range spooferToProbes {
			spooferip, _ := util.IPStringToInt32(spoofer)
			for _, probe := range probes {
				dstip, _ := util.IPStringToInt32(probe[0])
				tss := ipoptions.TimestampAddrToString(probe)
				if iputil.IsPrivate(net.ParseIP(probe[0])) {
					continue
				}
				p := &datamodel.PingMeasurement{
					Src:         spooferip,
					Spoof:       true,
					Dst:         dstip,
					SpooferAddr: recip,
					SAddr      : reciever,
					TimeStamp:   tss,
					Timeout:     10,
					Count:       "1",
					CheckCache:  checkCache,
					CheckDb:     false,
					Staleness:   staleness,
					Label:       revtr.Label,
					SaveDb:      true,
					SpoofTimeout: 60,
					FromRevtr:   true,
				}
				pings = append(pings, p)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		log.Error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// log.Error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		vp, _ := util.Int32ToIPString(pr.SpoofedFrom)
		fn(srcs, vp, pr)
	}
	return nil
}

var (
	errPrivateIP = fmt.Errorf("The target is a private IP addr")
)

type intersectingTR struct {
	date time.Time
	tracerouteID int64
	ripeProbeID int64
	platform string
	src string 
	dst string 
	addr string
	hops []string
	hopTypes [] pb.IntersectHopType
}

type tracerouteError struct {
	err   error
	trace *datamodel.Traceroute
	extra string
}

func (te tracerouteError) Error() string {
	var buf bytes.Buffer
	if te.err != nil {
		buf.WriteString(te.err.Error() + "\n")
	}
	if te.trace.Error != "" {
		buf.WriteString(fmt.Sprintf("Error running traceroute %v ", te.trace) + "\n")
		if te.extra != "" {
			buf.WriteString(te.extra + "\n")
		}
		return buf.String()
	}
	buf.WriteString(te.trace.ErrorString() + "\n")
	if te.extra != "" {
		buf.WriteString(te.extra + "\n")
	}
	return buf.String()
}

type traceroute struct {
	src, dst string
	hops     []string
	measurementID int64
	fromCache bool
}

func issueTraceroute(cl client.Client, cm clustermap.ClusterMap,
	src, dst string, staleness int64, checkCache bool) (traceroute, error) {

	srci, _ := util.IPStringToInt32(src)
	dsti, _ := util.IPStringToInt32(dst)
	if iputil.IsPrivate(net.ParseIP(dst)) {
		return traceroute{}, errPrivateIP
	}
	tr := datamodel.TracerouteMeasurement{
		Src:        srci,
		Dst:        dsti,
		CheckCache: checkCache,
		CheckDb:    false,
		SaveDb:     true,
		Staleness:  staleness,
		Timeout:    60,
		Wait:       "2",
		Attempts:   "1",
		LoopAction: "1",
		Loops:      "3",
		FromRevtr:  true,
	}
	log.Debug("Issuing traceroute src: ", src, " dst: ", dst)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()
	st, err := cl.Traceroute(ctx, &datamodel.TracerouteArg{
		Traceroutes: []*datamodel.TracerouteMeasurement{&tr},
	})
	if err != nil {
		log.Error(err)
		return traceroute{}, fmt.Errorf("Failed to run traceroute: %v", err)
	}
	for {
		trace, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			log.Errorf("Error running traceroute from %s to %s", src, dst)
			return traceroute{}, fmt.Errorf("Error running traceroute: %v", err)
		}
		if trace.Error != "" {
			return traceroute{}, tracerouteError{trace: trace}
		}
		trdst, _ := util.Int32ToIPString(trace.Dst)
		var hopst []string
		cls := cm.Get(trdst)
		log.Debug("Got traceroute: ", tr)
		for i, hop := range trace.GetHops() {
			if i != len(trace.GetHops())-1 {
				j := hop.ProbeTtl + 2
				for j < trace.GetHops()[i].ProbeTtl {
					hopst = append(hopst, "*")
				}
			}
			addrst, _ := util.Int32ToIPString(hop.Addr)
			hopst = append(hopst, addrst)
		}
		if len(hopst) == 0 {
			log.Debug("Received traceroute with no hops")

			return traceroute{}, tracerouteError{trace: trace}
		}
		if cm.Get(hopst[len(hopst)-1]) != cls {
			return traceroute{}, tracerouteError{
				err:   fmt.Errorf("Traceroute didn't reach destination"),
				trace: trace,
				extra: fmt.Sprintf("<a href=\"/runrevtr?src=%s&dst=%s\">Try rerunning from the last responsive hop! </a>", src, hopst[len(hopst)-1])}
		}
		log.Debug("Got traceroute ", hopst)
		return traceroute{src: src, dst: dst, hops: hopst, measurementID: int64(trace.RevtrMeasurementId), fromCache: trace.FromCache}, nil
	}
	return traceroute{}, fmt.Errorf("Issue traceroute failed to do anything")
}

func intersectingTraceroute(src, dst string, addrs []uint32,
	atl at.Atlas,
	cm clustermap.ClusterMap, atlasOpts rt.AtlasOptions) (intersectingTR, []*apb.IntersectionResponse, error) {
	// It can take a while if the traceroutes need to be refreshed. 
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30) 
	defer cancel()
	as, err := atl.GetIntersectingPath(ctx)
	if err != nil {
		log.Error(err)
		return intersectingTR{}, nil, err
	}
	dest, _ := util.IPStringToInt32(src)
	srci, _ := util.IPStringToInt32(dst)
	for _, addr := range addrs {
		log.Debug("Attempting to find TR for hop: ", addr,
			"(", ipstr(addr).String(), ")", " to ", src)
		is := apb.IntersectionRequest{
			UseAliases: true,
			UseAtlasRr: atlasOpts.UseRRPings,
			Staleness:  atlasOpts.Staleness,
			StalenessBeforeRefresh: atlasOpts.StalenessBeforeRefresh,
			Dest:       dest,
			Address:    addr,
			Src:        srci,
			IgnoreSource: atlasOpts.IgnoreSource,
			IgnoreSourceAs: atlasOpts.IgnoreSourceAS,
			Platforms: atlasOpts.Platforms,
		}
		err := as.Send(&is)
		if err != nil {
			log.Error(err)
			return intersectingTR{}, nil, err
		}
	}
	err = as.CloseSend()
	if err != nil {
		log.Error(err)
		return intersectingTR{}, nil, err
	}
	var tokens []*apb.IntersectionResponse
	for {
		itr, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return intersectingTR{}, nil, err
		}
		log.Debug("Received Response: ", itr)
		switch itr.Type {
		case apb.IResponseType_PATH:
			var hs []string
			var ht [] pb.IntersectHopType
			var found bool
			addr, _ := util.Int32ToIPString(itr.Path.Address)
			for _, h := range itr.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				log.Debug("Fixing up hop: ", hss)
				clusterIDAddr := cm.Get(addr)
				clusterIDHss := cm.Get(hss)
				if !found && clusterIDAddr != clusterIDHss {
					continue
				}
				found = true
				hs = append(hs, hss)
				ht = append(ht, h.IntersectHopType)
			}
			srcI, _ := util.Int32ToIPString(itr.Src)
			return intersectingTR{tracerouteID:itr.TracerouteId, hopTypes: ht, hops: hs, addr: addr,
				platform: itr.Platform,
				src: srcI,
				ripeProbeID: itr.SourceProbeId,
				date: time.Unix(itr.Timestamp, 0),}, nil, nil
		case apb.IResponseType_NONE_FOUND:
			log.Debug("Found no path for ", itr)
		case apb.IResponseType_TOKEN:
			// TODO Unused memory here
			tokens = append(tokens, itr)
		}
	}
	return intersectingTR{}, tokens, nil
}

func retreiveTraceroutes(reqs []*apb.IntersectionResponse, atl at.Atlas,
	cm clustermap.ClusterMap) (intersectingTR, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3600) // For debug
	defer cancel()
	as, err := atl.GetPathsWithToken(ctx)
	if err != nil {
		return intersectingTR{}, err
	}

	for _, req := range reqs {
		log.Debug("Sending for token ", req)
		err := as.Send(&apb.TokenRequest{
			Token: req.Token,
		})
		if err != nil {
			log.Error(err)
			return intersectingTR{}, err
		}
	}
	err = as.CloseSend()
	if err != nil {
		log.Error(err)
	}
	for {
		resp, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return intersectingTR{}, err
		}
		log.Debug("Received token response: ", resp)
		switch resp.Type {
		case apb.IResponseType_PATH:
			var hs []string
			var ht []apb.IntersectHopType
			var found bool
			addr, _ := util.Int32ToIPString(resp.Path.Address)
			for _, h := range resp.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				if !found && cm.Get(addr) != cm.Get(hss) {
					continue
				}
				found = true
				hs = append(hs, hss)
				ht = append(ht, h.IntersectHopType)
			}
			return intersectingTR{tracerouteID: resp.TracerouteId, hopTypes: ht, hops: hs, addr: addr, date: time.Unix(resp.Timestamp, 0)}, nil
		}
	}
	return intersectingTR{}, fmt.Errorf("no traceroute found")
}

func issueSpoofedRRImpl(revtr *rt.ReverseTraceroute, dst string,
	pms []*datamodel.PingMeasurement, cl client.Client, cm clustermap.ClusterMap) (map[uint32]types.SpoofRRHops, error) {
	rrs := map[uint32]types.SpoofRRHops{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*70)
	defer cancel()
	// log.Info("Spoofed ping to send to controller")
	st, err := cl.Ping(ctx, &datamodel.PingArg{
		Pings: pms,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for {
		log.Debug("entered recv loop")
		p, err := st.Recv()
		if err == io.EOF {
			log.Debug("Got EOF")
			break
		}
		log.Debug("st.Recv() done, p=", p)
		if err != nil {

			log.Debug("got error:", err)
			log.Error(err)
			return rrs, err
		}
		
		if p.FromCache {
			revtr.Stats.SpoofedRRProbes--
		}
		// revtr.AddMeasurementsID(dst, p.Id)

		pr := p.GetResponses()

		if len(pr) == 0 {
			continue
		}
		log.Debug("p.GetResponses() done. pr[0].RR = ", pr[0].RR)
		sspoofer, _ := util.Int32ToIPString(p.SpoofedFrom)
		if len(pr[0].RR) == 0 {
			// log.Error("Got rr response with no hops")
		}

		rrs[p.SpoofedFrom] = types.SpoofRRHops{
			Hops: processRR(revtr, dst, sspoofer, pr[0].RR, true, cm),
			VP:   sspoofer,
			MeasurementID: int64(p.RevtrMeasurementId),
			FromCache: p.FromCache,
		}
		// log.Debug("receiver: ", recv, " destinaton: ", dst, " vp: ", sspoofer, " hops: ", pr[0].RR)
	}
	return rrs, nil
}

func issueSpoofedRR(revtr *rt.ReverseTraceroute, recv, dst string, spoofers []string, staleness int64,
	cl client.Client, cm clustermap.ClusterMap, checkCache bool) (map[uint32]types.SpoofRRHops, error) {
	// Rcv is the spoofed address
	/*
	now := time.Now().Format("2006_01_02_15_04")

	fname := "text_go_" + now + ".log"
	logf, errf := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if errf != nil {
		log.Error(errf)
	}

	defer logf.Close()
	*/

	if iputil.IsPrivate(net.ParseIP(dst)) {
		return nil, errPrivateIP
	}
	dsti, _ := util.IPStringToInt32(dst)
	var pms []*datamodel.PingMeasurement
	for _, spoofer := range spoofers {
		log.Debug("Creating spoofed ping from ", spoofer, " to ", dst, " received by ", recv)
		/*
		if _, errf := logf.WriteString("issuing spoofed ping from" + src + " to " + dst + " received by " + recv + "\n"); errf != nil {
			log.Error(errf)
		}
		*/
		spooferi, _ := util.IPStringToInt32(spoofer)
		recvi, _ := util.IPStringToInt32(recv)
		sPort := rand.Intn(environment.MaxSport - environment.MinSport) + environment.MinSport
		pms = append(pms, &datamodel.PingMeasurement{
			Src:        spooferi,
			Dst:        dsti,
			SpooferAddr: recvi,
			SAddr:      recv,
			Sport: strconv.Itoa(sPort),
			Timeout:    20,
			Count:      "1",
			Staleness:  staleness,
			CheckCache: checkCache,
			Spoof:      true,
			RR:         true,
			Label:      revtr.Label, 
			SaveDb:     true,
			SpoofTimeout: 20,
			FromRevtr: true,
			// Wait: "0.001", // To get ARP reply, see Mattthew's email
		})
	}
	rrs, err := issueSpoofedRRImpl(revtr, dst, pms, cl, cm )
	if err != nil {
		return nil, err
	}
	return rrs, nil
}

func issueTunnelDestinationBasedCheckSpoofedRR(revtr *rt.ReverseTraceroute, srcs [] string, dst string, spoofers[] string, 
	staleness int64,
	cl client.Client,
	cm clustermap.ClusterMap,
	checkCache bool) (map[uint32]types.SpoofRRHops, error) {
	// Send spoofed packets to different sources to reveal next hop. 
	pms := [] *datamodel.PingMeasurement {}
	for i, src := range(srcs) {
		// srci, _ := util.IPStringToInt32(src)
		// if iputil.IsPrivate(net.ParseIP(src)) {
		// 	continue
		// }
		spooferi , _ := util.IPStringToInt32(spoofers[i])
		dsti, _ := util.IPStringToInt32(dst)
		pm := &datamodel.PingMeasurement{
			Src:        spooferi,
			Dst:        dsti,
			SAddr:      src,
			Timeout:    20,
			Count:      "1",
			Staleness:  staleness,
			CheckCache: checkCache,
			Spoof:      true,
			RR:         true,
			Label:      revtr.Label, 
			SaveDb:     true,	
			SpoofTimeout: 60,
			FromRevtr: true,
		}
		pms = append(pms, pm) 
	}
	rrs, err := issueSpoofedRRImpl(revtr, dst, pms, cl, cm )
	if err != nil {
		return nil, err
	}
	return rrs, nil
}

func issueRR(revtr *rt.ReverseTraceroute, src, dst string, staleness int64,
	cl client.Client, cm clustermap.ClusterMap, checkCache bool) (types.RRHops, error) {
	if iputil.IsPrivate(net.ParseIP(dst)) {
		return types.RRHops{}, errPrivateIP
	}
	srci, _ := util.IPStringToInt32(src)
	dsti, _ := util.IPStringToInt32(dst)
	pm := &datamodel.PingMeasurement{
		Src:        srci,
		Dst:        dsti,
		RR:         true,
		Timeout:    20,
		Count:      "1",
		CheckCache: checkCache,
		CheckDb:    false,
		Staleness:  staleness,
		Label:      revtr.Label,
		SaveDb:     true,
		FromRevtr:  true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second* 30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{
		Pings: []*datamodel.PingMeasurement{
			pm,
		},
	})
	if err != nil {

		return types.RRHops{}, err
	}
	for {
		p, err := st.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Error(err)
			log.Errorf("Failed running RR ping from %s to %s", src, dst)
			return types.RRHops{}, err
		}

		// revtr.AddMeasurementsID(dst, p.Id)

		pr := p.GetResponses()
		if len(pr) == 0 {
			return types.RRHops{}, fmt.Errorf("no responses")
		}
		hops := processRR(revtr, dst, rt.NON_SPOOFED, pr[0].RR, true, cm)
		return types.RRHops{Hops: hops, MeasurementID: int64(p.RevtrMeasurementId), FromCache: p.FromCache}, nil
	}
	return types.RRHops{}, fmt.Errorf("no responses")
}

func stringSliceRIndex(ss []string, s string) int {
	var rindex int
	rindex = -1
	for i, sss := range ss {
		if sss == s {
			rindex = i

		}

	}
	return rindex
}

func updateIngress(revtr *rt.ReverseTraceroute, predictedIngress string, dst string, hops []uint32) {
	// Return the spoofers with the ingress that are still to be tested
	
	isSameIngress := false 
	foundIngress := ""
	for _, hop := range(hops) {
		hopS, _ := util.Int32ToIPString(hop)
		if hopS == predictedIngress {
			isSameIngress = true
			break
		}
		if _, ok := revtr.FailedIngress[dst][hopS]; ok {
			foundIngress = hopS
		}
	}
	// Look why we failed to uncover hops
	// 1. Right ingress but not the same distance, still try the same ingress 
	if isSameIngress {
		revtr.FailedIngress[dst][predictedIngress]++
	} else {
		// 2. Wrong ingress, two cases
		if foundIngress != "" {
			// 2.1 Ingress is in ingress set
			revtr.FailedIngress[dst][predictedIngress]++
			revtr.FailedIngress[dst][foundIngress]++
		} else {
			revtr.FailedIngress[dst][predictedIngress]++
		}	
	}
}

func processRR(revtr *rt.ReverseTraceroute, dst string, spoofer string, hops []uint32,
	removeLoops bool, cm clustermap.ClusterMap) [] string {
	if len(hops) == 0 {
		return []string{}
	}
	dstcls := cm.Get(dst)
	var hopss []string
	for _, s := range hops {

		hs, _ := util.Int32ToIPString(s)
		hopss = append(hopss, hs)
	}
	if cm.Get(hopss[len(hopss)-1]) == dstcls {
		// The last hop is the destination, so no hops
		if spoofer != rt.NON_SPOOFED {
			ingress := revtr.RRHop2VPSIngress[dst][spoofer]
			updateIngress(revtr, ingress, dst, hops)
		}
		
		return [] string {}
	}
	i := len(hops) - 1
	var found bool
	// check if we reached dst with at least one hop to spare
	for !found && i > 0 {
		i--
		if dstcls == cm.Get(hopss[i]) {
			found = true
		}
	}
	
	if !found {
		// Does not work 
		// if revtr.RRHeuristicsOpt.UseDoubleStamp {
		// 	// Check if we have a double stamp IP address. 
		// 	// If it is the case, take this hop as the next hop 
		// 	for j, hop := range(hops[:len(hops)-1]) {
		// 		nextHop := hops[j+1]
		// 		if hop == 0 {
		// 			continue
		// 		}
				
		// 		if hop == nextHop  {
		// 			found = true
		// 			i = j
		// 		}
		// 	}
		// }
	}
	if found {
		log.Debug("Found hops RR at: ", i)
		// remove the destination cluster
		// If the last hop is the destination cluster
		// then we have no hops, otherwise get all the hops
		if i == len(hops)-1 {
			hopss = []string{}
		} else {
			hopss = hopss[i+1:]
		}
		// remove cluster level loops
		if removeLoops {
			var clusters []string
			for _, hop := range hopss {
				clusters = append(clusters, cm.Get(hop))
			}
			var retHops []string
			var currIndex int
			for currIndex <= len(clusters)-1 {
				ri := stringSliceRIndex(clusters, clusters[currIndex])
				if ri >= currIndex {
					retHops = append(retHops, hopss[ri])
					currIndex = ri + 1
				}
			}
			log.Debug("Got Hops: ", retHops)
			return retHops
		}
		log.Debug("Got Hops: ", hopss)
		return hopss
	}

	if spoofer != rt.NON_SPOOFED {
		ingress := revtr.RRHop2VPSIngress[dst][spoofer]
		updateIngress(revtr, ingress, dst, hops)
	}
	
	return []string{}
}

func (b *rtBatch) run(revtr *rt.ReverseTraceroute,
	ret chan<- *rt.ReverseTraceroute) {
	defer b.wg.Done()
	currStep := b.initialStep
	for {
		select {
		case <-b.opts.ctx.Done():
			revtr.StopReason = rt.Canceled
			ret <- revtr
			return
		default:
			if revtr.StopReason == rt.Failed {
				// This is a weird bug, sometimes we arrive here with a failed status
				if revtr.IsRunForwardTraceroute {
					b.forwardTraceroute(revtr)
				}
				ret <- revtr
				return
			}
			currStep = currStep(revtr)
			if currStep == nil {
				if revtr.IsRunForwardTraceroute {
					b.forwardTraceroute(revtr)
				}
				if revtr.IsRunRTTPings {
					if revtr.StopReason != rt.Failed {
						b.RTTPings(revtr)
					}
				}

				logRevtr(revtr).Debug("Done running ", revtr)
				ret <- revtr
				return
			}
		}
	}
}