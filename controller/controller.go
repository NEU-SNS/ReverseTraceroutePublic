/*
 Copyright (c) 2015, Northeastern University
, r All rights reserved.

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

// Package controller is the library for creating a central controller
package controller

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	// "math"

	ca "github.com/NEU-SNS/ReverseTraceroute/cache"
	controllerapi "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/router"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/prometheus/client_golang/prometheus"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	nameSpace     = "controller"
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
	rpcCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "rpc",
		Name:      "count",
		Help:      "Count of Rpc Calls sent",
	})
	timeoutCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "rpc",
		Name:      "timeout_count",
		Help:      "Count of Rpc Timeouts",
	})
	errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "rpc",
		Name:      "error_count",
		Help:      "Count of Rpc Errors",
	})
	pingResponseTimes = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: nameSpace,
		Subsystem: "measurements",
		Name:      "ping_response_times",
		Help:      "The time it takes for pings to respond",
	})
	tracerouteResponseTimes = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: nameSpace,
		Subsystem: "measurements",
		Name:      "traceroute_response_times",
		Help:      "The time it takes for traceroutes to respond",
	})
	pingsFromCache = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "cache",
		Name:      "pings_from_cache",
		Help:      "The number of pings retrieved from the cache.",
	})
	pingsFromDB = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "db",
		Name:      "pings_from_db",
		Help:      "The number of pings retrieved from the db.",
	})
	tracesFromCache = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "cache",
		Name:      "traces_from_cache",
		Help:      "The number of traceroutes retrieved from the cache.",
	})
	tracesFromDB = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: nameSpace,
		Subsystem: "db",
		Name:      "traces_from_db",
		Help:      "The number of traceroutes retrieved from the db.",
	})
)

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(rpcCounter)
	prometheus.MustRegister(timeoutCounter)
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(pingResponseTimes)
	prometheus.MustRegister(tracerouteResponseTimes)
	prometheus.MustRegister(pingsFromCache)
	prometheus.MustRegister(pingsFromDB)
	prometheus.MustRegister(tracesFromCache)
	prometheus.MustRegister(tracesFromDB)
}

// DataAccess defines the interface needed by a DB
type DataAccess interface {
	GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error)
	GetPingsMulti([]*dm.PingMeasurement) ([]*dm.Ping, error)
	StorePing(*dm.Ping) (int64, error)
	StorePingBulk(ps [] *dm.Ping) ([]int64, error)
	GetTRBySrcDst(uint32, uint32) ([]*dm.Traceroute, error)
	GetTraceMulti([]*dm.TracerouteMeasurement) ([]*dm.Traceroute, error)
	StoreTraceroute(*dm.Traceroute) (int64, error)
	StoreTracerouteBulk([]*dm.Traceroute) ([]int64, error)
	Close() error
	GetUser(string) (dm.User, error)
	AddPingBatch(dm.User) (int64, error)
	AddPingsToBatch(int64, []int64) error
	GetPingBatch(dm.User, int64) ([]*dm.Ping, error)
	GetPingBatchNResponses(dm.User, int64)(uint32, error)
	AddTraceBatch(dm.User) (int64, error)
	AddTraceToBatch(int64, []int64) error
	GetTraceBatch(dm.User, int64) ([]*dm.Traceroute, error)
	Monitor()(sql.DBStats, sql.DBStats)
	GetMaxRevtrMeasurementID(string) (uint64, error)
}

type controllerT struct {
	config  Config
	db      DataAccess
	cache   ca.Cache
	router  router.Router
	server  *grpc.Server
	spoofID uint32
	sm      *spoofMap
	// To limit number of measurement sent to the PL controller
	// queue   chan int  
	flyingTraceroutes uint32
	flyingPings uint32

	sitePerVP  map[uint32]string
	sitePerVPLock sync.RWMutex
	// For DB rate limit
	queueInsertTokens chan int
	queueInsertPing chan * dm.Ping
	queueInsertTraceroute chan *dm.Traceroute
	maxParallelInsertQueries int 
	maxRevtrMeasurementIDPing uint64
	maxRevtrMeasurementIDTraceroute uint64
}

var controllerInstance controllerT

func (c *controllerT) flushPingBatch(pings []*dm.Ping) {
	<-c.queueInsertTokens	
	_, err := c.db.StorePingBulk(pings)
	if err != nil {
		log.Error(err)
	}
	c.queueInsertTokens <- 0
}

func (c *controllerT) flushTracerouteBatch(traceroutes []*dm.Traceroute) {
	<-c.queueInsertTokens	
	c.db.StoreTracerouteBulk(traceroutes)
	c.queueInsertTokens <- 0
}

func (c *controllerT) insertIntoDB() {
	// Every second, insert what's in the queue if we have not sent what was in the queue before
	t := time.NewTicker(2 * time.Second)
	pingsToInsert := [] *dm.Ping {}
	traceroutesToInsert := [] *dm.Traceroute {} 
	batchSizePing := 20000
	batchSizeTraceroute := 1000

	// Start by filling tokens 
	for i := 0; i < c.maxParallelInsertQueries; i++ {
		c.queueInsertTokens <- 0
	}

	for {
		select{
		case <-t.C:
			// log.Infof("Size of the ping queue %d", len(pingsToInsert))
			if len(pingsToInsert) > 0 {
				// Flush what we can flush 
				log.Infof("Flushing batch of %d pings", len(pingsToInsert))
				pings := make([]*dm.Ping, len(pingsToInsert))
				copy(pings, pingsToInsert)
				go c.flushPingBatch(pingsToInsert)
				pingsToInsert = nil
			}

			if len(traceroutesToInsert) > 0 {
				log.Infof("Flushing batch of %d traceroutes", len(traceroutesToInsert))
				traceroutes := make([]*dm.Traceroute, len(traceroutesToInsert))
				copy(traceroutes, traceroutesToInsert)
				go c.flushTracerouteBatch(traceroutes)
				traceroutesToInsert = nil
			}
			
		case ping := <- c.queueInsertPing:
			// log.Infof("Inserting one element in the queue %d", len(pingsToInsert))
			pingsToInsert = append(pingsToInsert, ping)
			// log.Infof("Queuing batch of %d pings", len(pingsToInsert))
			// If we have enough pings, create a query
			if len(pingsToInsert) == batchSizePing {
				log.Infof("Flushing batch of %d pings", len(pingsToInsert))
				pings := make([]*dm.Ping, len(pingsToInsert))
				copy(pings, pingsToInsert)
				go c.flushPingBatch(pings)
				pingsToInsert = nil
			}
		case traceroute := <- c.queueInsertTraceroute:
			traceroutesToInsert = append(traceroutesToInsert, traceroute)	
			if len(traceroutesToInsert) == batchSizeTraceroute {
				log.Infof("Flushing batch of %d traceroutes", len(traceroutesToInsert))
				traceroutes := make([]*dm.Traceroute, len(traceroutesToInsert))
				copy(traceroutes, traceroutesToInsert)
				go c.flushTracerouteBatch(traceroutes)
				traceroutesToInsert = nil
			}
		}
		
	}
}

func (c *controllerT) monitorCache() {
	t := time.NewTicker(5 * time.Second)
	
	for {
		select{
		case <-t.C:
			cacheRequests := c.cache.Monitor()
			if cacheRequests < c.cache.GetMaxTokens() {
				log.Infof("Number of cache ongoing requests: %d", cacheRequests)
			}
		}
	}
}

func (c *controllerT) monitorDB() {
	t := time.NewTicker(5 * time.Second)
	
	for {
		select{
		case <-t.C:
			dbReaderStats, dbWriterStats := c.db.Monitor()
			if dbReaderStats.InUse > 0 {
				log.Infof("Number of in use / open /max  connections reader: %d %d %d", 
				dbReaderStats.InUse, dbReaderStats.OpenConnections, dbReaderStats.MaxOpenConnections)
			} 
			if dbWriterStats.InUse > 0 {
				log.Infof("Number of in use / open /max  connections writer: %d %d %d", 
				dbWriterStats.InUse, dbWriterStats.OpenConnections, dbWriterStats.MaxOpenConnections)
			} 
			
		}
	}
}

func (c *controllerT) monitorTraceroutes () {
	t := time.NewTicker(5 * time.Second)
	
	for {
		select{
		case <-t.C:
			if c.flyingTraceroutes > 0 {
				log.Infof("Number of flying traceroutes %d", c.flyingTraceroutes)
			}
			
		}
	}
}

func (c *controllerT) monitorPings () {
	t := time.NewTicker(10 * time.Second)
	
	for {
		select{
		case <-t.C:
			if c.flyingPings > 0 {
				log.Infof("Number of flying ping %d", c.flyingPings)
			}
			
		}
	}
}

func (c *controllerT) updateSitePerVp() {

	// Load sites per vp
	mts := c.router.All()[:1]
	
	c.sitePerVPLock.Lock()

	for _, mt := range mts {
		vpc, err := mt.GetVPs(context.Background(), &dm.VPRequest{
			IsOnlyActive: true,
		})
		if err != nil {
			log.Error(err)
		}
		for vps := range vpc {
			log.Infof("Updating available sites for controller sites per vps... found %d vantage points", len(vps.Vps))
			for _, vp := range(vps.Vps) {
				if strings.Contains(vp.Site, "MLab") {
					tokens := strings.Split(vp.Site, " - ")
					// MLab format is: MLab - LGA0T
					site := tokens[1]
					c.sitePerVP[vp.Ip] = site
				} else {
					// controller.sitePerVP[vp.Ip] = vp.Site
				}
			}	
		}
		mt.Close()
	}

	c.sitePerVPLock.Unlock()

}

func (c* controllerT) monitorSitePerVp() {
	t := time.NewTicker(15 * time.Minute)
	
	for {
		select{
		case <-t.C:
			c.updateSitePerVp()
		}
	}
}

func (c *controllerT) nextSpoofID() uint32 {
	return atomic.AddUint32(&(c.spoofID), 1)
}

// HandleSig handles and signals received from the OS
func HandleSig(sig os.Signal) {
	controllerInstance.handleSig(sig)
}

func (c *controllerT) handleSig(sig os.Signal) {
	log.Infof("Got signal: %v", sig)
	c.stop()
}

func (c *controllerT) startRPC(eChan chan error) {
	addr := fmt.Sprintf("%s:%d", *c.config.Local.Addr,
		*c.config.Local.Port)
	log.Infof("Conecting to: %s", addr)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	log.Infof("Controller started, listening on: %s", addr)
	err := c.server.Serve(l)
	if err != nil {
		eChan <- err
	}
}

func errorAllPing(ctx con.Context, err error, out chan<- *dm.Ping, ps []*dm.PingMeasurement) {
	for _, p := range ps {
		select {
		case out <- &dm.Ping{
			Src:   p.Src,
			Dst:   p.Dst,
			Error: err.Error(),
		}:
		case <-ctx.Done():
		}
	}
}

func errorAllTrace(ctx con.Context, err error, out chan<- *dm.Traceroute, ts []*dm.TracerouteMeasurement) {
	for _, t := range ts {
		select {
		case out <- &dm.Traceroute{
			Src:   t.Src,
			Dst:   t.Dst,
			Error: err.Error(),
		}:
		case <-ctx.Done():
		}
	}
}

var (
	// ErrTimeout is used when the done channel on a context is received from
	ErrTimeout = fmt.Errorf("Request timeout")
)

func (c * controllerT) checkPingCache(ctx con.Context, keys []string, cache ca.Cache) (map[string]*dm.Ping, error) {
	log.Debug("Checking for pings in cache: ", keys)
	out := make(chan map[string]*dm.Ping, 1)
	quit := make(chan struct{})
	eout := make(chan error, 1)
	go func() {
		found := make(map[string]*dm.Ping)
		res, err := cache.GetMulti(keys)
		if err != nil {
			log.Error(err)
			eout <- err
			return
		}
        log.Debug("\n \n GET MULTI OUTPUT: \n \n: ", res)
		for key, item := range res {
            log.Debug("\n \n PRINT ITEM VALUE ***: ", key, item.Value())
            if bytes.Equal(item.Value(), []byte{'x'}) {
                log.Debug("\n \n DUMMY VALUE REACHED \n \n")
                ping := &dm.Ping {
					Id : -1,
				}
                pingsFromCache.Inc()
                found[key] = ping
            } else {
			    ping := &dm.Ping{}
			    err := ping.CUnmarshal(item.Value())
			    if err != nil {
				    log.Error(err)
				    continue
			    }
			    pingsFromCache.Inc()
			    found[key] = ping
            }
		}
		select {
		case <-quit:
			return
		case out <- found:
		}
	}()
	select {
	case <-ctx.Done():
		close(quit)
		return nil, ErrTimeout
	case ret := <-out:
		log.Debug("Got from ping cache: ", ret)
		return ret, nil
	case err := <-eout:
		return nil, err
	}
}

func checkPingDb(ctx con.Context, check []*dm.PingMeasurement, db DataAccess) (map[string]*dm.Ping, error) {
	out := make(chan map[string]*dm.Ping, 1)
	quit := make(chan struct{})
	eout := make(chan error, 1)
	go func() {
		foundMap := make(map[string]*dm.Ping)
		found, err := db.GetPingsMulti(check)
		if err != nil {
			log.Error(err)
			eout <- err
		}
		for _, p := range found {
			log.Debug("Got ping from db: ", p)
			pingsFromDB.Inc()
			// TODO change nil to something, but code should never come here
			foundMap[p.KeyPerSite(nil)] = p
		}
		select {
		case <-quit:
			return
		case out <- foundMap:
		}
	}()
	select {
	case <-ctx.Done():
		close(quit)
		return nil, ErrTimeout
	case ret := <-out:
		return ret, nil
	case err := <-eout:
		return nil, err
	}
}

// Such a shame go does not have generics
func (c * controllerT) flushBatchIfNeeded(batchInsert [] * dm.Ping, batchFlushSize int, ret chan *dm.Ping) [] * dm.Ping {
	if len(batchInsert) % 10000 == 0{
		log.Infof("Batch has %d pings", len(batchInsert))
	}
	if len(batchInsert) == batchFlushSize{
		// Bulk insert this measurement
		log.Debugf("Flushing batch of %d pings into the database", batchFlushSize)
		// Limiting the parallel number or insertions. 
		// c.dbRateLimit <- 0
		
		// batch the inserts so the query is not too big
		queryLimitPingSize := 10000 
		for k := 0; k < len(batchInsert); k+=queryLimitPingSize {
			maxIndex := k + queryLimitPingSize
			if maxIndex > len(batchInsert) {
				maxIndex = len(batchInsert)
			}
			// Put the pings in the queue 
			ids, err := c.db.StorePingBulk(batchInsert[k:maxIndex])
			// <- c.dbRateLimit
			if err != nil {
				log.Error(err)
			}
			for i, ping := range batchInsert[k:maxIndex]{
				// Generate fake IDs now as we bulk insert in the DB
				ping.Id = ids[i]
				// err = c.cache.SetWithExpire(ping.KeyPerSite(c.sitePerVP), ping.CMarshal(), int32(*c.config.Cache.CacheTime))
				// if err != nil {
				// 	log.Error(err)
				// }
				ret <- ping
			}
		}
		
		
		log.Debugf("Inserted batch of %d pings into the database", batchFlushSize)
		return nil
	}
	return batchInsert 
}

func (c *controllerT) doPing(ctx con.Context, pm []*dm.PingMeasurement) <-chan *dm.Ping {

	// Control the number of spoofed flying orders
	// if pm[0].Spoof {
	// 	controller.queue <- 1
	// } 
	start := time.Now()
	// batchFlushSize := 10000

	ret := make(chan *dm.Ping, len(pm))
	label := ""
	saveDB := true
	isAbsoluteTimeoutSpoof := false
	spoofTimeout := 0
	isTimestamp := false
	fromRevtr := false
	useCache :=  true

	if len(pm) > 0 {
		if pm[0].FromRevtr {
			fromRevtr = true
		}
		if pm[0].Label != "" {
			// Super sloppy way to do this, but:
			// I assume that a doPing call with the measurements batcho
			// cannot have a mix of production and ranking measurements.
			// Therefore, if one of them has the flag up,
			// then this doPing was called for a ranking round batch.
			log.Debug(pm[0].Label)
			label = pm[0].Label	
		}
		// Again, akward check to flag TS pings that did not get any answer
		// This assumes that we do not send multiple types of ping per measurement. 
		if pm[0].TimeStamp != "" {
			isTimestamp = true
		}
		saveDB = pm[0].SaveDb
		useCache = pm[0].CheckCache
		isAbsoluteTimeoutSpoof = pm[0].IsAbsoluteSpoofTimeout
		spoofTimeout = int(pm[0].SpoofTimeout)

	} 
	go func() {
		var checkCache = make(map[string]*dm.PingMeasurement)
		var remaining []*dm.PingMeasurement
		// The cache key is (source, destination)
		var cacheKeys []string
		for _, pm := range pm {
			if pm.CheckCache && pm.RR && pm.TimeStamp == "" {
				key := ""
				if pm.Spoof {
					// pm.Src is the spoofer or the source, so we want to retrieve the site of the spoofer or the source
					c.sitePerVPLock.RLock()
					site, ok := c.sitePerVP[pm.Src] 
					c.sitePerVPLock.RUnlock()
					if !ok {
						site = strconv.FormatUint(uint64(pm.Src), 10)
					}
					key = pm.Key(site)
				} else {
					key = pm.Key("")
				}	
				
				// log.Infof("Trying to retrieve from cache %s", key)
				if key != "" {
					checkCache[key] = pm
					cacheKeys = append(cacheKeys, key)
				}
				continue
			}
			remaining = append(remaining, pm)

		}
		var found map[string]*dm.Ping
		if len(cacheKeys) > 0 {
			var err error
			found, err = c.checkPingCache(ctx, cacheKeys, c.cache)
			if err != nil {
				log.Error(err)
			}
		}
		// Figure out what we got vs what we asked for
		for key, val := range checkCache {
			// For each one that we looked for,
			// If it was found, send it back,
			// Otherwise, add it to the remaining list
			if p, ok := found[key]; ok {
                // if ( p.Id == -1) {
				// 	// Was unresponsive so do not probe again. 
				// 	log.Debugf("Unresponsive cached %s", key)

                //     continue
                // } else {
					// log.Infof("Cache hit for %s", key)
				log.Debug("sending: ", p)
				p.FromCache = true
				ret <- p
					// }
			} else {
				remaining = append(remaining, val)
			}
		}
		var checkDb = make(map[string]*dm.PingMeasurement)
		var checkDbArg []*dm.PingMeasurement
		working := remaining
		remaining = nil
		for _, pm := range working {
			if pm.CheckDb && pm.TimeStamp == "" {
				// TODO see if we still want to retrieve things from DB...
				// c.sitePerVPLock.Lock()
				// checkDb[pm.Key(c.sitePerVP)] = pm
				// c.sitePerVPLock.RUnlock()
				checkDbArg = append(checkDbArg, pm)
				continue
			}
			remaining = append(remaining, pm)
		}
		if len(checkDbArg) > 0 {
			dbs, err := checkPingDb(ctx, checkDbArg, c.db)
			if err != nil {
				log.Error(err)
			}
			// Again figure out what we got out of what we asked for
			for key, val := range checkDb {
				if p, ok := dbs[key]; ok {
					//This should be stored in the cache
					go func() {
						var err = c.cache.SetWithExpire(key, p.CMarshal(), int32(*c.config.Cache.CacheTime))
						if err != nil {
							log.Info(err)
						}
					}()
					ret <- p
				} else {
					remaining = append(remaining, val)
				}
			}
		}
		//Remaining are the measurements that need to be run
		var spoofs []*dm.PingMeasurement
		old := remaining
		remaining = nil
        // separates measurements that need to be spoofed with
        // those that don't need to be spoofed
		for _, pm := range old {
			if pm.Spoof {
				spoofs = append(spoofs, pm)
			} else {
				remaining = append(remaining, pm)
			}
		}
		mts := make(map[router.ServiceDef][]*dm.PingMeasurement)
		for _, pm := range remaining {
			var err error
			if pm.GetIsRipeAtlas(){
				sd, err1 := c.router.GetService(router.RIPEAtlas)
				err = err1
				mts[sd] = append(mts[sd], pm)
			} else  {
				sd, err1 := c.router.GetService(router.PlanetLab)
				err = err1
				mts[sd] = append(mts[sd], pm)
			}
			// ip, _ := util.Int32ToIPString(pm.Src)
			if err != nil {
				log.Error(err)
				ret <- &dm.Ping{
					Src:   pm.Src,
					Dst:   pm.Dst,
					Error: err.Error(),
				}
				continue
			}
		}

		// Use a chanel that stores periodic batches of Ping 
		// batchChan := make(chan *dm.Ping, len(pm))
		// flushedLastBatchChan := make(chan bool)
		// allMeasurementsDoneChan := make(chan bool)
		// go func() {
		// 	batchInsert := [] * dm.Ping{} 

		// 	for {
		// 		select {
		// 		case p := <- batchChan:
		// 			if saveDB {
		// 				batchInsert = append(batchInsert, p)
		// 				batchInsert = c.flushBatchIfNeeded(batchInsert, batchFlushSize, ret)
		// 			} else {
		// 				ret <- p
		// 			}
					
					
		// 		case <- allMeasurementsDoneChan:
		// 			// Everything is in the chanel so, just pull it and flush it to DB
		// 			close(batchChan)
		// 			for p := range batchChan {
		// 				if saveDB {
		// 					batchInsert = append(batchInsert, p)
		// 					batchInsert = c.flushBatchIfNeeded(batchInsert, batchFlushSize, ret)
		// 				} else {
		// 					ret <- p
		// 				}
		// 			}
		// 			if len(batchInsert) > 0{
		// 				if saveDB {
		// 					c.flushBatchIfNeeded(batchInsert, len(batchInsert), ret)
		// 				}
		// 				// Nothing to do if no save db
		// 			}
		// 			flushedLastBatchChan <- true
		// 			return 
		// 		case <-ctx.Done():
		// 			return 
		// 		}
		// 	}
			
		// }()

		var wg sync.WaitGroup
		atomic.AddUint32(&c.flyingPings, 1)
		for sd, pms := range mts {
			wg.Add(1)
			go func(s router.ServiceDef, meas []*dm.PingMeasurement) {
				defer wg.Done()
				mt, err := c.router.GetMT(s)
				if err != nil {
					log.Error(err)
					errorAllPing(ctx, err, ret, meas)
					return
				}
				defer mt.Close()
				pc, err := mt.Ping(ctx, &dm.PingArg{
					Pings: meas,
				})

				if err != nil {
					log.Error(err)
					errorAllPing(ctx, err, ret, meas)
					return
				}
				
				for {
					select {
					case <-ctx.Done():
						return
					case pp, ok := <-pc:
						
						if !ok {
							return
						}
						if pp == nil {
							return
						}
						log.Debug("labelling ping")
						pp.Label = label
						if isTimestamp {
							pp.Flags = append(pp.Flags, "tsandaddr")
						}

						if pp.Error != "" {
							log.Errorf("Ping %s, %d, %d", pp.Error, pp.Src, pp.Dst)
						}
						
						// Put in the DB 
						if saveDB {
							if fromRevtr {
								pp.FromRevtr = true
								// now := time.Now()
								revtrMeasurementID := atomic.AddUint64(&c.maxRevtrMeasurementIDPing, 1)
								// execTime := time.Now().Sub(now)
								// log.Infof("Took %s seconds", execTime)
								pp.RevtrMeasurementId = revtrMeasurementID
							}
							
							c.queueInsertPing <- pp
							
						}	
						// No need to pass the map of sites as it is not spoofed
						if useCache{
							cacheKey := pp.KeyPerSite(nil)
							// cacheTime := int32(*c.config.Cache.CacheTime)
							if len(pp.GetResponses()) == 0 && pp.Error == "" {
								log.Debug("\n \n PING GOT NO RESPONSES \n \n")
								notFound := []byte{'x'}
								
								// Set the timeout of unresponsive to 1 min
								err = c.cache.SetWithExpire(cacheKey, notFound, 60)
								if err != nil {
									log.Error(err)
								}
								
								// resX, err := c.cache.Get(cacheKey)
								// if err != nil {
								// 	log.Error(err)
								// }
								// log.Debug("\n \n CACHE VALUE: ", resX)
							} else {
								go func() {
									c.updateCacheRRPing(pp)
								} ()
							}
						}
						
						
						ret <- pp
						
						// select {
						// case <-ctx.Done():
						// 	return
						// }
					}
				}
			}(sd, pms)	
		} 
		receivedResponsesIds := map[uint32]bool{}
		// receivedResponsesIdsDebug := map[uint32] dm.Spoof{}
		sdForSpoof := make(map[router.ServiceDef][]*dm.Spoof)
		sdForSpoofP := make(map[router.ServiceDef][]*dm.PingMeasurement)
		var spoofIds []uint32
		for _, sp := range spoofs {
			// ip, _ := util.Int32ToIPString(sp.Src)
			// At the moment, only PL and MLab nodes can receive spoofed probes
			sd, err := c.router.GetService(router.PlanetLab)
			if err != nil {
				log.Error(err)
				ret <- &dm.Ping{
					Src:   sp.Src,
					Dst:   sp.Dst,
					Error: err.Error(),
				}
				continue
			}
			// ips, _ := util.Int32ToIPString(sp.SpooferAddr)
			sds, err := c.router.GetService(router.PlanetLab)
			if err != nil {
				log.Error(err)
				ret <- &dm.Ping{
					Src:   sp.Src,
					Dst:   sp.Dst,
					Error: err.Error(),
				}
				continue
			}
			
			sdForSpoofP[sd] = append(sdForSpoofP[sd], sp)
			id := c.nextSpoofID()
			sp.Payload = fmt.Sprintf("%08x", id)
			spoofIds = append(spoofIds, id)
			receivedResponsesIds[id] = false
			log.Debug("Creating dm.Spoof for: ", *sp)
			sa, err := util.IPStringToInt32(sp.SAddr)
			if err != nil {
				log.Error(err)
				
				continue
			}
			spoof := dm.Spoof{
				Ip:  sp.Src,
				Id:  id,
				Sip: sa,
				Dst: sp.Dst,
			}
			sdForSpoof[sds] = append(sdForSpoof[sds], &spoof)
			// receivedResponsesIdsDebug[id] = spoof
		}
		rChan := make(chan *dm.Probe, len(spoofIds))
		kill := make(chan struct{})
		if len(spoofIds) != 0 {
			c.sm.Add(rChan, kill, spoofIds)
		} else {
			// This is ugly but prevent waiting for no reason
			close(rChan)
		}
		for sd, spoofs := range sdForSpoof {
			wg.Add(1)
			go func(s router.ServiceDef, sps []*dm.Spoof) {
				defer wg.Done()
				mt, err := c.router.GetMT(s)
				if err != nil {
					log.Error(err)
					return
				}
				defer mt.Close()
				// Prepare to receive the spoofed packet
				resp, err := mt.ReceiveSpoof(ctx, &dm.RecSpoof{
					Spoofs: sps,
				})
				if err != nil {
					log.Error(err)
					return
				}
				for _ = range resp {
				}
			}(sd, spoofs)
		}
		for sd, spoofs := range sdForSpoofP {
			wg.Add(1)
			go func(s router.ServiceDef, sps []*dm.PingMeasurement) {
				defer wg.Done()
				mt, err := c.router.GetMT(s)
				if err != nil {
					log.Error(err)
					return
				}
				defer mt.Close()
				resp, err := mt.Ping(ctx, &dm.PingArg{
					Pings: sps,
				})
				if err != nil {
					log.Error(err)
					return
				}
				for _ = range resp {

				}
			}(sd, spoofs)
		}
		wg.Add(1)

		go func() {
			// This boolean ensures that the system absords the load at the beginning
			hasReceivedAtLeastOneResponse := false
			isLastTick := false
			ticker := time.NewTicker(5 * time.Second)
			// Absolute timeout if no response, it's 60 seconds.
			maxTicksWithoutResponse := spoofTimeout / 5
			ticksWithoutResponse := 0
			
			defer ticker.Stop()
			defer wg.Done()
			for {
				select {
				case <-ticker.C:
					if isLastTick || ticksWithoutResponse == maxTicksWithoutResponse {
						if isLastTick {
							for spoofID, gotResponse := range(receivedResponsesIds) {
								if !gotResponse {
									log.Debugf("Timeout because did not receive response for %d", spoofID)
								}
							}
						} else {
							log.Debugf("Absolute timeout because no responses")
						}
						close(kill)
						return
					}
					if hasReceivedAtLeastOneResponse {
						// Now that we have at least one reply, next ticker we stop 
						// (because we expect packets to be queued together)
						
						if !isAbsoluteTimeoutSpoof {
							ticksWithoutResponse = 0
							isLastTick = true
						} else {
							ticksWithoutResponse ++
						}
						
					} else {
						ticksWithoutResponse ++
					}
					
				case <-ctx.Done():
					close(kill)
					return
				case probe, ok := <-rChan:
					if !ok {
						close(kill)
						return
					}
					if probe == nil {
						close(kill)
						return
					}

					px, x := toPing(probe, label)
					receivedResponsesIds[probe.ProbeId] = true
					// log.Infof("%d", receivedResponsesIdsDebug[probe.ProbeId].Sip)
					// we received a response, so starts the timer 
					hasReceivedAtLeastOneResponse = true
                    if x == 1 { // 1 indicates no TS, so either RR or standard ping 
						if saveDB {
							if fromRevtr {
								px.FromRevtr = true
								revtrMeasurementID := atomic.AddUint64(&c.maxRevtrMeasurementIDPing, 1)
								px.RevtrMeasurementId = revtrMeasurementID
							}
							c.queueInsertPing <- px
						}
						// log.Debug("Caching spoofed result with key: ", px.Key())
                        // notFound := []byte{'x'}
                        // log.Debug("\n \n CREATING DUMMY VALUE \n \n", notFound)
						if useCache {
							go func() {
								c.updateCacheRRPing(px)
							} ()
						}
						
						
                        // resX, err := c.cache.Get(px.Key())
                        // if err != nil {
                        //     log.Error(err)
                        // }
						// log.Debug("\n \n CACHE VALUE: :", resX)
				
						ret <- px
						// batchChan <- px
                        // select {
                        // case <-ctx.Done():
                        //     close(kill)
                        //     return
                        // case ret <- px:
                        // }
                    } else {
						// if !saveDB {
						// 	err := c.cache.SetWithExpire(px.Key(c.sitePerVP), px.CMarshal(), int32(*c.config.Cache.CacheTime))
						// 	if err != nil {
						// 		log.Error(err)
						// 	}
						// }
						
						if saveDB {
							if fromRevtr {
								px.FromRevtr = true
								revtrMeasurementID := atomic.AddUint64(&c.maxRevtrMeasurementIDPing, 1)
								px.RevtrMeasurementId = revtrMeasurementID
							}
							c.queueInsertPing <- px
						}
						
						ret <- px

					    // pid, err := c.db.StorePing(px)
					    // if err != nil {
						//     log.Error(err)
					    // }
					    // px.Id = pid
					    // select {
					    // case <-ctx.Done():
						//     close(kill)
						//     return
					    // case ret <- px:
						// }
                    }
				}
			}
		}()
		wg.Wait()
		atomic.AddUint32(&c.flyingPings, ^uint32(0))
		// allMeasurementsDoneChan <- true
		// <- flushedLastBatchChan
		isSpoof := pm[0].Spoof
		s := "RR"
		if isSpoof {
			s += " spoof"
		}
		elapsed := time.Since(start)
		log.Debugf("Took %f seconds to run %s pings", elapsed.Seconds(), s)
		close(ret)
		// Release 1 measurement token
		// if pm[0].Spoof{
		// 	<-controller.queue 
		// }
	}()
	return ret
}

func toPing(probe *dm.Probe, label string) (*dm.Ping, int) {
	var ping dm.Ping
	// Ping src and dst are reversed from the probe
	// since the probe is a response to a spoofed ping
	// Ping srcs and dsts are from the measurement
	// not the response
	ping.Src = probe.Dst
	ping.Dst = probe.Src
	ping.SpoofedFrom = probe.SpooferIp
	ping.Flags = append(ping.Flags, "spoof")
	ping.PingSent = 1
	ping.Label = label
	var pr dm.PingResponse
	tx := &dm.Time{}
	now := time.Now().Unix()
	tx.Sec = now
	pr.Tx = tx
	rx := &dm.Time{}
	rx.Sec = now
	pr.Rx = rx
	pr.From = probe.Src
	rrs := probe.GetRR()
    x := 0
	ts := probe.GetTs()
	ping.Responses = []*dm.PingResponse{&pr}
	if rrs != nil {
        //if len(rrs.Hops) == 0 {
        //    log.Debug("\n \n NO RECORD ROUTE HOPS FOUND \n \n")
        //    ping.Flags = append(ping.Flags, "v4rr")
        //    pr.RR = []uint32{}
        //    ping.Responses = []*dm.PingResponse{&pr}
        //    x = 1
		ping.Flags = append(ping.Flags, "v4rr")
		pr.RR = rrs.Hops
		x = 1
	}
	//  else if ts == nil { 
	// 	//TODO Do better than this. Currently, only  spoofed RR and TS, so if it's not one, it's the other one. 
    //     log.Debug("\n \n RRS NIL \n \n")
    //     ping.Flags = append(ping.Flags, "v4rr")
    //     pr.RR = []uint32{}
    //     ping.Responses = []*dm.PingResponse{&pr}
    //     x = 1
    // }
	
	if ts != nil {
		switch ts.Type {
		case dm.TSType_TSOnly:
			ping.Flags = append(ping.Flags, "tsonly")
			stamps := ts.GetStamps()
			var ts []uint32
			for _, stamp := range stamps {
				ts = append(ts, stamp.Time)
			}
			pr.Tsonly = ts
		default:
			stamps := ts.GetStamps()
			var ts []*dm.TsAndAddr
			if stamps != nil {

				ping.Flags = append(ping.Flags, "tsandaddr")
				for _, stamp := range stamps {
					ts = append(ts, &dm.TsAndAddr{
						Ip: stamp.Ip,
						Ts: stamp.Time,
					})
				}
				pr.Tsandaddr = ts
			}
		}
	}
	return &ping, x
}


func (c *controllerT) updateCacheRRPing(ping *dm.Ping) {

	
	site := "0"
	if ping.SpoofedFrom != 0 {
		var ok bool
		c.sitePerVPLock.RLock()
		site, ok = c.sitePerVP[ping.SpoofedFrom]
		c.sitePerVPLock.RUnlock()
		if !ok {
			site = strconv.FormatUint(uint64(ping.SpoofedFrom), 10)
		}
	}

	distancePerDst := map[uint32]int{}
	cacheKeys := []string{}
	for _, pingResponse := range(ping.Responses) {
		isAfterDestination := false
		for i, rrHop := range(pingResponse.RR) {
			if rrHop == ping.Dst {
				isAfterDestination = true
			}
			if !isAfterDestination {
				continue
			} 	
			if rrHop != 0 && rrHop != ping.Src {
				// Create a ping measurement for RR 
				cacheKey := fmt.Sprintf("XRRP_%s_%d_%d", site, ping.Src, rrHop)
				cacheKeys = append(cacheKeys, cacheKey)
				distancePerDst[rrHop] = i
			}
					
		}
	}

	hits, err := c.cache.GetMulti(cacheKeys)
	if err != nil {
		log.Error(err)
	}
	
	cachedDistancePerDst := map[uint32]int {}
	for _, item := range hits {
		if bytes.Equal(item.Value(), []byte{'x'}) {
			// This was a dummy value so update the cache
			continue
		} else {
			ping := &dm.Ping{}
			err := ping.CUnmarshal(item.Value())
			if err != nil {
				log.Error(err)
				continue
			}
			for _, resp := range(ping.Responses) {
				isAfterDestination := false 
				for cacheDistance, rrHop := range(resp.RR) {
					if rrHop == ping.Dst {
						isAfterDestination = true
					}
					if isAfterDestination {
						cachedDistancePerDst[rrHop] = cacheDistance
					}
					
				}
			}
		}
	}

	cacheTime := int32(*c.config.Cache.CacheTime)
	if _, ok := distancePerDst[ping.Dst]; !ok {
		// if it did not reach the dst, set the src, dst ping in the cache 
		cacheKey := fmt.Sprintf("XRRP_%s_%d_%d", site, ping.Src, ping.Dst)
		err = c.cache.SetWithExpire(cacheKey, ping.CMarshal(), cacheTime)
		if err != nil {
			log.Error(err)
		}
	} else {
		// otherwise set all the reverse hops in the cache
		for rrHop, measuredDistance := range(distancePerDst) {
			cachedDistance, ok := cachedDistancePerDst[rrHop]
			if !ok || measuredDistance <= cachedDistance {
				cacheKey := fmt.Sprintf("XRRP_%s_%d_%d", site, ping.Src, rrHop)
				err := c.cache.SetWithExpire(cacheKey, ping.CMarshal(), cacheTime)
				log.Debugf("Putting %s in cache", cacheKey)
				if err != nil {
					log.Error(err)
				}
			}	
		}
	}
	
	
	
}

func (c *controllerT) doRecSpoof(ctx con.Context, pr *dm.Probe) {
	c.sm.Notify(pr)
}




// Such a shame go does not have generics
func (c * controllerT) flushBatchIfNeededTraceroutes(batchInsert [] * dm.Traceroute, 
	batchFlushSize int, ret chan *dm.Traceroute) [] * dm.Traceroute {
	if len(batchInsert) % 1000 == 0{
		log.Infof("Batch has %d traceroutes", len(batchInsert))
	}
	if len(batchInsert) == batchFlushSize{
		// Bulk insert this measurement
		log.Infof("Flushing batch of %d traceroutes into the database", batchFlushSize)
		// Limiting the parallel number or insertions. 
		// c.dbRateLimit <- 0
		
		// batch the inserts so the query is not too big
		queryLimitPingSize := 10000 
		for k := 0; k < len(batchInsert); k+=queryLimitPingSize {
			maxIndex := k + queryLimitPingSize
			if maxIndex > len(batchInsert) {
				maxIndex = len(batchInsert)
			}
			ids, err := c.db.StoreTracerouteBulk(batchInsert[k:maxIndex])
			// <- c.dbRateLimit
			if err != nil {
				log.Error(err)
			}
			
			for i, traceroute := range batchInsert[k:maxIndex]{
				traceroute.Id = ids[i]
				err = c.cache.SetWithExpire(traceroute.Key(), traceroute.CMarshal(), int32(*c.config.Cache.CacheTime))
				if err != nil {
					log.Error(err)
				}
				ret <- traceroute
			}
		}
		
		
		log.Debugf("Inserted batch of %d pings into the database", batchFlushSize)
		return nil
	}
	return batchInsert 
}

func checkTraceCache(ctx con.Context, keys []string, ca ca.Cache) (map[string]*dm.Traceroute, error) {
	log.Debug("Checking for traceroute in cache: ", keys)
	out := make(chan map[string]*dm.Traceroute, 1)
	eout := make(chan error, 1)
	go func() {
		found := make(map[string]*dm.Traceroute)
		res, err := ca.GetMulti(keys)
		if err != nil {
			log.Error(err)
			select {
			case eout <- err:
			case <-ctx.Done():
				return
			}
		}
		for key, item := range res {
			trace := &dm.Traceroute{}
			err := trace.CUnmarshal(item.Value())
			if err != nil {
				log.Error(err)
				continue
			}
			tracesFromCache.Inc()
			found[key] = trace
		}
		select {
		case <-ctx.Done():
		case out <- found:
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ErrTimeout
	case ret := <-out:
		log.Debug("Got from traceroute cache: ", ret)
		return ret, nil
	case err := <-eout:
		return nil, err
	}
}

func checkTraceDb(ctx con.Context, check []*dm.TracerouteMeasurement, db DataAccess) (map[string]*dm.Traceroute, error) {
	out := make(chan map[string]*dm.Traceroute, 1)
	quit := make(chan struct{})
	eout := make(chan error, 1)
	go func() {
		foundMap := make(map[string]*dm.Traceroute)
		found, err := db.GetTraceMulti(check)
		if err != nil {
			log.Error(err)
			select {
			case eout <- err:
			case <-ctx.Done():
			}
			return
		}
		for _, p := range found {
			tracesFromDB.Inc()
			foundMap[p.Key()] = p
		}
		select {
		case <-quit:
		case out <- foundMap:
		case <-ctx.Done():
		}
	}()
	select {
	case <-ctx.Done():
		close(quit)
		return nil, ErrTimeout
	case ret := <-out:
		return ret, nil
	case err := <-eout:
		return nil, err
	}
}

func (c *controllerT) doTraceroute(ctx con.Context, tms []*dm.TracerouteMeasurement) <-chan *dm.Traceroute {
	start := time.Now()
	ret := make(chan *dm.Traceroute)
	log.Debug("Running traceroutes: ", tms)
	label := ""
	saveDB := true
	fromRevtr := false
	if len(tms) > 0 {
		fromRevtr = tms[0].FromRevtr
		if tms[0].Label != "" {
			// Super sloppy way to do this, but:
			// I assume that a doPing call with the measurements batcho
			// cannot have a mix of production and ranking measurements.
			// Therefore, if one of them has the flag up,
			// then this doPing was called for a ranking round batch.
			label = tms[0].Label	
		}
		saveDB = tms[0].SaveDb
	} 
	

	// Use a chanel that stores periodic batches of Ping 
	// batchChan := make(chan *dm.Traceroute, len(tms))
	// flushedLastBatchChan := make(chan bool)
	// allMeasurementsDoneChan := make(chan bool)
	// batchFlushSize := 1000
	// go func() {
	// 	batchInsert := [] * dm.Traceroute{} 

	// 	for {
	// 		select {
	// 		case p := <- batchChan:
	// 			if saveDB {
	// 				batchInsert = append(batchInsert, p)
	// 				batchInsert = c.flushBatchIfNeededTraceroutes(batchInsert, batchFlushSize, ret)
	// 			} else {
	// 				ret <- p
	// 			}
				
				
	// 		case <- allMeasurementsDoneChan:
	// 			// Everything is in the chanel so, just pull it and flush it to DB
	// 			close(batchChan)
	// 			for p := range batchChan {
	// 				if saveDB {
	// 					batchInsert = append(batchInsert, p)
	// 					batchInsert = c.flushBatchIfNeededTraceroutes(batchInsert, batchFlushSize, ret)
	// 				} else {
	// 					ret <- p
	// 				}
	// 			}
	// 			if len(batchInsert) > 0{
	// 				if saveDB {
	// 					c.flushBatchIfNeededTraceroutes(batchInsert, len(batchInsert), ret)
	// 				}
	// 				// Nothing to do if no save db
	// 			}
	// 			flushedLastBatchChan <- true
	// 			return 
	// 		case <-ctx.Done():
	// 			return 
	// 		}
	// 	}
		
	// }()

	go func() {
		var checkCache = make(map[string]*dm.TracerouteMeasurement)
		var remaining []*dm.TracerouteMeasurement
		var cacheKeys []string
		for _, tm := range tms {
			if tm.CheckCache {
				key := tm.Key()
				checkCache[key] = tm
				cacheKeys = append(cacheKeys, key)
				continue
			}
			remaining = append(remaining, tm)
		}
		var found map[string]*dm.Traceroute
		if len(cacheKeys) > 0 {
			var err error
			found, err = checkTraceCache(ctx, cacheKeys, c.cache)
			if err != nil {
				log.Error(err)
			}
		}
		// Figure out what we got vs what we asked for
		for key, val := range checkCache {
			// For each one that we looked for,
			// If it was found, send it back,
			// Otherwise, add it to the remaining list
			if p, ok := found[key]; ok {
				p.FromCache = true
				ret <- p
			} else {
				remaining = append(remaining, val)
			}
		}
		var checkDb = make(map[string]*dm.TracerouteMeasurement)
		var checkDbArg []*dm.TracerouteMeasurement
		working := remaining
		remaining = nil
		for _, pm := range working {
			if pm.CheckDb {
				checkDb[pm.Key()] = pm
				checkDbArg = append(checkDbArg, pm)
				continue
			}
			remaining = append(remaining, pm)
		}
		if len(checkDbArg) > 0 {
			dbs, err := checkTraceDb(ctx, checkDbArg, c.db)
			if err != nil {
				log.Error(err)
			}
			// Again figure out what we got out of what we asked for
			for key, val := range checkDb {
				if p, ok := dbs[key]; ok {
					//This should be stored in the cache
					go func() {
						if p.StopReason != "COMPLETED" {
							return
						}
						var err = c.cache.SetWithExpire(key, p.CMarshal(), int32(*c.config.Cache.CacheTime))
						if err != nil {
							log.Info(err)
						}
					}()
					ret <- p
				} else {
					remaining = append(remaining, val)
				}
			}
		}
		mts := make(map[router.ServiceDef][]*dm.TracerouteMeasurement)
		for _, tm := range remaining {
			// ip, _ := util.Int32ToIPString(tm.Src)

			if tm.GetIsRipeAtlas(){
				sd, _ := c.router.GetService(router.RIPEAtlas)
				mts[sd] = append(mts[sd], tm)
			} else  {
				sd, err := c.router.GetService(router.PlanetLab)
				mts[sd] = append(mts[sd], tm)
				if err != nil {
					log.Error(err)
					ret <- &dm.Traceroute{
						Src:   tm.Src,
						Dst:   tm.Dst,
						Error: err.Error(),
					}
					continue
				}
			}			
		}
		var wg sync.WaitGroup
		atomic.AddUint32(&c.flyingTraceroutes, 1)
		for sd, tms := range mts {
			wg.Add(1)
			go func(s router.ServiceDef, meas []*dm.TracerouteMeasurement) {
				defer wg.Done()
				mt, err := c.router.GetMT(s)
				if err != nil {
					log.Error(err)
					errorAllTrace(ctx, err, ret, meas)
					return
				}
				defer mt.Close()
				
				pc, err := mt.Traceroute(ctx, &dm.TracerouteArg{
					Traceroutes: meas,
				})
				if err != nil {
					log.Error(err)
					errorAllTrace(ctx, err, ret, meas)
					return
				}
				for {
					select {
					case <-ctx.Done():
						return
					case pp, ok := <-pc:
						
						if !ok {
							// log.Errorf(pp.Error)
							return
						}

						if pp.Error != "" {
							log.Errorf("Traceroute %s, %d, %d", pp.Error, pp.Src, pp.Dst)
						}

						log.Debug("Got TR ", pp)
						// Akward way to check if this traceroute comes from the revtr system or from the
						// atlas...
						// id := int64(0)
						pp.Label = label
						if saveDB {
							if fromRevtr {
								pp.FromRevtr = true
								revtrMeasurementID := atomic.AddUint64(&c.maxRevtrMeasurementIDTraceroute, 1)
								pp.RevtrMeasurementId = revtrMeasurementID
							}
							c.queueInsertTraceroute <- pp
						}
						// Put in the cache
						// Cache in seconds
						err = c.cache.SetWithExpire(pp.Key(), pp.CMarshal(), int32(*c.config.Cache.CacheTime))
						if err != nil {
							log.Error(err)
						}

						

						ret <- pp
						
						// if saveDB {
						// 	id, err = c.db.StoreTraceroute(pp)
						// }
						
						// if err != nil {
						// 	log.Error(err)
						// }
						// pp.Id = id
						// if pp.Error == "" {
						// 	// Cache in seconds
						// 	err = c.cache.SetWithExpire(pp.Key(), pp.CMarshal(), int32(*c.config.Cache.CacheTime))
						// 	if err != nil {
						// 		log.Error(err)
						// 	}
						// }
						// ret <- pp
						// select {
						// case ret <- pp:
						// 	continue
						// case <-ctx.Done():
						// 	return
						// }
					}
				}
			}(sd, tms)
		}
		wg.Wait()
		atomic.AddUint32(&c.flyingTraceroutes, ^uint32(0))
		// allMeasurementsDoneChan <- true
		// <- flushedLastBatchChan
		elapsed := time.Since(start)
		log.Debugf("Took %f seconds to run %d traceroutes", elapsed.Seconds(), len(tms))
		close(ret)
	}()

	return ret
}

func (c *controllerT) fetchVPs(ctx con.Context, gvp *dm.VPRequest) (*dm.VPReturn, error) {
	mts := c.router.All()[:1]
	var ret dm.VPReturn
	for _, mt := range mts {
		vpc, err := mt.GetVPs(ctx, gvp)
		if err != nil {
			return nil, err
		}
		for vp := range vpc {
			ret.Vps = append(ret.Vps, vp.Vps...)
		}
		mt.Close()
	}
	return &ret, nil
}

func (c *controllerT) doGetVPs(ctx con.Context, gvp *dm.VPRequest) (*dm.VPReturn, error) {
	return c.fetchVPs(ctx, gvp)
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

func (c *controllerT) stop() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *controllerT) run(ec chan error, con Config, db DataAccess, cache ca.Cache, r router.Router) {
	controllerInstance.config = con
	controllerInstance.db = db
	controllerInstance.cache = cache
	controllerInstance.router = r
	controllerInstance.maxParallelInsertQueries = 500
	controllerInstance.queueInsertTokens = make(chan int , controllerInstance.maxParallelInsertQueries)
	controllerInstance.queueInsertPing = make(chan *dm.Ping, 10000000)
	controllerInstance.queueInsertTraceroute = make(chan *dm.Traceroute, 10000000)
	// controller.queue = make(chan int, 200)
	if db == nil {
		log.Errorf("Nil db in Controller Start")
		c.stop()
		ec <- errors.New("Controller Start, nil DB")
		return
	}
	if cache == nil {
		log.Errorf("Nil cache in Controller start")
		c.stop()
		ec <- errors.New("Controller Start, nil Cache")
		return
	}
	if r == nil {
		log.Errorf("Nil router in Controller start")
		c.stop()
		ec <- errors.New("Controller Start, nil router")
		return
	}

	maxRevtrMeasurementIDPing, err := db.GetMaxRevtrMeasurementID("pings")
	if err != nil {
		panic(err)
	} 
	controllerInstance.maxRevtrMeasurementIDPing = maxRevtrMeasurementIDPing

	maxRevtrMeasurementIDTraceroute, err := db.GetMaxRevtrMeasurementID("traceroutes")
	if err != nil {
		panic(err)
	} 
	controllerInstance.maxRevtrMeasurementIDTraceroute = maxRevtrMeasurementIDTraceroute

	// Sanity check to test connections with measurement tools (plcontroller)
	sd, _  := r.GetService(router.PlanetLab)
	_, err  = r.GetMT(sd)
	if err != nil {
		c.stop()
		ec <- err
		return 
	}
	

	opts := []grpc.ServerOption{}
	if !environment.IsDebugController {

		log.Infof("Starting with creds: %s, %s", *con.Local.CertFile, *con.Local.KeyFile)
		certs, err := credentials.NewServerTLSFromFile(*con.Local.CertFile, *con.Local.KeyFile)
		if err != nil {
			log.Error(err)
			c.stop()
			ec <- err
			return
		}
		opts = append(opts, grpc.Creds(certs))
	}
	opts = append(opts, grpc.MaxRecvMsgSize(10010241024))
	opts = append(opts, grpc.MaxSendMsgSize(10010241024))
	controllerInstance.server = grpc.NewServer(opts...)
	controllerapi.RegisterControllerServer(controllerInstance.server, c)
	controllerInstance.sm = &spoofMap{
		sm: make(map[uint32] *channel),
		tm: make(map[uint32] int64),
	}
	controllerInstance.sitePerVP = make(map[uint32]string)
	controllerInstance.sitePerVPLock = sync.RWMutex{}
	go controllerInstance.monitorTraceroutes()
	go controllerInstance.monitorPings()
	go controllerInstance.insertIntoDB()
	
	go controllerInstance.monitorDB()
	go controllerInstance.monitorCache()
	go controllerInstance.startRPC(ec)
	
	controllerInstance.updateSitePerVp()
	go controllerInstance.monitorSitePerVp()

}

// Start starts a central controller with the given configuration
func Start(c Config, db DataAccess, cache ca.Cache, r router.Router) chan error {

	apiPort := 8080
	if environment.IsDebugController {
		apiPort = environment.ControllerAPIPortDebug
	}

	log.Info("Starting controller")
	http.Handle("/metrics", prometheus.Handler())
	mux := http.NewServeMux()
	mux.HandleFunc(v1Prefix+"vps", controllerInstance.VPSHandler)
	mux.HandleFunc(v1Prefix+"rr", controllerInstance.RecordRouteHandler)
	mux.HandleFunc(v1Prefix+"ts", controllerInstance.TimeStampHandler)
	mux.HandleFunc(v1Prefix+"pings", controllerInstance.GetPingsHandler)
	mux.HandleFunc(v1Prefix+"pings/batch/responses", controllerInstance.GetPingsBatchNResponsesHandler)
	mux.HandleFunc(v1Prefix+"ping", controllerInstance.PingHandler)
	mux.HandleFunc(v1Prefix+"traceroute", controllerInstance.TracerouteHandler)
	mux.HandleFunc(v1Prefix+"traceroutes", controllerInstance.GetTracesHandler)
	go func() {
		log.Error(http.ListenAndServe(":"+strconv.Itoa(apiPort), mux))
	}()
	if !environment.IsDebugController {
		go startHTTP(*c.Local.PProfAddr)
	}
	errChan := make(chan error, 2)
	go controllerInstance.run(errChan, c, db, cache, r)
	return errChan
}
