package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/radix"
	rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/client"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	iputil "github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	repo "github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	reversetraceroute "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/runner"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	nameSpace   = "revtr"
	goCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
	runningRevtrs = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "revtrs",
		Name:      "running_revtrs",
		Help:      "The count of currently running reverse traceroutes.",
	})
)

func init() {
	prometheus.MustRegister(goCollector)
	prometheus.MustRegister(runningRevtrs)
}

type errorf func() error

func logError(ef errorf) {
	if err := ef(); err != nil {
		log.Error(err)
	}
}

// BatchIDError is returned when an invalid batch id is sent
type BatchIDError struct {
	batchID uint32
}

func (be BatchIDError) Error() string {
	return fmt.Sprintf("invalid batch id %d", be.batchID)
}

// SrcError is returned when an invalid src address is given
type SrcError struct {
	src string
}

func (se SrcError) Error() string {
	return fmt.Sprintf("invalid src address %s", se.src)
}

// DstError is returned when an invalid src address is given
type DstError struct {
	dst string
}

func (de DstError) Error() string {
	return fmt.Sprintf("invalid dst address %s", de.dst)
}

func validSrc(src string, vps []*vppb.VantagePoint) (string, bool) {
	for _, vp := range vps {
		s, _ := util.Int32ToIPString(vp.Ip)
		if vp.Hostname == src || s == src {
			return s, true
		}
	}
	return "", false
}

func validDest(dst string, vps []*vppb.VantagePoint) (string, bool) {
	var notIP bool
	ip := net.ParseIP(dst)
	if ip == nil {
		notIP = true
	}
	if notIP {
		for _, vp := range vps {
			if vp.Hostname == dst {
				ips, _ := util.Int32ToIPString(vp.Ip)
				return ips, true
			}
		}
		res, err := net.LookupHost(dst)
		if err != nil {
			log.Error(err)
			return "", false
		}
		if len(res) == 0 {
			return "", false
		}
		return res[0], true
	}
	if iputil.IsPrivate(ip) {
		return "", false
	}
	return dst, true
}

func verifyAddrs(src, dst string, vps []*vppb.VantagePoint) (string, string, error) {
	nsrc, valid := validSrc(src, vps)

	if !valid {
		srcI, _ := util.IPStringToInt32(src)
		log.Errorf("Invalid source: %s, %d", src, srcI)
		return "", "", SrcError{src: src}
	}
	ndst, valid := validDest(dst, vps)
	if !valid {
		log.Errorf("Invalid destination: %s", dst)
		return "", "", DstError{dst: dst}
	}
	// ensure they're valid ips
	if net.ParseIP(nsrc) == nil {
		log.Errorf("Invalid source: %s", nsrc)
		return "", "", SrcError{src: nsrc}
	}
	if net.ParseIP(ndst) == nil {
		log.Errorf("Invalid destination: %s", ndst)
		return "", "", DstError{dst: ndst}
	}
	return nsrc, ndst, nil
}

// RTStore is the interface for storing/loading/allowing revtrs to be run
type RTStore interface {
	GetUserByKey(string) (pb.RevtrUser, error)
	StoreRevtr(pb.ReverseTraceroute) error
	GetRevtrsInBatch(uint32, uint32) ([]*pb.ReverseTraceroute, error)
	GetRevtrsByLabel(uint32, string) ([]*pb.ReverseTraceroute, error)
	GetRevtrsMetaOnly(uint32, uint32) ([]*pb.ReverseTracerouteMetaOnly, error)
	GetRevtrsBatchStatus(uint32, uint32) ([]pb.RevtrStatus, error)
	UpdateRevtr(uint32, uint32, int64) (error)
	CreateRevtrBatch([]*pb.RevtrMeasurement, string) ([]*pb.RevtrMeasurement, uint32, error)
	StoreBatchedRevtrs([]pb.ReverseTraceroute) error
	ExistsRevtr(string) (map[types.SrcDstPair]struct{}, error)  
}

// RevtrServer in the interface for the revtr server
type RevtrServer interface {
	RunRevtr(*pb.RunRevtrReq) (*pb.RunRevtrResp, error)
	GetRevtr(*pb.GetRevtrReq) (*pb.GetRevtrResp, error)
	GetRevtrByLabel(*pb.GetRevtrByLabelReq) (*pb.GetRevtrByLabelResp, error)
	GetRevtrsMetaOnly(*pb.GetRevtrMetaOnlyReq) (*pb.GetRevtrMetaOnlyResp, error)
	GetRevtrBatchStatus(*pb.GetRevtrBatchStatusReq) (*pb.GetRevtrBatchStatusResp, error)
	UpdateRevtr(*pb.UpdateRevtrReq) (*pb.UpdateRevtrResp, error)
	GetSources(*pb.GetSourcesReq) (*pb.GetSourcesResp, error)
	AddRevtr(pb.RevtrMeasurement) (uint32, error)
	StartRevtr(context.Context, uint32) (<-chan Status, error)
	CleanAtlas(*pb.CleanAtlasReq) (*pb.CleanAtlasResp, error)
	RunAtlas(*pb.RunAtlasReq) (*pb.RunAtlasResp, error)
}

type serverOptions struct {
	rts                       RTStore
	vps                       vpservice.VPSource
	as                        types.AdjacencySource
	cs                        types.ClusterSource
	rkc						  rk.RankingSource
	ca                        types.Cache
	run                       runner.Runner
	cacheTime int
	rootCA, certFile, keyFile string
	
}

// Option configures the server
type Option func(*serverOptions)

// WithCacheTime configures the server to use the cache c
func WithCacheTime(ct int) Option {
	return func(so *serverOptions) {
		so.cacheTime = ct
	}
}

// WithCache configures the server to use the cache c
func WithCache(c types.Cache) Option {
	return func(so *serverOptions) {
		so.ca = c
	}
}

// WithRunner returns an  Option that sets the runner to r
func WithRunner(r runner.Runner) Option {
	return func(so *serverOptions) {
		so.run = r
	}
}

// WithRTStore returns an Option that sets the RTStore to rts
func WithRTStore(rts RTStore) Option {
	return func(so *serverOptions) {
		so.rts = rts
	}
}

// WithVPSource returns an Option that sets the VPSource to vps
func WithVPSource(vps vpservice.VPSource) Option {
	return func(so *serverOptions) {
		so.vps = vps
	}
}

// WithAdjacencySource returns an Option that sets the AdjacencySource to as
func WithAdjacencySource(as types.AdjacencySource) Option {
	return func(so *serverOptions) {
		so.as = as
	}
}

func WithRKClient(rkc rk.RankingSource) Option {
	return func(so *serverOptions) {
		so.rkc = rkc
	}
} 

// WithClusterSource returns an Option that sets the ClusterSource to cs
func WithClusterSource(cs types.ClusterSource) Option {
	return func(so *serverOptions) {
		so.cs = cs
	}
}

// WithRootCA returns an Option that sets the rootCA to rootCA
func WithRootCA(rootCA string) Option {
	return func(so *serverOptions) {
		so.rootCA = rootCA
	}
}

// WithCertFile returns an Option that sets the certFile to certFile
func WithCertFile(certFile string) Option {
	return func(so *serverOptions) {
		so.certFile = certFile
	}
}

// WithKeyFile returns an Option that sets the keyFile to keyFile
func WithKeyFile(keyFile string) Option {
	return func(so *serverOptions) {
		so.keyFile = keyFile
	}
}

// NewRevtrServer creates an new Server with the given options
func NewRevtrServer(opts ...Option) RevtrServer {
	var serv revtrServer
	for _, opt := range opts {
		opt(&serv.opts)
	}
	// serv.nextID = new(uint32)
	s, err := connectToServices(serv.opts.rootCA)
	if err != nil {
		log.Fatalf("Could not connect to services: %v", err)
	}
	serv.s = s
	serv.vps = serv.opts.vps
	serv.cs = serv.opts.cs
	serv.rts = serv.opts.rts
	serv.as = serv.opts.as
	serv.run = serv.opts.run
	serv.cm = clustermap.New(serv.opts.cs, serv.opts.ca)
	serv.ca = serv.opts.ca
	serv.mu = &sync.Mutex{}
	serv.revtrs = make(map[uint32]revtrOutput)
	serv.running = make(map[uint32]<-chan *reversetraceroute.ReverseTraceroute)
	serv.revtrQueue = make(chan *revtrChannel, 100000)
	serv.flyingRevtr = make(chan int64, 12000) // Maximum number of parallel revtr in the system 
	serv.cacheTime = serv.opts.cacheTime
	serv.revtrStatsByUserMap = make(map[string]RevtrUserStats)
	serv.statsRevtrUserChan = make(chan RunningRevtrUser, 100000)
	serv.revtrInsertChan = make(chan pb.ReverseTraceroute, 1000000)
	// Build first before starting website so people can not run revtrs before having BGP tree loaded
	serv.ip2asnTree = buildIP2ASNTreeFromRouteViews()
	go serv.runRevtrFromQueue()
	go serv.storeRevtrFromQueue()
	go serv.statsUserUpdate()
	
	go serv.refreshIP2ASMapping()
	return serv
}

type ErrorBatchID struct {
	batchID uint32
	err     error 
}

type RunningRevtrUser struct {
	key string
	valueChange int
}

type RevtrUserStats struct {
	revtrRunToday uint32
	revtrFlyingParallel uint32
}

type revtrChannel struct {
	RevtrMeasurement  []*pb.RevtrMeasurement
	RevtrChannel      chan ErrorBatchID
	UserKey           string
} 

type revtrServer struct {
	rts    RTStore
	vps    vpservice.VPSource
	as     types.AdjacencySource
	cs     types.ClusterSource
	conf   types.Config
	run    runner.Runner
	opts   serverOptions
	// nextID *uint32
	s      services
	cm     clustermap.ClusterMap
	ca     types.Cache
	cacheTime int 
	// mu protects the revtr maps
	mu      *sync.Mutex
	// For the website calls?
	revtrs  map[uint32]revtrOutput
	running map[uint32]<-chan *reversetraceroute.ReverseTraceroute
	revtrQueue chan *revtrChannel
	flyingRevtr chan int64
	// Flying revtr per user
	revtrStatsByUserMap map[string]RevtrUserStats
	statsRevtrUserChan chan RunningRevtrUser

	// Insert queue
	revtrInsertChan chan pb.ReverseTraceroute

	// Mapping between the IP addresses and their ASN. 
	ip2asnTree *radix.BGPRadixTree

}

func buildIP2ASNTreeFromRouteViews() *radix.BGPRadixTree {

	// Write the current date in a file
	baseDir := "/revtr/ip2as"
	if environment.IsDebugRevtr {
		baseDir = "revtr/ip2as"
	}
	// gobIP2ASNTreeFile := fmt.Sprintf("%s/ipASN.gob", baseDir)
	// if environment.IsDebugRevtr {
	// 	// Do not reload for debugging
	// 	if _, err := os.Stat(gobIP2ASNTreeFile); err == nil {
	// 		gobFile, _ := os.Open(gobIP2ASNTreeFile)
	// 		defer gobFile.Close()
	// 		gobDecoder := gob.NewDecoder(gobFile)
	// 		var gobIP2ASNTree radix.BGPRadixTree
	// 		gobDecoder.Decode(&gobIP2ASNTree)

	// 		return &gobIP2ASNTree

	// 	}
	// }
	now := time.Now()
	nowStr := now.Format("20060102")
	dateFile := "bgp_dates"
	log.Infof("Writing date %s into %s/%s file", nowStr, baseDir, dateFile)

	f, err := os.Create(fmt.Sprintf("%s/%s", baseDir, dateFile))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	_, err = w.WriteString(fmt.Sprintf("%s\n", nowStr))
	if err != nil {
		panic(err)
	}
	w.Flush()

	// Download the last bgpdump 
	_, err = util.RunCMD(fmt.Sprintf("cd %s; pyasn_util_download.py -f %s ", baseDir, dateFile))
	if err != nil {
		log.Error(err)
	}

	// Move the rib file to another name
	ribFile := fmt.Sprintf("%s/latest_rib.bz2", baseDir)
	_, err = util.RunCMD(fmt.Sprintf("mv %s/%s %s ", baseDir, "rib*.bz2", ribFile))
	if err != nil {
		log.Error(err)
	}
	
	ipAsnFile := fmt.Sprintf("%s/ipASN.dat", baseDir)
	_, err = util.RunCMD(fmt.Sprintf("pyasn_util_convert.py --single %s %s", ribFile, ipAsnFile))
	if err != nil {
		log.Error(err)
	}

	ip2AsnTree := radix.CreateRadix(ipAsnFile)

	// Adding IXP IPs in the tree
	ixpIPsFilePath := fmt.Sprintf("%s/%s", baseDir, "interfaces_ixp-20210504.txt")

	// Open file 

	ixpIPsFile, err := os.Open(ixpIPsFilePath)
    if err != nil {
        log.Fatal(err)
    }
    defer ixpIPsFile.Close()

    scanner := bufio.NewScanner(ixpIPsFile)
    // optionally, resize scanner's capacity for lines over 64K, see next example
    for scanner.Scan() {
        line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Split line to get IP address
		tokens := strings.Split(line, "\t")
		ip := tokens[0]
		asn := tokens[1]
		if asn == "None" {
			continue
		}
		asnI, err := strconv.Atoi(asn)
		if err != nil {
			log.Error(err)
			continue
		}
		ip2AsnTree = radix.Insert(ip2AsnTree, ip, "32", asnI)
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

	gobIP2ASNTree := radix.BGPRadixTree{
		Tree: *ip2AsnTree,
	}

	// Serialize the tree 
	// gobFile, err := os.Create(gobIP2ASNTreeFile)
	// if err != nil {
	// 	log.Error(err)
	// }
	// defer gobFile.Close()
	// gobEncoder := gob.NewEncoder(gobFile)
	// gobEncoder.Encode(gobIP2ASNTree)
	
	return &gobIP2ASNTree
	
}

func (rs revtrServer) refreshIP2ASMapping() {
	// Every day refresh the radix tree

	t := time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-t.C:
			rs.ip2asnTree = buildIP2ASNTreeFromRouteViews()
		}
	}

}

func (rs revtrServer) statsUserUpdate ()  {

	// Every 24 hours, reset the map. 
	t := time.NewTicker(24 * time.Hour)
	
	tLogs := time.NewTicker(2 * time.Second)

	go func() {
		for {
			select {
			case <-tLogs.C:
				for user, stats := range(rs.revtrStatsByUserMap) {
					if stats.revtrFlyingParallel > 0 {
						log.Infof("User: %s, ongoing revtrs: %d", user, stats.revtrFlyingParallel)	
					}
					
				}
			case ur := <-rs.statsRevtrUserChan:
				if _, ok := rs.revtrStatsByUserMap[ur.key]; !ok {
					// Lazy initialization
					rs.revtrStatsByUserMap[ur.key] = RevtrUserStats{}
				}
				revtrStatsUser := rs.revtrStatsByUserMap[ur.key]
				if ur.valueChange > 0 {
					// We are adding a reverse traceroute
					rs.revtrStatsByUserMap[ur.key] =
					 RevtrUserStats {revtrFlyingParallel: revtrStatsUser.revtrFlyingParallel + uint32(ur.valueChange),
						 revtrRunToday: revtrStatsUser.revtrRunToday + uint32(ur.valueChange)} 
				} else {
					// Do not decrement the number of revtrs run today for this user. 
					rs.revtrStatsByUserMap[ur.key] =
					 RevtrUserStats {revtrFlyingParallel: revtrStatsUser.revtrFlyingParallel + uint32(ur.valueChange),
						 revtrRunToday: revtrStatsUser.revtrRunToday} 
				}
			
			case <-t.C:
				// Reset the counter of revtrs run today
				for user, stats := range(rs.revtrStatsByUserMap) {
					rs.revtrStatsByUserMap[user] = RevtrUserStats{revtrFlyingParallel: stats.revtrFlyingParallel, revtrRunToday: 0}
				}
			}
		}
	} ()

}

func (rs revtrServer) storeRevtrFromQueue() {
	// Batch the inserts to not overload the DB
	ticker := time.NewTicker(1 * time.Second)
	revtrsToInsert := []pb.ReverseTraceroute{}
	batchSizeRevtr := 1000

	for {
		select {
		case <-ticker.C:
			if len(revtrsToInsert) > 0 {
				// Flush what we can flush 
				log.Debugf("Flushing batch of %d revtrs", len(revtrsToInsert))
				revtrs := make([]pb.ReverseTraceroute, len(revtrsToInsert))
				copy(revtrs, revtrsToInsert)
				go rs.rts.StoreBatchedRevtrs(revtrsToInsert)
				revtrsToInsert = nil
			}
		case revtr := <- rs.revtrInsertChan:
			// log.Infof("Inserting one element in the queue %d", len(pingsToInsert))
			revtrsToInsert = append(revtrsToInsert, revtr)
			// log.Infof("Queuing batch of %d pings", len(pingsToInsert))
			// If we have enough pings, create a query
			if len(revtrsToInsert) == batchSizeRevtr {
				log.Debugf("Flushing batch of %d revtr", len(revtrsToInsert))
				revtrs := make([]pb.ReverseTraceroute, len(revtrsToInsert))
				copy(revtrs, revtrsToInsert)
				go rs.rts.StoreBatchedRevtrs(revtrsToInsert)
				revtrsToInsert = nil
			}
		}	
	}
}

func (rs revtrServer) runRevtrFromQueue(){
	servs, err := connectToServices(rs.opts.rootCA)
	if err != nil {
		log.Error(err)
		// return nil, ErrConnectFailed
	}
	defer servs.Close()
	done := make(chan bool)

	limiter := time.NewTicker(3 * time.Millisecond)


	for {
		select {
		case <-done:
			return
		case r := <-rs.revtrQueue:
			// Check that the user has enough "credits" to run the measurement he's trying to run 
			user, _ := rs.rts.GetUserByKey(r.UserKey) 
			revtrStatsUser := rs.revtrStatsByUserMap[user.Key]
			if int(revtrStatsUser.revtrRunToday) + len(r.RevtrMeasurement) > int(user.MaxRevtrPerDay) {
				r.RevtrChannel <- ErrorBatchID{batchID: 0, err: types.ErrTooManyMesaurementsToday}
				break
			}

			
			if revtrStatsUser.revtrFlyingParallel + uint32(len(r.RevtrMeasurement)) > user.MaxParallelRevtr {
				r.RevtrChannel <- ErrorBatchID{batchID: 0, err: types.ErrTooManyMesaurementsInParallel}
				break
			} 
			
			<-limiter.C
			log.Infof("Creating %d reverse traceroutes", len(r.RevtrMeasurement))
			reqToRun, batchID, err := rs.rts.CreateRevtrBatch(r.RevtrMeasurement, r.UserKey)
			if err == repo.ErrCannotAddRevtrBatch {
				log.Error(err)
				panic(err)
			}

			r.RevtrChannel <- ErrorBatchID{batchID:batchID, err: nil}
			
			for i, r := range reqToRun {
				if i % 10000 == 0 {
					log.Infof("Starting %d th reverse traceroutes", i)
				}
				rs.flyingRevtr <- int64(r.Id)
				// add the revtr to the number of revtr flying for this user. 
				rs.statsRevtrUserChan <- RunningRevtrUser{key:user.Key, valueChange: 1}
				rtr := reversetraceroute.CreateReverseTraceroute(
						*r,
						rs.cs,
						nil,
						nil,
						nil,
				)
				
				go func() {
					runningRevtrs.Add(float64(1))
					var rtrs []*reversetraceroute.ReverseTraceroute
					rtrs = append(rtrs, rtr)
					done := rs.run.Run(rtrs,
						runner.WithContext(context.Background()),
						runner.WithClient(servs.cl),
						runner.WithAtlas(servs.at),
						runner.WithVPSource(servs.vpserv),
						runner.WithRKClient(servs.rks),
						runner.WithAdjacencySource(rs.as),
						runner.WithClusterMap(rs.cm),
						runner.WithCache(rs.ca),
						runner.WithCacheTime(rs.cacheTime),		
						runner.WithIP2AS(rs.ip2asnTree),
					) 
					for drtr := range done {
						runningRevtrs.Sub(1)
						revtr := drtr.ToStorable(servs.at)
						rs.revtrInsertChan <- revtr
						// err = rs.rts.StoreBatchedRevtrs([]pb.ReverseTraceroute{})
						// if err != nil {
						// 	log.Errorf("Error storing Revtr(%d): %v", drtr.ID, err)
						// }
						// Removing one element from flying reverse traceroutes 
						<-rs.flyingRevtr
						// Remove the reverse traceroute from the flying for the user
						rs.statsRevtrUserChan <- RunningRevtrUser{key:user.Key, valueChange: -1}
					}
				}()
				<-limiter.C
			}
			
		}
	}
}

// func (rs revtrServer) getID() uint32 {
// 	return atomic.AddUint32(rs.nextID, 1)
// }

func (rs revtrServer) RunRevtr(req *pb.RunRevtrReq) (*pb.RunRevtrResp, error) {
	user, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
		return nil, types.ErrUserNotFound
	}

	// vps, err := rs.vps.GetVPs() // Get Vantage Points here.
	// if err != nil {
	// 	log.Error(err)
	// 	return nil, err
	// }
	var reqToRun []*pb.RevtrMeasurement
	// var srcDstPairs map[types.SrcDstPair] struct{}
	// if req.CheckDB {
	// 	srcDstPairs, err = rs.rts.ExistsRevtr(req.GetRevtrs()[0].Label)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	sources, err := rs.vps.GetVPs()
	
	sourcesSet := map[uint32]struct{}{}
	for _, source := range(sources.Vps) {
		sourcesSet[source.Ip] = struct{}{}
	}

	for i, r := range req.GetRevtrs() {
		if i % 100000 == 0 {
			log.Infof("Checking %d th revtr", i)
		}
		
		// Check that the source asked is in the DB
		log.Debugf("checking revtr %s and destination %s", r.Src, r.Dst)
		srcI, _ := util.IPStringToInt32(r.Src)
		dstI, _ := util.IPStringToInt32(r.Dst)
		if _, ok := sourcesSet[srcI]; !ok {
			log.Errorf("Cannot run reverse traceroute with source %s (%d) and destination %s (%d)",
			r.Src, srcI, r.Dst, dstI)
			// return nil, types.ErrIncorrectSource
		} else {
			reqToRun = append(reqToRun, r)
		}


		// src, dst, err := verifyAddrs(r.Src, r.Dst, vps.GetVps())
		// if err != nil {
		// 	log.Errorf("Cannot run reverse traceroute from %s to %s, %s", src, dst, err)
		// 	return nil, types.ErrIncorrectSource
		// }
		// r.Src = src
		// r.Dst = dst

		// In the particular case of a large survey, check if we already have 
		// the reverse traceroute with the same label/ src/ dst 
		
		// if req.CheckDB {	
		// 	srcI, _ := util.IPStringToInt32(r.Src)
		// 	dstI, _ := util.IPStringToInt32(r.Dst)
		// 	if _, isExists := srcDstPairs[types.SrcDstPair{Src: srcI, Dst:dstI}]; isExists{
		// 		continue
		// 	}
		// }

		// src, _ = util.Int32ToIPString(r.Src)
		// dst, _ = util.Int32ToIPString(r.Dst)
	}

	if len(reqToRun) == 0 {
		return nil, types.ErrNoRevtrsToRun
	}	

	
	// servs, err := connectToServices(rs.opts.rootCA)
	// if err != nil {
	// 	log.Error(err)
	// 	return nil, ErrConnectFailed
	// }
	// reqToRun, batchID, err := rs.rts.CreateRevtrBatch(reqToRun, user_key)
	// if err == repo.ErrCannotAddRevtrBatch {
	// 	log.Error(err)
	// 	return nil, ErrFailedToCreateBatch
	// }

	// Push the measurements into a global queue that rate limits 
	// burst of revtr and burst to the db
	// Create a chanel 
	revtrBatchChannel := make(chan ErrorBatchID, 1)
	
	rs.revtrQueue <- &revtrChannel{
		RevtrMeasurement: reqToRun,
		RevtrChannel: revtrBatchChannel,
		UserKey: user.Key,
	}



	errorBatchID := <- revtrBatchChannel
	if errorBatchID.err != nil {
		log.Errorf(errorBatchID.err.Error())
		return nil, errorBatchID.err
	}
	// // run these guys
	// go func() {
	// 	defer servs.Close()
	// 	runningRevtrs.Add(float64(len(reqToRun)))
	// 	var rtrs []*reversetraceroute.ReverseTraceroute
	// 	for _, r := range reqToRun {
	// 		rtrs = append(rtrs, reversetraceroute.CreateReverseTraceroute(
	// 			*r,
	// 			rs.cs,
	// 			nil,
	// 			nil,
	// 			nil,
	// 		))
	// 	}
	// 	done := rs.run.Run(rtrs,
	// 		runner.WithContext(context.Background()),
	// 		runner.WithClient(servs.cl),
	// 		runner.WithAtlas(servs.at),
	// 		runner.WithVPSource(servs.vpserv),
	// 		runner.WithRKClient(servs.rks),
	// 		runner.WithAdjacencySource(rs.as),
	// 		runner.WithClusterMap(rs.cm),
	// 		runner.WithCache(rs.ca),		
	// 	)
	// 	for drtr := range done {
	// 		runningRevtrs.Sub(1)
	// 		err = rs.rts.StoreBatchedRevtrs([]pb.ReverseTraceroute{drtr.ToStorable()})
	// 		if err != nil {
	// 			log.Errorf("Error storing Revtr(%d): %v", drtr.ID, err)
	// 		}
	// 	}
	// }()
	
	return &pb.RunRevtrResp{
		BatchId: errorBatchID.batchID,
	}, nil
}

func (rs revtrServer) GetRevtr(req *pb.GetRevtrReq) (*pb.GetRevtrResp, error) {
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if req.BatchId == 0 {
		return nil, BatchIDError{batchID: req.BatchId}
	}
	
	revtrs, err := rs.rts.GetRevtrsInBatch(usr.Id, req.BatchId)
	if err != nil {
		return nil, err
	}	
	
	return &pb.GetRevtrResp{
	Revtrs: revtrs,
	}, nil
}

func (rs revtrServer) GetRevtrByLabel(req *pb.GetRevtrByLabelReq) (*pb.GetRevtrByLabelResp, error) {
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	
	revtrs, err := rs.rts.GetRevtrsByLabel(usr.Id, req.Label)
	if err != nil {
		return nil, err
	}	
	
	return &pb.GetRevtrByLabelResp{
	Revtrs: revtrs,
	}, nil
}

func (rs revtrServer) GetRevtrsMetaOnly(req *pb.GetRevtrMetaOnlyReq) (*pb.GetRevtrMetaOnlyResp, error){
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if req.BatchId == 0 {
		return nil, BatchIDError{batchID: req.BatchId}
	}

	revtrsMetaOnly, err := rs.rts.GetRevtrsMetaOnly(usr.Id, req.BatchId)
	if err != nil {
		return nil, err
	}
	return &pb.GetRevtrMetaOnlyResp{
		RevtrsMeta: revtrsMetaOnly,
	}, nil
}

func (rs revtrServer) GetRevtrBatchStatus(req *pb.GetRevtrBatchStatusReq) (*pb.GetRevtrBatchStatusResp, error) {
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if req.BatchId == 0 {
		return nil, BatchIDError{batchID: req.BatchId}
	}
	
	revtrsStatus, err := rs.rts.GetRevtrsBatchStatus(usr.Id, req.BatchId)
	if err != nil {
		return nil, err
	}	
	
	return &pb.GetRevtrBatchStatusResp{
	RevtrsStatus: revtrsStatus,
	}, nil
}

func (rs revtrServer) UpdateRevtr(req *pb.UpdateRevtrReq) (*pb.UpdateRevtrResp, error) {
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = rs.rts.UpdateRevtr(usr.Id, req.RevtrId,req.TracerouteId)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateRevtrResp{}, nil

}

func (rs revtrServer) GetSources(req *pb.GetSourcesReq) (*pb.GetSourcesResp, error) {
	_, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	vps, err := rs.vps.GetVPs()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	sr := &pb.GetSourcesResp{}
	for _, vp := range vps.GetVps() {
		s := new(pb.Source)
		s.Hostname = vp.Hostname
		s.Ip, _ = util.Int32ToIPString(vp.Ip)
		s.Site = vp.Site
		sr.Srcs = append(sr.Srcs, s)
	}
	return sr, nil
}

type revtrOutput struct {
	rt *reversetraceroute.ReverseTraceroute
	oc chan Status
}

func (rs revtrServer) AddRevtr(rtm pb.RevtrMeasurement) (uint32, error) {
	vps, err := rs.vps.GetVPs()
	if err != nil {
		log.Error(err)
		return 0, err
	}
	src, dst, err := verifyAddrs(rtm.Src, rtm.Dst, vps.GetVps())
	if err != nil {
		return 0, err
	}
	rtm.Src = src
	rtm.Dst = dst
	rs.mu.Lock()
	defer rs.mu.Unlock()
	// id := rs.getID()
	oc := make(chan Status, 5)
	pf := makePrintHTML(oc, rs)
	
	rtmWithID, _, err := rs.rts.CreateRevtrBatch([]*pb.RevtrMeasurement{&rtm}, "revtr_website_key")
	if err != nil || len(rtmWithID) != 1 {
		log.Error(err)
		return 0, err
	}
	rt := reversetraceroute.CreateReverseTraceroute(*rtmWithID[0],
		rs.cs,
		reversetraceroute.OnAddFunc(pf),
		reversetraceroute.OnFailFunc(pf),
		reversetraceroute.OnReachFunc(pf))
	// print out the initial state
	pf(rt)
	rs.revtrs[rt.ID] = revtrOutput{rt: rt, oc: oc}
	return rt.ID, nil
}

func (rs revtrServer) CleanAtlas(req *pb.CleanAtlasReq) (*pb.CleanAtlasResp, error) {
	_, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	sourceI, err := util.IPStringToInt32(req.Source)
	err = rs.s.at.MarkTracerouteStaleSource(context.Background(), sourceI)
	return &pb.CleanAtlasResp{}, err 
}

func (rs revtrServer) RunAtlas(req *pb.RunAtlasReq) (*pb.RunAtlasResp, error) {
	_, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	sourceI, err := util.IPStringToInt32(req.Source)
	err = rs.s.at.RunTracerouteAtlasToSource(context.Background(), sourceI)
	return &pb.RunAtlasResp{}, err 
}

// IDError is returned when StartRevtr is called with an invalid id
type IDError struct {
	id uint32
}

func (ide IDError) Error() string {
	return fmt.Sprintf("invalid id %d", ide.id)
}

// Status represents the current running state of a reverse traceroute
// it is use for the web interface. Something better is probably needed
type Status struct {
	Rep    string
	Status bool
	Error  string
}

func (rs revtrServer) StartRevtr(ctx context.Context, id uint32) (<-chan Status, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rt, ok := rs.revtrs[id]; ok {

		go func() {
			if _, ok := rs.running[id]; ok {
				return
			}
			runningRevtrs.Add(1)
			rs.mu.Lock()
			rtrs := []*reversetraceroute.ReverseTraceroute{rt.rt}
			rs.flyingRevtr <- int64(1)
			done := rs.run.Run(rtrs,
				runner.WithContext(context.Background()),
				runner.WithRKClient(rs.s.rks),
				runner.WithClient(rs.s.cl),
				runner.WithAtlas(rs.s.at),
				runner.WithVPSource(rs.s.vpserv),
				runner.WithAdjacencySource(rs.as),
				runner.WithClusterMap(rs.cm),
				runner.WithCache(rs.ca),
				runner.WithIP2AS(rs.ip2asnTree),
				
			)
			rs.running[id] = done
			rs.mu.Unlock()
			for drtr := range done {
				runningRevtrs.Sub(1)
				rs.mu.Lock()
				delete(rs.revtrs, id)
				delete(rs.running, id)
				// done, close the channel
				close(rt.oc)
				rs.mu.Unlock()
				revtr := drtr.ToStorable(rs.s.at)
				rs.revtrInsertChan <- revtr
				// Removing one element from flying reverse traceroutes 
				<-rs.flyingRevtr
				// err := rs.rts.StoreBatchedRevtrs([]pb.ReverseTraceroute{drtr.ToStorable(rs.s.at)})
				// if err != nil {
				// 	log.Error(err)
				// }
			}
		}()
		return rt.oc, nil
	}
	return nil, IDError{id: id}
}

func (rs revtrServer) resolveHostname(ip string) string {
	// TODO clean up error logging
	item, ok := rs.ca.Get(ip)
	// If it's a cache miss, look through the vps
	// if its found set it.
	if !ok {
		vps, err := rs.vps.GetVPs()
		if err != nil {
			log.Error(err)
			return ""
		}
		for _, vp := range vps.GetVps() {
			ips, _ := util.Int32ToIPString(vp.Ip)
			// found it, set the cache and return the hostname
			if ips == ip {
				rs.ca.Set(ip, vp.Hostname, time.Hour*4)
				return vp.Hostname
			}
		}
		// not found from vps try reverse lookup
		hns, err := net.LookupAddr(ip)
		if err != nil {
			log.Error(err)
			// since the lookup failed, just set it blank for now
			// after it expires we'll try again
			rs.ca.Set(ip, "", time.Hour*4)
			return ""
		}
		rs.ca.Set(ip, hns[0], time.Hour*4)
		return hns[0]
	}
	return item.(string)
}

func (rs revtrServer) getRTT(src, dst string) float32 {
	key := fmt.Sprintf("%s:%s:rtt", src, dst)
	item, ok := rs.ca.Get(key)
	if !ok {
		targ, _ := util.IPStringToInt32(dst)
		src, _ := util.IPStringToInt32(src)
		ping := &datamodel.PingMeasurement{
			Src:     src,
			Dst:     targ,
			Count:   "1",
			Timeout: 5,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		st, err := rs.s.cl.Ping(ctx,
			&datamodel.PingArg{Pings: []*datamodel.PingMeasurement{ping}})
		if err != nil {
			log.Error(err)
			rs.ca.Set(key, float32(0), time.Minute*30)
			return 0
		}
		for {
			p, err := st.Recv()
			if err == io.EOF {
				break

			}
			if err != nil {
				log.Error(err)
				rs.ca.Set(key, float32(0), time.Minute*30)
				return 0

			}
			if len(p.Responses) == 0 {
				rs.ca.Set(key, float32(0), time.Minute*30)
				return 0
			}
			rs.ca.Set(key, float32(p.Responses[0].Rtt)/1000, time.Minute*30)
			return float32(p.Responses[0].Rtt) / 1000
		}
	}
	return item.(float32)
}

func makePrintHTML(sc chan<- Status, rs revtrServer) func(*reversetraceroute.ReverseTraceroute) {
	return func(rt *reversetraceroute.ReverseTraceroute) {
		//need to find hostnames and rtts
		hopsSeen := make(map[string]bool)
		var out bytes.Buffer
		out.WriteString(`<table class="table">`)
		out.WriteString(`<caption class="text-center">Reverse Traceroute from `)
		out.WriteString(fmt.Sprintf("%s (%s) back to ", rt.Dst, rs.resolveHostname(rt.Dst)))
		out.WriteString(rt.Src)
		out.WriteString(fmt.Sprintf(" (%s)", rs.resolveHostname(rt.Src)))
		out.WriteString("</caption>")
		out.WriteString(`<tbody>`)
		first := new(bool)
		var i int
		if len(*rt.Paths) > 0 {

			for _, segment := range *rt.CurrPath().Path {
				*first = true
				symbol := new(string)
				switch segment.(type) {
				case *reversetraceroute.DstSymRevSegment:
					*symbol = "sym"
				case *reversetraceroute.DstRevSegment:
					*symbol = "dst"
				case *reversetraceroute.TRtoSrcRevSegment:
					*symbol = "tr"
				case *reversetraceroute.SpoofRRRevSegment:
					*symbol = "rr"
				case *reversetraceroute.RRRevSegment:
					*symbol = "rr"
				case *reversetraceroute.SpoofTSAdjRevSegmentTSZeroDoubleStamp:
					*symbol = "ts"
				case *reversetraceroute.SpoofTSAdjRevSegmentTSZero:
					*symbol = "ts"
				case *reversetraceroute.SpoofTSAdjRevSegment:
					*symbol = "ts"
				case *reversetraceroute.TSAdjRevSegment:
					*symbol = "ts"
				}
				for _, hop := range segment.Hops() {
					if hopsSeen[hop] {
						continue

					}
					hopsSeen[hop] = true
					tech := new(string)
					if *first {
						*tech = *symbol
						*first = false

					} else {
						*tech = "-" + *symbol

					}
					if hop == "0.0.0.0" || hop == "*" {
						out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s</td><td></td><td>%s</td></tr>", i, "* * *", *tech))

					} else {
						out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s (%s)</td><td>%.3fms</td><td>%s</td></tr>", i, hop, rs.resolveHostname(hop), rs.getRTT(rt.Src, hop), *tech))

					}
					i++

				}

			}

		}
		out.WriteString("</tbody></table>")
		out.WriteString(fmt.Sprintf("\n%s", rt.StopReason))
		var showError = rt.StopReason == reversetraceroute.Failed
		var errorText string
		if showError {
			errorText = strings.Replace(rt.ErrorDetails.String(), "\n", "<br>", -1)

		}
		var stat Status
		stat.Rep = strings.Replace(out.String(), "\n", "<br>", -1)
		stat.Status = rt.StopReason != ""
		stat.Error = errorText
		select {
		case sc <- stat:
		default:
		}
	}
}

type services struct {
	cl     client.Client
	clc    *grpc.ClientConn
	at     at.Atlas
	atc    *grpc.ClientConn
	vpserv vpservice.VPSource
	vpsc   *grpc.ClientConn
	rks	   rk.RankingSource
	rksc   *grpc.ClientConn
}

func (s services) Close() error {
	var err error
	if s.clc != nil {
		err = s.clc.Close()
	}
	if s.atc != nil {
		err = s.atc.Close()
	}
	if s.vpsc != nil {
		err = s.vpsc.Close()
	}
	if s.rksc != nil {
		err = s.rksc.Close()
	}
	return err
}

func connectToServices(rootCA string) (services, error) {
	var ret services
	insecureOptionGrpc := grpc.WithInsecure()
	cc, _:= grpc.Dial("dummy") // To be able to shut down in narrow scope
	c2, _:= grpc.Dial("dummy") // To be able to shut down in narrow scope
	c3, _:= grpc.Dial("dummy") // To be able to shut down in narrow scope
	if !environment.IsDebugController {
		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			return ret, err
		}
		ccreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			return ret, err
		}
		connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		cc, err = grpc.Dial(connstr, grpc.WithTransportCredentials(ccreds))
		if err != nil {
			return ret, err
		}
		cli := client.New(context.Background(), cc)
		ret.cl = cli
		ret.clc = cc
	} else {
		// Controller
		connstr := fmt.Sprintf("%s:%d", "localhost", environment.ControllerPortDebug) // 4382 is the production one
		cc, err := grpc.Dial(connstr, insecureOptionGrpc)
		if err != nil {
			return ret, err
		}
		cli := client.New(context.Background(), cc)
		ret.cl = cli
		ret.clc = cc
	}
	if !environment.IsDebugAtlas{
		_, srvs, err := net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			logError(cc.Close)
			return ret, err
		}
		atcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			logError(cc.Close)
			return ret, err
		}
		connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c2, err = grpc.Dial(connstrat, grpc.WithTransportCredentials(atcreds))
		if err != nil {
			logError(cc.Close)
			return ret, err
		}
		atl := at.New(context.Background(), c2)
		ret.at = atl
		ret.atc = c2
	} else {
		// Atlas 
		connstrat := fmt.Sprintf("%s:%d", "localhost", environment.AtlasPortDebug) // 55000 is the default
		c2, err := grpc.Dial(connstrat, insecureOptionGrpc)
		if err != nil {
			logError(cc.Close)
			return ret, err
		}
		atl := at.New(context.Background(), c2)
		ret.at = atl
		ret.atc = c2
	}
	if !environment.IsDebugVPService{
		_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			return ret, err
		}
		vpcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			logError(cc.Close)
			logError(c2.Close)
			return ret, err
		}
		connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
		if err != nil {
			logError(cc.Close)
			logError(c2.Close)
			return ret, err
		}
		vps := vpservice.New(context.Background(), c3)
		ret.vpserv = vps
		ret.vpsc = c3
	} else  {
		// VPService
		connvp := fmt.Sprintf("%s:%d", "localhost", environment.VPServicePortDebug) // 45000 is the production one
		c3, err := grpc.Dial(connvp, insecureOptionGrpc)
		if err != nil {
			logError(cc.Close)
			logError(c2.Close)
			return ret, err
		}
		vps := vpservice.New(context.Background(), c3)
		ret.vpserv = vps
		ret.vpsc = c3
	}

	connrs := ""
	if !environment.IsDebugRankingService{ 
		_, srvs, err := net.LookupSRV("rankingservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatal(err)
		}
		rscreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
		}
		connrs = fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c4, err := grpc.Dial(connrs, grpc.WithTransportCredentials(rscreds))
		if err != nil {
			logError(cc.Close)
			logError(c2.Close)
			logError(c3.Close)
			return ret, err
		} 
		rks := rk.New(context.Background(), c4)
		ret.rks = rks
		ret.rksc = c4

	} else {
		connrs = fmt.Sprintf("%s:%d", "localhost", environment.RankingServicePortDebug)
		c4, err := grpc.Dial(connrs, insecureOptionGrpc)
		if err != nil {
			logError(cc.Close)
			logError(c2.Close)
			logError(c3.Close)
			return ret, err
		}
		rks := rk.New(context.Background(), c4)
		ret.rks = rks
		ret.rksc = c4
	}
	
	return ret, nil
}
