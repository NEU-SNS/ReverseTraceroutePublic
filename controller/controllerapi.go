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

// Package controller is the library for creating a central controller
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	cont "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/gogo/protobuf/jsonpb"
	con "golang.org/x/net/context"
)

func (c *controllerT) Ping(pa *dm.PingArg, stream cont.Controller_PingServer) error {
	pms := pa.GetPings()
	for _, px := range pms {
		log.Debug("Printing a PingMeasurement (Ping):", px.Label, ",", px.Src, ",", px.Dst, ",", px.RR)
	}
	if pms == nil {
		return nil
	}
	start := time.Now()
	ctx, cancel := con.WithCancel(stream.Context())
	defer cancel()
	res := c.doPing(ctx, pms)
	for {
		select {
		case p, ok := <-res:
			if !ok {
				end := time.Since(start).Seconds()
				pingResponseTimes.Observe(end)
				return nil
			}
			if err := stream.Send(p); err != nil {
				log.Error(err)
				end := time.Since(start).Seconds()
				pingResponseTimes.Observe(end)
				return err
			}
		case <-ctx.Done():
			end := time.Since(start).Seconds()
			pingResponseTimes.Observe(end)
			return ctx.Err()
		}
	}
}

func (c *controllerT) Traceroute(ta *dm.TracerouteArg, stream cont.Controller_TracerouteServer) error {
	tms := ta.GetTraceroutes()
	if tms == nil {
		return nil
	}
	start := time.Now()
	ctx, cancel := con.WithCancel(stream.Context())
	defer cancel()
	res := c.doTraceroute(ctx, tms)
	for {
		select {
		case t, ok := <-res:
			if !ok {
				end := time.Since(start).Seconds()
				tracerouteResponseTimes.Observe(end)
				return nil
			}
			if err := stream.Send(t); err != nil {
				log.Error(err)
				end := time.Since(start).Seconds()
				tracerouteResponseTimes.Observe(end)
				return err
			}
		case <-ctx.Done():
			end := time.Since(start).Seconds()
			tracerouteResponseTimes.Observe(end)
			return ctx.Err()
		}
	}
}

//Priority is the priority for ping request
type Priority uint32

//PingReq is a request for pings
type PingReq struct {
	Pings    []Ping   `json:"pings,omitempty"`
	Priority Priority `json:"priority,omitempty"`
}

//Ping is an individual measurement
type Ping struct {
	Src       string `json:"src,omitempty"`
	Dst       string `json:"dst,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Count     int    `json:"count,omitempty"`
	Timeout   int     `json:"timeout,omitempty"`
	IsRecordRoute bool `json:"is_record_route,omitempty"`
	IsCheckCache bool `json:"is_check_cache,omitempty"`
	IsSaveDB bool `json:"is_save_db,omitempty"`
	Label string `json:"label,omitempty"`
	IsRipeAtlas          bool     `json:"is_ripe,omitempty"`
	RipeApiKey           string   `json:"ripe_api_key,omitempty"`
	RipeQueryUrl         string   `json:"ripe_query_url,omitempty"`
	RipeQueryBody        string   `json:"ripe_query_body,omitempty"`

}

// TraceReq is a request for traceroutes
type TraceReq struct {
	Traceroutes []Trace  `json:"traceroutes,omitempty"`
	Priority    Priority `json:"priority,omitempty"`
}

// Trace is an individual traceroute measurement
type Trace struct {
	Src      string `json:"src,omitempty"`
	Dst      string `json:"dst,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	IsRipeAtlas          bool     `json:"is_ripe,omitempty"`
	RipeApiKey           string   `json:"ripe_api_key,omitempty"`
	RipeQueryUrl         string   `json:"ripe_query_url,omitempty"`
	RipeQueryBody        string   `json:"ripe_query_body,omitempty"`
}



const (
	apiKey                = "Api-Key"
	v1Prefix              = "/api/v1/"
	maxProbeCount         = 4
	maxTracerouteAttempts = 4
)

func (c *controllerT) verifyKey(key string) (dm.User, bool) {
	u, err := c.db.GetUser(key)
	if err != nil {
		log.Error(err)
		return u, false
	}
	log.Debug("Got user ", u)
	return u, true
}

func (c *controllerT) GetPingsHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id := req.FormValue("id")
	idi, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Debug("Getting pings for batch ", idi)
	pings, err := c.db.GetPingBatch(u, idi)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	var marsh jsonpb.Marshaler
	if err := marsh.Marshal(rw, &dm.PingArgResp{Pings: pings}); err != nil {
		panic(err)
	}
}

func (c *controllerT) GetPingsBatchNResponsesHandler(rw http.ResponseWriter, req *http.Request){
	if req.Method != http.MethodGet {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id := req.FormValue("id")
	idi, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Debug("Getting pings for batch ", idi)
	count, err := c.db.GetPingBatchNResponses(u, idi)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		Count uint32 `json:"count,omitempty"`
	}{Count: count})
	if err != nil {
		panic(err)
	}
}

func (c *controllerT) GetTracesHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id := req.FormValue("id")
	idi, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Debug("Getting traces for batch ", idi)
	traces, err := c.db.GetTraceBatch(u, idi)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	var marsh jsonpb.Marshaler
	if err := marsh.Marshal(rw, &dm.TracerouteArgResp{Traceroutes: traces}); err != nil {
		panic(err)
	}
}

func (c *controllerT) TracerouteHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	var treq TraceReq
	if err := json.NewDecoder(req.Body).Decode(&treq); err != nil {
		panic(err)
	}
	log.Debug("Running traces treq")
	var traces []*dm.TracerouteMeasurement
	for _, t := range treq.Traceroutes {
		if t.Attempts > maxTracerouteAttempts || t.Attempts < 0 {
			rw.Header().Set("Content-Type", "text/plain")
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		srci := uint32(0)
		dsti := uint32(0)
		err  := errors.New("") 
		if !t.IsRipeAtlas{
			srci, err = util.IPStringToInt32(t.Src)
			if err != nil {
				rw.Header().Set("Content-Type", "text/plain")
				http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			dsti, err = util.IPStringToInt32(t.Dst)
			if err != nil {
				rw.Header().Set("Content-Type", "text/plain")
				http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if t.Attempts == 0 {
				t.Attempts = 2
			}
		}
		
		traces = append(traces, &dm.TracerouteMeasurement{
			Src:      srci,
			Dst:      dsti,
			Attempts: fmt.Sprintf("%d", t.Attempts),
			Timeout:  20,
			IsRipeAtlas: t.IsRipeAtlas, 
			RipeApiKey: t.RipeApiKey,        
			RipeQueryUrl: t.RipeQueryUrl,
			RipeQueryBody: t.RipeQueryBody,
		})
	}
	if len(traces) == 0 {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	bid, err := c.db.AddTraceBatch(u)
	if err != nil {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	go func() {
		ctx := con.Background()
		ctx, cancel := con.WithTimeout(ctx, time.Second*120)
		defer cancel()
		results := c.doTraceroute(ctx, traces)
		var ids []int64
		for t := range results {
			ids = append(ids, t.Id)
		}
		err := c.db.AddTraceToBatch(bid, ids)
		if err != nil {
			log.Error(err)
		}
	}()
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		Results string `json:"results,omitempty"`
	}{Results: fmt.Sprintf("https://%s%s?id=%d", "revtr.ccs.neu.edu", v1Prefix+"traceroutes", bid)})
	if err != nil {
		panic(err)
	}
}

func (c *controllerT) PingHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// var preqDebug PingReq
	// preqDebug.Pings = []Ping{
	// 	{Src: "213.244.128.153", Dst: "8.8.8.8", Count: 5}}

	// debug, err := json.Marshal(preqDebug)
	// debugStr := string(debug)
	// var preqUnmarshall PingReq

	// debug, err = ioutil.ReadAll(req.Body)
	// json.Unmarshal([]byte(debug), &preqUnmarshall)
	// debugStr = string(debug)
	// println(debugStr)
	var preq PingReq
	decodedJSONBody := json.NewDecoder(req.Body)
	if err := decodedJSONBody.Decode(&preq); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		// Catch any syntax errors in the JSON and send an error message
		// which interpolates the location of the problem to make it
		// easier for the client to fix.
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			println(msg)

		// In some circumstances Decode() may also return an
		// io.ErrUnexpectedEOF error for syntax errors in the JSON. There
		// is an open issue regarding this at
		// https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			println(msg)

		// Catch any type errors, like trying to assign a string in the
		// JSON request body to a int field in our Person struct. We can
		// interpolate the relevant field name and position into the error
		// message to make it easier for the client to fix.
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			println(msg)

		// Catch the error caused by extra unexpected fields in the request
		// body. We extract the field name from the error message and
		// interpolate it in our custom error message. There is an open
		// issue at https://github.com/golang/go/issues/29035 regarding
		// turning this into a sentinel error.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			println(msg)

		// An io.EOF error is returned by Decode() if the request body is
		// empty.
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			println(msg)

		// Catch the error caused by the request body being too large. Again
		// there is an open issue regarding turning this into a sentinel
		// error at https://github.com/golang/go/issues/30715.
		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			print(msg)

		// Otherwise default to logging the error and sending a 500 Internal
		// Server Error response.
		default:
			log.Println(err.Error())
		}
		panic(err)
	}
	var pings []*dm.PingMeasurement
	for _, p := range preq.Pings {
		srci := uint32(0)
		dsti := uint32(0)
		err  := errors.New("")
		if !p.IsRipeAtlas{
			// Sanity checks of the parameters
			if p.Count > maxProbeCount || (p.Count <= 0) {
				rw.Header().Set("Content-Type", "text/plain")
				http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			srci, err = util.IPStringToInt32(p.Src)
			if err != nil {
				rw.Header().Set("Content-Type", "text/plain")
				http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			dsti, err = util.IPStringToInt32(p.Dst)
			if err != nil {
				rw.Header().Set("Content-Type", "text/plain")
				http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
		
		pings = append(pings, &dm.PingMeasurement{
			Src:     srci,
			Dst:     dsti,
			Timeout: int64(p.Timeout),
			Count:   fmt.Sprintf("%d", p.Count),
			RR:      p.IsRecordRoute, 
			CheckCache : p.IsCheckCache,
			Label: p.Label,
			IsRipeAtlas: p.IsRipeAtlas, 
			RipeApiKey: p.RipeApiKey,        
			RipeQueryUrl: p.RipeQueryUrl,
			RipeQueryBody: p.RipeQueryBody,
			SaveDb: p.IsSaveDB,
		})

	}
	bid, err := c.db.AddPingBatch(u)
	if err != nil {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	go func() {
		ctx := con.Background()
		ctx, cancel := con.WithTimeout(ctx, time.Hour*2) // ensure a very large timeout for large measurements
		defer cancel()
		results := c.doPing(ctx, pings)
		var ids []int64
		for p := range results {
			ids = append(ids, p.Id)
		}
		err := c.db.AddPingsToBatch(bid, ids)
		if err != nil {
			log.Error(err)
		}
	}()

	resultUrl := fmt.Sprintf("https://%s%s?id=%d", "revtr.ccs.neu.edu", v1Prefix+"pings", bid)
	if env.IsDebugController{
		resultUrl = fmt.Sprintf("http://localhost:%d%s?id=%d", env.ControllerAPIPortDebug, 
		v1Prefix+"pings/batch/responses", bid)
	}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		Results string `json:"results,omitempty"`
	}{Results: resultUrl})
	if err != nil {
		panic(err)
	}
}

func (c *controllerT) RecordRouteHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	var preq PingReq
	if err := json.NewDecoder(req.Body).Decode(&preq); err != nil {
		panic(err)
	}
	log.Debug("Running record routes ", preq)
	var pings []*dm.PingMeasurement
	for _, p := range preq.Pings {
		srci, err := util.IPStringToInt32(p.Src)
		if err != nil {
			rw.Header().Set("Content-Type", "text/plain")
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		dsti, err := util.IPStringToInt32(p.Dst)
		if err != nil {
			rw.Header().Set("Content-Type", "text/plain")
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		pings = append(pings, &dm.PingMeasurement{
			Src:     srci,
			Dst:     dsti,
			Timeout: 10,
			Count:   "1",
			RR:      true,
		})
	}
	if len(pings) == 0 {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	bid, err := c.db.AddPingBatch(u)
	if err != nil {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	go func() {
		ctx := con.Background()
		ctx, cancel := con.WithTimeout(ctx, time.Second*120)
		defer cancel()
		results := c.doPing(ctx, pings)
		var ids []int64
		for p := range results {
			ids = append(ids, p.Id)
		}
		err := c.db.AddPingsToBatch(bid, ids)
		if err != nil {
			log.Error(err)
		}
	}()
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		Results string `json:"results,omitempty"`
	}{Results: fmt.Sprintf("https://%s%s?id=%d", "revtr.ccs.neu.edu", v1Prefix+"pings", bid)})
	if err != nil {
		panic(err)
	}
}

func (c *controllerT) TimeStampHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(apiKey)
	u, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	var preq PingReq
	if err := json.NewDecoder(req.Body).Decode(&preq); err != nil {
		panic(err)
	}
	var pings []*dm.PingMeasurement
	for _, p := range preq.Pings {
		if p.Timestamp == "" {
			rw.Header().Set("Content-Type", "text/plain")
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		srci, err := util.IPStringToInt32(p.Src)
		if err != nil {
			rw.Header().Set("Content-Type", "text/plain")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		dsti, err := util.IPStringToInt32(p.Dst)
		if err != nil {
			rw.Header().Set("Content-Type", "text/plain")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		pings = append(pings, &dm.PingMeasurement{
			Src:       srci,
			Dst:       dsti,
			Timeout:   10,
			Count:     "1",
			TimeStamp: p.Timestamp,
		})
	}
	if len(pings) == 0 {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	bid, err := c.db.AddPingBatch(u)
	if err != nil {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	go func() {
		ctx := con.Background()
		ctx, cancel := con.WithTimeout(ctx, time.Second*120)
		defer cancel()
		results := c.doPing(ctx, pings)
		var ids []int64
		for p := range results {
			log.Debug("Got timestamp ", p)
			ids = append(ids, p.Id)
		}
		err := c.db.AddPingsToBatch(bid, ids)
		if err != nil {
			log.Error(err)
		}
	}()
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		Results string `json:"results,omitempty"`
	}{Results: fmt.Sprintf("https://%s%s?id=%d", "revtr.ccs.neu.edu", v1Prefix+"pings", bid)})
	if err != nil {
		panic(err)
	}
}

type vps struct {
	IP string
}

type vpret struct {
	VPS []vps
}

func (c *controllerT) VPSHandler(rw http.ResponseWriter, req *http.Request) {
	key := req.Header.Get(apiKey)
	_, ok := c.verifyKey(key)
	if !ok {
		rw.Header().Set("Content-Type", "text/plain")
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	ctx := con.Background()
	ctx, cancel := con.WithTimeout(ctx, time.Second*30)
	defer cancel()
	vpr, err := c.doGetVPs(ctx, &dm.VPRequest{})
	if err != nil {
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var ret vpret
	for _, vp := range vpr.GetVps() {
		ips, _ := util.Int32ToIPString(vp.Ip)
		ret.VPS = append(ret.VPS, vps{IP: ips})
	}
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(ret)
	if err != nil {
		panic(err)
	}
	return
}

func (c *controllerT) GetVPs(ctx con.Context, gvp *dm.VPRequest) (vpr *dm.VPReturn, err error) {
	vpr, err = c.doGetVPs(ctx, gvp)
	return
}

func (c *controllerT) ReceiveSpoofedProbes(probes cont.Controller_ReceiveSpoofedProbesServer) error {
	log.Debug("ReceiveSpoofedProbes")
	for {
		pr, err := probes.Recv()
		if err == io.EOF {
			return probes.SendAndClose(&dm.ReceiveSpoofedProbesResponse{})
		}
		if err != nil {
			log.Error(err)
			return err
		}
		c.doRecSpoof(probes.Context(), pr)
	}
}
