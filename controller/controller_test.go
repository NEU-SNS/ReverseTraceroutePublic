// /*
//  Copyright (c) 2015, Northeastern University
//  All rights reserved.

//  Redistribution and use in source and binary forms, with or without
//  modification, are permitted provided that the following conditions are met:
//      * Redistributions of source code must retain the above copyright
//        notice, this list of conditions and the following disclaimer.
//      * Redistributions in binary form must reproduce the above copyright
//        notice, this list of conditions and the following disclaimer in the
//        documentation and/or other materials provided with the distribution.
//      * Neither the name of the Northeastern University nor the
//        names of its contributors may be used to endorse or promote products
//        derived from this software without specific prior written permission.

//  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
//  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
//  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
//  DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
//  DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
//  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
//  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
//  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
//  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
//  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// */

package controller

import (
	"context"
	"database/sql"
	"flag"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"

	// 	"net"
	// 	"net/http"
	// 	_ "net/http/pprof"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// 	"google.golang.org/grpc/grpclog"

	// 	"golang.org/x/net/trace"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/ipoptions"
	iputil "github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/test"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"

	// 	"github.com/NEU-SNS/ReverseTraceroute/environment"
	controllerapi "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"

	// For spoofing test
	"github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	// The same default as tcpdump.
	defaultSnapLen = 262144
	// which VP is running the test
	// vpTest = "walter"
)

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local"

var pingTable = "pings"
var pingResponseTable = "ping_responses"
var pingStatsTable = "ping_stats"
var pingRRTable = "record_routes"
var pingTSTable = "timestamps"
var pingTSAddrTable = "timestamp_addrs"
var pingICMPTSTable = "icmp_timestamps"
var tracerouteTable = "traceroutes"
var tracerouteHopsTable = "traceroute_hops" 

var tables = []*string{&pingTable, &pingResponseTable, &pingStatsTable, &pingRRTable, 
	&pingTSTable, &pingTSAddrTable, &pingICMPTSTable,
	 &tracerouteTable, &tracerouteHopsTable}

func monitorDB(da *da.DataAccess) {
	t := time.NewTicker(1 * time.Second)
	
	for {
		select{
		case <-t.C:
			dbReaderStats, dbWriterStats := da.Monitor()
			log.Infof("Number of in use / open /max  connections reader: %d %d %d", 
			dbReaderStats.InUse, dbReaderStats.OpenConnections, dbReaderStats.MaxOpenConnections)
			log.Infof("Number of in use / open /max  connections writer: %d %d %d", 
			dbWriterStats.InUse, dbWriterStats.OpenConnections, dbWriterStats.MaxOpenConnections)
		}
	}
}

func getConfig() Config {
	var conf = NewConfig()
	config.AddConfigPath("../cmd/controller/controller.config")
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Errorf("Failed to parse config: %v", err)
		exit(1)
	}

	util.CloseStdFiles(*conf.Local.CloseStdDesc)

	if conf.Db.Environment == "debug" {
		for _, table :=range(tables) {
			if !strings.Contains(*table, "debug") {
				*table += "_debug"
			}
		}
	}

	return conf
}

func GetMySQLDB() (db *sql.DB) {
	
	conf := getConfig()
	db, err := sql.Open("mysql", fmt.Sprintf(conFmt, conf.Db.WriteConfigs[0].User, conf.Db.WriteConfigs[0].Password,
		 conf.Db.WriteConfigs[0].Host, conf.Db.WriteConfigs[0].Port, "ccontroller"))
	if err != nil {
		panic(err)
	}

	return db
}

func createDB() *da.DataAccess{
	conf := getConfig()
	db, err := da.New(conf.Db)
	if err != nil {
		log.Errorf("Failed to create db: %v", err)
		exit(1)
	}
	return db 
}

func deletePingByLabel(label string) (int64, error) {
	db := GetMySQLDB()
	defer db.Close()
	result, err := db.Exec(fmt.Sprintf("DELETE p, ps, pr, ts, rr, tsa " + 
	"from %s as p " +
	"LEFT JOIN %s ps on ps.ping_id=p.id " +
	"LEFT JOIN %s as pr ON pr.ping_id=p.id " +
	"LEFT JOIN %s rr ON rr.response_id = pr.id  " +
	"LEFT JOIN %s ts on ts.response_id=pr.id " +
	"LEFT JOIN %s tsa ON tsa.response_id=pr.id  " +
	"WHERE label=?", pingTable, pingStatsTable, pingResponseTable, pingRRTable, pingTSTable, pingTSAddrTable), label)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}

}

func selectPingByLabel(label string) (int64){
	db := GetMySQLDB()
	defer db.Close()
	rows, err := db.Query(fmt.Sprintf("SELECT count(*) " + 
	"from %s " +
	"WHERE label=?", pingTable), label)
	if err != nil {
		panic(err)
	}
	var count int64
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			panic(err)
		}
	} 
	return count
}


func deleteTracerouteByLabel(label string) (int64, error) {
	db := GetMySQLDB()
	defer db.Close()
	result, err := db.Exec(fmt.Sprintf("DELETE t, th " + 
	"from %s t " +
	"LEFT JOIN %s th on th.traceroute_id=t.id " +
	"WHERE label=?", tracerouteTable, tracerouteHopsTable), label)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}
}

func selectTracerouteByLabel(label string) (int64){
	db := GetMySQLDB()
	defer db.Close()
	rows, err := db.Query(fmt.Sprintf("SELECT count(*) " + 
	"from %s " +
	"WHERE label=?", tracerouteTable), label)
	if err != nil {
		panic(err)
	}
	var count int64
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			panic(err)
		}
	} 
	return count
}



func createPing(label string, recordRouteResponses [] uint32, revtrMeasurementIDPing uint64) dm.Ping {
	pingResponses := [] *dm.PingResponse {}
	if recordRouteResponses == nil {
		recordRouteResponses = [] uint32 {}
		recordRouteResponses = append(recordRouteResponses, 1081933877, 1081937169, 1047757060, 1047756911,
			1534840602, 1449915076, 1534840631, 1534840669, 1534840546)
	}
	
	pingResponse := dm.PingResponse{
		From:3169961473,
		Seq:0,
		ReplySize:124,
		ReplyTtl: 57,
		ReplyProto: "icmp",
		Tx: &dm.Time{
			Sec:1606429656,
			Usec: 392600,
		},
		Rx: &dm.Time{
			Sec: 1606429656,
			Usec: 583351,
		},
		Rtt:190751,
		ProbeIpid: 40943,
		ReplyIpid: 48069,
		IcmpType: 0,
		IcmpCode: 0,
		RR: recordRouteResponses,
		Tsonly: [] uint32{1,1,1,1},
		Tsandaddr:[] *dm.TsAndAddr{&dm.TsAndAddr{
			Ip: 0,
			Ts: 0,
		}}, 
	}
	pingResponses = append(pingResponses, &pingResponse)

	pingStats := dm.PingStats{
		Replies: 1, 
		Loss: 0, 
		Min: 190.751,
		Max: 190.751,
		Avg: 190.751,
		Stddev: 0,
	}
	ping := dm.Ping{
	Type          : "ping"    ,   
		Method        : "icmp-echo",
		Src           : 2159114201 ,
		Dst           : 3169961473 ,
		Start         : &dm.Time{
			Sec: 1606429656,
			Usec: 392458,
		},       
		PingSent      : 1       ,
		ProbeSize     : 124     ,
		UserId        : 228734  ,
		Ttl           : 64      ,
		Wait          : 1       ,
		Timeout       : 0       ,
		Flags         : [] string {"v4rr"} ,
		Responses     : pingResponses,      
		Statistics    : &pingStats,       
		Error         : "",
		Version       : "",
		SpoofedFrom   : 0 ,
		Id            : 0 ,
		Label 				: label, 
		RevtrMeasurementId: revtrMeasurementIDPing,
	}
	return ping
}

func TestInsert(t *testing.T) {
	label := "controller_test_bulk_no_regression"
	deletePingByLabel(label)
	deletePingByLabel(label)
	db := createDB()
	// Create fake ping measurement 
	maxRevtrMeasurementIDPing, err := db.GetMaxRevtrMeasurementID("pings")
	if err != nil {
		panic(err)
	}
	nInsert := 1000
	pings := []*dm.Ping{}
	for i := 0; i < nInsert; i++{
		ping := createPing(label, nil, maxRevtrMeasurementIDPing + uint64(i + 1))
		pings = append(pings, &ping)
	}
	start := time.Now()
	for i, ping := range(pings) {
		if i % 100 == 0{
			log.Infof("Inserted %d pings in the database\n", i)		
		}
		_, err := db.StorePing(ping)
		if err != nil {
			panic(err)
		}
	}
	end := time.Now()
	elapsed := end.Sub(start).Seconds()
	log.Infof("Took %f seconds to insert %d pings in the database\n", elapsed, nInsert)
	db.Close()
}

func TestInsertBulk(t *testing.T){
	label := "controller_test_bulk_no_regression"
	deletePingByLabel(label)
	defer deletePingByLabel(label)
	db := createDB()
	go monitorDB(db)
	// Create fake ping measurement 
	
	
	maxRevtrMeasurementIDPing, err := db.GetMaxRevtrMeasurementID("pings")
	if err != nil {
		panic(err)
	}
	nInsert := 10000
	pings := []*dm.Ping{}
	for i := 0; i < nInsert; i++{
		ping := createPing(label, nil, maxRevtrMeasurementIDPing + uint64(i+1))
		pings = append(pings, &ping)
	}
	start := time.Now()
	_, err = db.StorePingBulk(pings)
	if err != nil {
		panic(err)
	}
	end := time.Now()
	elapsed := end.Sub(start).Seconds()
	log.Infof("Took %f seconds to insert %d pings bulk in the database\n", elapsed, nInsert)
	
	time.Sleep(30*time.Second)
	db.Close()

}

func TestInsertBulkOpenConnections(t *testing.T){
	// Check that connections are not considered as used after each insert
	label := "controller_test_bulk_no_regression"
	deletePingByLabel(label)
	defer deletePingByLabel(label)
	db := createDB()
	defer db.Close()
	go monitorDB(db)


	maxRevtrMeasurementIDPing, err := db.GetMaxRevtrMeasurementID("pings")
	if err != nil {
		panic(err)
	}

	start := time.Now()

	// Create fake ping measurement 
	// Simulate revtr pings coming in the system. 
	var wg sync.WaitGroup
	nBatch := 10000
	for i := 0; i < nBatch; i++ {
		if i % 100 == 0 {
			log.Infof("Done inserting %d batch of pings", i)
		}

		wg.Add(1)
		time.Sleep(5 * time.Millisecond)
		go func(k int) {
			defer wg.Done()
			ping := createPing(label, nil, maxRevtrMeasurementIDPing + uint64(k + 1))
			nInsert := 1
			pings := []*dm.Ping{}
			for i := 0; i < nInsert; i++{
				pings = append(pings, &ping)
			}
			_, err := db.StorePingBulk(pings)
			if err != nil {
				panic(err)
			}
			
		} (i)
	}
	wg.Wait()
	end := time.Now()
	elapsed := end.Sub(start).Seconds()
	log.Infof("Took %f seconds to insert %d pings bulk in the database\n", elapsed, nBatch)
	// time.Sleep(30 * time.Second)
}

func TestInsertBulkRowNoRegression(t *testing.T){
	// Delete the rows previously created  
	labelBulk := "controller_test_bulk_no_regression_rr"
	_, err := deletePingByLabel(labelBulk)
	if err != nil {
		panic(err)
	}


	db := createDB()
	// Create fake ping measurement 
	recordRouteResponses0 := [] uint32 {}
	recordRouteResponses0 = append(recordRouteResponses0, 1081933877, 1081937169, 1047757060, 1047756911,
		1534840602, 1449915076, 1534840631, 1534840669, 1534840546)
	recordRouteResponses1 := [] uint32 {}
	recordRouteResponses1 = append(recordRouteResponses1, 1081933877, 1081937169, 1047757060, 1047756911,
			1534840602, 1449915076, 1534840631, 1534840669, 1534840547) // Last element is different
	
	maxRevtrMeasurementIDPing, err := db.GetMaxRevtrMeasurementID("pings")
	if err != nil {
		panic(err)
	}
	ping0 := createPing(labelBulk, recordRouteResponses0, maxRevtrMeasurementIDPing + 1)
	ping1 := createPing(labelBulk, recordRouteResponses1, maxRevtrMeasurementIDPing + 2)
	ping0Bulk := createPing(labelBulk, recordRouteResponses0, maxRevtrMeasurementIDPing + 3)
	ping1Bulk := createPing(labelBulk, recordRouteResponses1, maxRevtrMeasurementIDPing + 4)
	pingsBulk := []*dm.Ping{}

	pingsBulk = append(pingsBulk, &ping0Bulk)
	pingsBulk = append(pingsBulk, &ping1Bulk)
	_, err = db.StorePingBulk(pingsBulk)
	if err != nil {
		panic(err)
	}

	pings := []*dm.Ping{}
	pings = append(pings, &ping0)
	pings = append(pings, &ping1)
	for _, ping := range pings {
		_, err := db.StorePing(ping)
		if err != nil {
			panic(err)
		}
	}

	// Now retrieve the data and ensure that they are the same
	fetchedPings, err := db.GetPingByLabel(labelBulk)
	if err != nil {
		panic(err)
	}
	
	var rr08, rr18, rr28, rr38 uint32
	rr08 = fetchedPings[0].Responses[0].RR[8]
	rr18 = fetchedPings[1].Responses[0].RR[8]
	rr28 = fetchedPings[2].Responses[0].RR[8]
	rr38 = fetchedPings[3].Responses[0].RR[8]
	assert.EqualValues(t, rr08, rr28)
	assert.EqualValues(t, rr18, rr38)
	assert.EqualValues(t, rr08, 1534840546)
	assert.EqualValues(t, rr18, 1534840547)

	// Delete the rows created  
	_, err = deletePingByLabel(labelBulk)
	if err != nil {
		panic(err)
	}
	// assert.EqualValues(t, rowsAffected, 4)
	db.Close()
}

func sendPings(controllerClient controllerapi.ControllerClient, pms []*dm.PingMeasurement) []*dm.Ping {
	
	st, err := controllerClient.Ping(context.Background(), &dm.PingArg{Pings: pms[:]})
	if err != nil {
		panic(err)
	}

	pings := [] *dm.Ping {}
	
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		pings = append(pings, p)
	}
	return pings
}

func sendTraceroutes(tms []*dm.TracerouteMeasurement) []*dm.Traceroute {
	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	st, err := controllerClient.Traceroute(context.Background(), &dm.TracerouteArg{Traceroutes: tms[:]})
	if err != nil {
		panic(err)
	}

	traceroutes := [] *dm.Traceroute {}
	
	for {
		t, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if t.Error != "" {
			log.Errorf(t.Error)
		}
		traceroutes = append(traceroutes, t)
	}
	return traceroutes
}

func assertCache (t *testing.T, pms []*dm.PingMeasurement, expected bool) *dm.Ping {
	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	require.EqualValues(t, len(pms), 2)
	ps := sendPings(controllerClient, pms[:1])
	p := ps[0]
	require.False(t, p.FromCache)
	ps = sendPings(controllerClient, pms[1:2])
	p  = ps[0]
	require.EqualValues(t, p.FromCache, expected)
	return p 
}

func assertCachePingMeasurement(controllerClient controllerapi.ControllerClient, t *testing.T, pm *dm.PingMeasurement, expected bool) *dm.Ping {
	// controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	// if err != nil {
	// 	panic(err)
	// }
	ps := sendPings(controllerClient, []*dm.PingMeasurement{pm}) 
	p := ps[0]
	srcS, _ := util.Int32ToIPString(p.Src)
	dstS, _ := util.Int32ToIPString(p.Dst)
	require.EqualValues(t, p.FromCache, expected, fmt.Sprintf("%s, %s, %s, %s", strconv.FormatInt(int64(p.Src), 10), strconv.FormatInt(int64(p.Dst), 10),
	 srcS, dstS))
	
	spooferS, _ := util.Int32ToIPString(p.SpoofedFrom)
	log.Infof("Retrieved from cache %t ping from %s (%d) to %s (%d) spoofed as %s (%d)", 
	expected, srcS, p.Src, dstS, p.Dst, spooferS, p.SpoofedFrom)
	return p 
	
}

func TestSendPing(t *testing.T) {
	label := "controller_test_ping_insert"
	_, err := deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	src, _ := util.IPStringToInt32("195.113.161.14")
	dst, _ := util.IPStringToInt32("62.40.98.192")
	pms := [] * dm.PingMeasurement {}
	pm := &dm.PingMeasurement{
		Src:        src,
		Dst:        dst,
		// SAddr:      recv,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: false,
		// Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 60,
		FromRevtr: true,
	}
	pms = append(pms, pm)
	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	ps := sendPings(controllerClient, pms)
	p := ps[0]

	// Check that the counter has been incremented
	assert.NotEqual(t, p.RevtrMeasurementId, 0)
	time.Sleep(3*time.Second)
	// Check that it has been inserted in the DB
	db := createDB()
	insertedPings, err := db.GetPingByLabel(label)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, len(insertedPings), 1)
	db.Close()
	
}

func TestSendPingMultiple(t *testing.T) {
	label := "controller_test_ping_insert_mutiple"
	_, err := deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")
	srcs := [] uint32 {1357899187, 2159111539, 2501117299, 2736135193}
	 
	pms := [] * dm.PingMeasurement {}
	dst1, err := util.IPStringToInt32("8.8.8.8")
	dst2, err := util.IPStringToInt32("1.1.1.1")

	dsts := [] uint32 {dst1, dst2}
	for _, src := range(srcs) {
		for _, dst := range(dsts) {
			pm := &dm.PingMeasurement{
				Src:        src,
				Dst:        dst,
				// SAddr:      recv,
				Timeout:    60,
				Count:      "5",
				Staleness:  43800,
				CheckCache: false,
				// Spoof:      true,
				RR:         true,
				Label:      label, 
				SaveDb:     true,
				SpoofTimeout: 60,
				FromRevtr: true,
			}
			pms = append(pms, pm)	
		}
		
	} 
	
	nPings := 1
	
	now := time.Now()
		
	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	// errorTypes := map[string] int {}
	for i := 0; i< nPings; i++ {
		wg.Add(1)
		// time.Sleep(100 * time.Millisecond)
		go func() {
			defer wg.Done()
			sendPings(controllerClient, pms)
			// ps := sendPings(pms)
			// p := ps[0]
			// if _, ok := errorTypes[p.Error]; ok {
			// 	errorTypes[p.Error] += 1
			// } else {
			// 	errorTypes[p.Error] = 0
			// }

			// Check that the counter has been incremented
			// assert.NotEqual(t, p.RevtrMeasurementId, 0)
		} ()
	} 
	wg.Wait()
	execTime := time.Now().Sub(now)
	log.Infof("Took %s to run test ping", execTime)
	time.Sleep(3*time.Second)
	// for errorType, count := range(errorTypes) {
	// 	fmt.Printf("%s, %d\n", errorType, count)
	// }
	// Check that it has been inserted in the DB
	inserted := selectPingByLabel(label)
	assert.Equal(t, inserted, int64(nPings) * int64(len(pms)))
	time.Sleep(2*time.Second)
}

func TestSendTraceroute(t *testing.T) {
	label := "controller_test_traceroute_insert"
	_, err := deleteTracerouteByLabel(label)
	defer deleteTracerouteByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")
	src := uint32(2164945206)
	dst, _ := util.IPStringToInt32("62.40.98.192")
	

	pms := [] * dm.TracerouteMeasurement {}
	pm := &dm.TracerouteMeasurement{
		Src:        src,
		Dst:        dst,
		CheckCache: false,
		CheckDb:    false,
		SaveDb:     true,
		Staleness:  15,
		Timeout:    10,
		Wait:       "2",
		Attempts:   "1",
		// LoopAction: "1",
		Loops:      "3",
		FromRevtr: true,
		Label: label,
	}
	pms = append(pms, pm)
	ps := sendTraceroutes(pms)
	p := ps[0]

	// Check that the counter has been incremented
	assert.NotEqual(t, p.RevtrMeasurementId, 0)
	time.Sleep(2*time.Second)
	// Check that it has been inserted in the DB
	inserted := selectTracerouteByLabel(label)
	assert.Equal(t, inserted, int64(1))
	
}

func TestSendTracerouteMultiple(t *testing.T) {
	label := "controller_test_traceroute_insert"
	_, err := deleteTracerouteByLabel(label)
	defer deleteTracerouteByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")

	pms := [] * dm.TracerouteMeasurement {}
	pm := &dm.TracerouteMeasurement{
		Src:        69426700,
		Dst:        68027161,
		CheckCache: false,
		CheckDb:    false,
		SaveDb:     true,
		Staleness:  15,
		Timeout:    10,
		Wait:       "2",
		Attempts:   "1",
		LoopAction: "1",
		Loops:      "3",
		FromRevtr: true,
		Label: label,
	}
	pms = append(pms, pm)
	nTraceroutes := 2000
	wg := sync.WaitGroup{}
	for i := 0; i< nTraceroutes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ps := sendTraceroutes(pms)
			p := ps[0]

			// Check that the counter has been incremented
			assert.NotEqual(t, p.RevtrMeasurementId, 0)
		} ()
	} 
	wg.Wait()
	time.Sleep(2*time.Second)
	// Check that it has been inserted in the DB
	inserted := selectTracerouteByLabel(label)
	assert.Equal(t, inserted, int64(nTraceroutes))
	
}

func TestPingRR(t *testing.T){
	err := test.ClearCache(11213)
	label := "controller_test_rr_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")

	srcI , dstI := uint32(703010073), uint32(3093903206)
	pms := [] * dm.PingMeasurement {}
	pm := &dm.PingMeasurement{
		Src:        srcI,
		// Dst:        68027161,
		Dst: dstI,
		// SAddr:      recv,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: false,
		// Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 60,
		FromRevtr: true,
	}
	pms = append(pms, pm)
	pms = append(pms, pm)

	assertCache(t, pms, false)
}

func TestRRPingTCPdump(t *testing.T) {
	// Prepare
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	} 
	label := "controller_test_rr_tcpdump"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}

	// Must use walter's source IP because the capture is done on Walter...
	// src, _ := util.IPStringToInt32("129.10.113.54")
	src, _ := util.IPStringToInt32("195.113.161.14")
	
	// src, _ := util.IPStringToInt32("4.14.3.51")
	dstS := "62.40.98.192"
	dst, _ := util.IPStringToInt32(dstS)

	// src, _ := util.IPStringToInt32("190.94.191.179")
	// dst, _ := util.IPStringToInt32("121.199.77.152")
	

	pm := &dm.PingMeasurement{
		// Src:        2736135193,
		// Dst:        1792950661,
		Src:        src,
		Dst:        dst,
		// SAddr:      recv,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		Payload: "magic string",
		// Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 60,
		FromRevtr: true,
	}
	pms := [] *dm.PingMeasurement {pm}
	// Start tcpdump 
	handle, err := pcap.OpenLive("em1", defaultSnapLen, true,
				     pcap.BlockForever)
	if err != nil {
		panic(err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter(fmt.Sprintf("icmp and dst %s", dstS)); err != nil {
		panic(err)
	}

	packets := gopacket.NewPacketSource(
		handle, handle.LinkType()).Packets()
	log.Infof("%s", "Sniffing packets...")
	sendPings(controllerClient, pms)
	// Test
	for packet := range packets { 
		log.Infof("%s", packet.String())
		if transp := packet.TransportLayer(); transp != nil {
			// require.Contains(t, string(transp.()), "magic string")
			break
		}	
	}

	// Clean 
}
func TestPingRRCache(t *testing.T){
	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	} 
	label := "controller_test_rr_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}

	src, _ := util.IPStringToInt32("195.113.161.14")
	// src, _ := util.IPStringToInt32("4.14.3.51")
	dst, _ := util.IPStringToInt32("62.40.98.191")

	// src, _ := util.IPStringToInt32("190.94.191.179")
	// dst, _ := util.IPStringToInt32("121.199.77.152")
	

	pm := &dm.PingMeasurement{
		// Src:        2736135193,
		// Dst:        1792950661,
		Src:        src,
		Dst:        dst,
		// SAddr:      recv,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		// Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 60,
		FromRevtr: true,
	}

	assertCachePingMeasurement(controllerClient, t, pm, false)
	time.Sleep(100 * time.Millisecond)
	assertCachePingMeasurement(controllerClient, t, pm, true)
}

var vpTest = "cesnet"
var spooferType = "sandbox"

func initSpoofConfig(vpTest string, spooferType string) (string, string, string, string, uint32, uint32, uint32) {

	var srcS, spooferS, dstS, spoofedAddr string 
	var src, dst, spoofer uint32
	if vpTest == "walter" {
		srcS = "129.10.113.54"
		src, _ = util.IPStringToInt32(srcS)
		
		dstS = "18.2.4.46"
		dst, _ = util.IPStringToInt32(dstS)
		

	} else if vpTest == "cesnet" {
		srcS = "195.113.161.14"
		
		src, _ = util.IPStringToInt32(srcS)
		
		dstS = "62.40.98.191"
		dst, _ = util.IPStringToInt32(dstS)
		
		
	} else if vpTest == "sandbox" {
		srcS = "4.14.159.76"
		src, _ = util.IPStringToInt32(srcS)
		
		dstS = "195.113.161.14"
		dst, _ = util.IPStringToInt32(dstS)
		
	} else if vpTest == "vpservice" {
		src = 68027187
		dst = 68027161
		// 68027148 to 68027161 spoofed as 68027187
		// 68027161 to 68027148 spoofed as 68027187
		// 68027161 to 68027187 spoofed as 68027148
		// 68027187 to 68027161 spoofed as 68027148 
		
	}

	spoofedAddr, _ = util.Int32ToIPString(src)

	if spooferType == "self" {
		spooferS = srcS
		spoofer = src
		
	} else if spooferType == "sandbox" {
		spooferS = "4.14.3.12"
		spoofer, _ = util.IPStringToInt32(spooferS)
	} else if spooferType == "vpservice" {
		spoofer = 68027148
		spooferS, _ = util.Int32ToIPString(spoofer)
	} else {
		panic("No spoofer type provided")
	}

	return srcS, spooferS, dstS, spoofedAddr, src, dst, spoofer
}

func TestPingSpoofedRR(t * testing.T) {
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	} 
	label := "controller_test_spoofed"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	_, _, _, spoofedAddr, src, dst, spoofer := initSpoofConfig(vpTest, spooferType)
	
	pms := [] * dm.PingMeasurement {}
	
	for i := 0 ; i < 1; i++ {
		pm := &dm.PingMeasurement{
			Src:        spoofer,
			Dst:        dst,
			Sport: strconv.Itoa(50000 + i),
			SpooferAddr: src,
			SAddr:      spoofedAddr,
			Timeout:    20,
			Count:      "1",
			// Wait: "0.001",
			Staleness:  43800,
			CheckCache: false,
			Spoof:      true,
			RR:         true,
			Label:      label, 
			SaveDb:     true,
			SpoofTimeout: 20,
			IsAbsoluteSpoofTimeout: true,
			FromRevtr: true,
		}
		pms = append(pms, pm)
	}
	
	

	ps := sendPings(controllerClient, pms)
	require.Equal(t, len(ps), len(pms))
	p := ps[0]
	require.Equal(t, p.Src, src)
	require.Equal(t, p.SpoofedFrom, spoofer)
	require.Equal(t, len(p.Responses), 1)
	require.True(t, p.Responses[0].RR != nil)
}

func TestPingSpoofedRRCache(t *testing.T){	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	} 
	label := "controller_test_spoofed_rr_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}


	vpTest := "sandbox"
	spooferType := "self"

	srcS, spooferS, _, spoofedAddr, src, dst, spoofer := initSpoofConfig(vpTest, spooferType)
	
	pms := [] * dm.PingMeasurement {}
	
	pm := &dm.PingMeasurement{
		Src:        spoofer,
		Dst:        dst,
		Sport: "50000",
		SpooferAddr: src,
		SAddr:      spoofedAddr,
		Timeout:    20,
		Count:      "1",
		// Wait: "0.001",
		Staleness:  43800,
		CheckCache: true,
		Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 20,
		IsAbsoluteSpoofTimeout: true,
		FromRevtr: true,
	}
	pms = append(pms, pm)
	pms = append(pms, pm)

	p := assertCache(t, pms, true)
	// Test that a hop is also cached

	hopsAfterDestination := map [uint32] struct{}{}
	for _, r := range(p.Responses[:1]) {
		isAfterDestination := false
		for _, rrHop := range(r.RR) {
			if rrHop == dst {
				isAfterDestination = true
			}
			if  isAfterDestination {
				hopsAfterDestination[rrHop] = struct{}{}
			}
		}
	}
	for _, r := range(p.Responses[:1]) {
		for _, rrHop := range(r.RR) {
			// if !isAfterDestination {
			// 	continue
			// }
			rrHopS, _ := util.Int32ToIPString(rrHop)
			if iputil.IsPrivate(net.ParseIP(rrHopS)){
				continue
			}
			shouldBeCached := false
			if _, ok := hopsAfterDestination[rrHop]; ok {
				shouldBeCached = true
			}
			
			log.Infof("Sending spoofed ping from %s to %s spoofed as %s", srcS, rrHopS, spooferS)
			
			if rrHop != 0 && rrHop != src{
				pm = &dm.PingMeasurement{
						Src:        spoofer,
						Dst:        rrHop,
						SpooferAddr: src,
						SAddr:      spoofedAddr,
						Sport: "50002",
						Timeout:    20,
						Count:      "1",
						Staleness:  43800,
						CheckCache: true,
						Spoof:      true,
						RR:         true,
						Label:      label, 
						SaveDb:     true,
						SpoofTimeout: 5,
						IsAbsoluteSpoofTimeout: true,
						FromRevtr: true,
				}
				assertCachePingMeasurement(controllerClient, t, pm, shouldBeCached)
				
			}

		}



	}
}

func TestPingSpoofedRRSiteCache(t *testing.T){
	err := test.ClearCache(11213)
	label := "controller_test_spoofed_rr_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	pms := [] * dm.PingMeasurement {}
	spoofedAddr, err := util.Int32ToIPString(71815910)
	
	pm := &dm.PingMeasurement{
		Src:        71802252,
		Dst:        68067148,
		SpooferAddr: 71815910,
		SAddr:      spoofedAddr,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 5,
		IsAbsoluteSpoofTimeout: true,
	}
	pms = append(pms, pm)
	pm = &dm.PingMeasurement{
		Src:        71802278,
		Dst:        68067148,
		SpooferAddr: 71815910,
		SAddr:      spoofedAddr,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 5,
		IsAbsoluteSpoofTimeout: true,
	}
	
	pms = append(pms, pm)

	assertCache(t, pms, true)
}

func TestPingTSCache(t *testing.T){
	err := test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	label := "controller_test_ts_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	dummyIP := "132.227.123.12"
	pms := [] * dm.PingMeasurement {}
	// spoofedAddr, err := util.Int32ToIPString(71815910)
	timestampDstS, _ := util.Int32ToIPString(3549005809) // Should be responsive to TS
	pm := &dm.PingMeasurement{
		Src:        68067187,
		Dst:        3549005809,
		// SpooferAddr: 71815910,
		// SAddr:      spoofedAddr,
		TimeStamp: ipoptions.TimestampAddrToString([]string{dummyIP, timestampDstS}),
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		// Spoof:      true,
		// RR:         false,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 5,
		IsAbsoluteSpoofTimeout: true,
	}
	pms = append(pms, pm)
	pms = append(pms, pm)

	assertCache(t, pms, false)
}

func TestPingSpoofedTSCache(t *testing.T){
	err := test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	label := "controller_test_spooofed_ts_cache"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	// dummyIP := "132.227.123.12"
	pms := [] * dm.PingMeasurement {}
	spoofedAddr, err := util.Int32ToIPString(68067187)
	timestampDstS, _ := util.Int32ToIPString(3549005809) // Should be responsive to TS
	pm := &dm.PingMeasurement{
		Src:        2159114201, // spoofer
		Dst:        3549005809,
		SpooferAddr: 68067187,
		SAddr:      spoofedAddr,
		TimeStamp: ipoptions.TimestampAddrToString([]string{timestampDstS}),
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		Spoof:      true,
		// RR:         false,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 5,
		IsAbsoluteSpoofTimeout: true,
	}
	pms = append(pms, pm)
	pms = append(pms, pm)

	assertCache(t, pms, false)
}


func TestPingRRConnections(t *testing.T){
	// Ensure controller is started
	err := test.ClearCache(11213)
	label := "controller_test_rr_connections"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}


	sources := util.IPsFromFile("../rankingservice/algorithms/evaluation/resources/sources.csv") 

	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for i := 0 ; i < 10000; i++ {
		if i % 10000 == 0 {
			log.Infof("Done inserting %d batch of pings", i)
		}

		sourceIndex := rand.Intn(len(sources))
		source, _ := util.IPStringToInt32(sources[sourceIndex])
		destinationIndex := rand.Intn(len(sources))
		destination, _ := util.IPStringToInt32(sources[destinationIndex])
		pm := &dm.PingMeasurement{
			// Non spoofed 
			// Src:        71815910,
			// Dst:        68067148,
			// // SAddr:      recv,
			// Spoofed
			Src:        source,
			Dst:        destination,
			Timeout:    20,
			Count:      "1",
			Staleness:  43800,
			// CheckCache: true,
			// Spoof:      true,
			RR:         true,
			Label:      label, 
			SaveDb:     true,
			// SpoofTimeout: 60,
			FromRevtr: true,
		}
		wg.Add(1)
		time.Sleep(100 * time.Nanosecond)
		go func(pmArg *dm.PingMeasurement ) {
			defer wg.Done()
			pms := [] * dm.PingMeasurement {}
			pms = append(pms, pmArg)
			
			st, err := controllerClient.Ping(context.Background(), &dm.PingArg{Pings: pms[:1]})
			if err != nil {
				log.Error(err)
			}
			for {
				p, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error(err)
				}
				// require.False(t, p.FromCache)
				if p.FromCache {
					print("From cache\n")
				}
			}
		} (pm)
	}
	wg.Wait()
}

func TestTracerouteConnections(t *testing.T){
	// Ensure controller is started
	err := test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	label := "controller_test_traceroute_connections"
	_, err = deletePingByLabel(label)
	defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	sources := util.IPsFromFile("../rankingservice/algorithms/evaluation/resources/sources.csv") 



	
	controllerClient , err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for i := 0 ; i < 50000; i++ {
		if i % 100 == 0 {
			log.Infof("Done inserting %d batch of tr", i)
		}
		wg.Add(1)
		time.Sleep(5 * time.Millisecond)
		// time.Sleep(30 * time.Millisecond)
		sourceIndex := rand.Intn(len(sources))
		source, _ := util.IPStringToInt32(sources[sourceIndex])
		go func() {
			defer wg.Done()
			pms := [] * dm.TracerouteMeasurement {}
			pm := &dm.TracerouteMeasurement{
				// Non spoofed 
				Src:        source,
				Dst:        68067148,
				CheckCache: false,
				CheckDb:    false,
				SaveDb:     true,
				Staleness:  15,
				Timeout:    10,
				Wait:       "2",
				Attempts:   "1",
				LoopAction: "1",
				Loops:      "3",
			}
			pms = append(pms, pm)
			st, err := controllerClient.Traceroute(context.Background(), &dm.TracerouteArg{Traceroutes: pms[:1]})
			if err != nil {
				panic(err)
			}
			for {
				p, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					panic(err)
				}
				require.False(t, p.FromCache)
			}
		} ()
	}
	wg.Wait()
}

func TestVPLoad(t *testing.T){
	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	label := "controller_test_vp_load"
	_, err = deletePingByLabel(label)
	// defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")
	// src, _ := util.IPStringToInt32("4.14.3.51")
	
	// src, _ := util.IPStringToInt32("")
	src := uint32(2735178035)
	dst, _ := util.IPStringToInt32("195.178.64.238")

	

	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup {}
	for i := 0; i < 10; i++ {
		sPort := 29000 + i
		wg.Add(1)
		go func() {
			defer wg.Done()
			pms := [] * dm.PingMeasurement {}
			pm := &dm.PingMeasurement{
				Src:        src,
				Dst:        dst,
				Sport: strconv.Itoa(sPort),
				// SAddr:      recv,
				Timeout:    20,
				Count:      "1",
				Staleness:  43800,
				CheckCache: false,
				// Spoof:      true,
				RR:         true,
				Label:      label, 
				SaveDb:     true,
				SpoofTimeout: 60,
				FromRevtr: true,
			}
			pms = append(pms, pm)	
			prs := sendPings(controllerClient, pms)	
			for _, pr := range(prs) {
				require.Equal(t, pr.Error, "")
				require.Equal(t, pr.PingSent, uint32(1))
			
			}	
		} ()
	}

	wg.Wait()
	

}

func TestVPLoadSpoofed(t *testing.T){
	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	label := "controller_test_vp_load"
	_, err = deletePingByLabel(label)
	// defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	spoofedAs, _ :=  util.Int32ToIPString(3517607590)
	// spoofedAs := ""
	// src, _ := util.IPStringToInt32("195.113.161.14")
	// dst, _ := util.IPStringToInt32("195.178.64.238")
	// Walter's VP
	// src, _ := util.IPStringToInt32("129.10.113.54")
	src := uint32(1296265779)
	// spoofedAs, _ = util.Int32ToIPString(src)
	dst, _ := util.IPStringToInt32("195.113.235.109")
	
	
	
	

	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup {}
	for i := 0; i < 1; i++ {
		wg.Add(1)
		sPort := 50000 + i 
		go func(sPortArg int) {
			defer wg.Done()
			pms := [] * dm.PingMeasurement {}
			pm := &dm.PingMeasurement{
				Src:        src,
				Dst:        dst,
				SAddr:      spoofedAs,
				Sport: strconv.Itoa(sPortArg),
				Timeout:    20,
				Count:      "1",
				Staleness:  43800,
				CheckCache: false,
				Spoof:      true,
				RR:         true,
				Label:      label, 
				SaveDb:     true,
				SpoofTimeout: 20,
				FromRevtr: true,
			}
			
			pms = append(pms, pm)	
			prs := sendPings(controllerClient, pms)	
			for _, pr := range(prs) {
				require.Equal(t, pr.Error, "")
				require.Equal(t, pr.PingSent, uint32(1))
				
			}	
		} (sPort)
	}

	wg.Wait()
	
	

}

func TestVPLoadMultiDst(t *testing.T){
	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	label := "controller_test_vp_load"
	_, err = deletePingByLabel(label)
	// defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	// src, _ := util.IPStringToInt32("195.113.161.14")

	

	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pms := [] * dm.PingMeasurement {}
			pm := &dm.PingMeasurement{
				Src:        2731893772,
				Dst:        rand.Uint32(),
				// SAddr:      recv,
				Timeout:    20,
				Count:      "1",
				Staleness:  43800,
				CheckCache: true,
				// Spoof:      true,
				RR:         true,
				Label:      label, 
				SaveDb:     true,
				SpoofTimeout: 60,
				FromRevtr: true,
			}
			pms = append(pms, pm)	
			prs := sendPings(controllerClient, pms)	
			for _, pr := range(prs) {
				require.Equal(t, pr.Error, "")
				require.Equal(t, pr.PingSent, uint32(1))
			}
			
			
		} ()
		
	}
	wg.Wait()
		
}

func TestCacheLoad(t *testing.T){
	cachePort := 11213
	err := test.ClearCache(cachePort)
	if err != nil {
		panic(err)
	}
	label := "controller_test_cache_load"
	_, err = deletePingByLabel(label)
	// defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	vpserviceClient , err := survey.CreateVPServiceClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}
	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}

	vpResponse , err := vpserviceClient.GetVPs(context.Background(), &pb.VPRequest{})
	trials := 20000
	requestsChan := make(chan int, trials)
	responseChan := make(chan int, trials)
	var wg sync.WaitGroup
	for i := 0; i < trials; i++ {
		wg.Add(1)
		go func(){
			defer wg.Done()
			vpSrc := vpResponse.Vps[uint32(rand.Intn(len(vpResponse.Vps)))]
			if !vpSrc.RecordRoute {
				return
			}
			pm := &dm.PingMeasurement{
				Src:        vpSrc.Ip,
				Dst:        rand.Uint32(),
				// SAddr:      recv,
				Timeout:    20,
				Count:      "1",
				Staleness:  43800,
				CheckCache: true,
				// Spoof:      true,
				RR:         true,
				Label:      label, 
				SaveDb:     true,
				SpoofTimeout: 60,
				FromRevtr: true,
			}
			requestsChan <- 1
			p := assertCachePingMeasurement(controllerClient, t, pm, false)
			if p.Src != pm.Src {
				log.Errorf("Bad address src %d, %d", p.Src, pm.Src)
				return
			}
			if p.Error ==  "" {
				time.Sleep(200 * time.Millisecond)
				assertCachePingMeasurement(controllerClient, t, pm, true)
				responseChan <- 1
			}
			


		} ()
	}
	wg.Wait()
	close(responseChan)
	close(requestsChan)
	log.Infof(fmt.Sprintf("%d ping should be sent", len(requestsChan)))
	log.Infof(fmt.Sprintf("%d ping sent", len(responseChan)))
		
}

func TestCacheUnresponsive(t *testing.T){
	
	err := test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	label := "controller_test_cache_unresponsive"
	_, err = deletePingByLabel(label)
	// defer deletePingByLabel(label)
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("../cmd/controller/root.crt")
	if err != nil {
		panic(err)
	}

	src, _ := util.IPStringToInt32("195.113.161.14")

	// Unresponsive address
	dst, _ := util.IPStringToInt32("132.227.123.7")

	pm := &dm.PingMeasurement{
		Src:        src,
		Dst:        dst,
		// SAddr:      recv,
		Timeout:    20,
		Count:      "1",
		Staleness:  43800,
		CheckCache: true,
		// Spoof:      true,
		RR:         true,
		Label:      label, 
		SaveDb:     true,
		SpoofTimeout: 60,
		FromRevtr: true,
	}
	assertCachePingMeasurement(controllerClient, t, pm, false)
	p := assertCachePingMeasurement(controllerClient, t, pm, true)
	assert.Equal(t, p.Id, int64(-1))

}

func exit(status int) {
	os.Exit(status)
}
