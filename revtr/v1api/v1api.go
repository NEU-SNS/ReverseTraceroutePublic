package v1api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	rs "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	repo "github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"github.com/golang/protobuf/jsonpb"
)

const (
	v1Prefix  = "/api/v1/"
	keyHeader = "Revtr-Key"
)
/*
var (
	serverAddr         = flag.String("server_addr", "localhost:49491", "The server address in the format of host:port")
)
 */

// V1Api is
type V1Api struct {
	s   server.RevtrServer
	mux *http.ServeMux
}

// NewV1Api creates a V1Api using RevtrServer s registering routes on ServeMux mux
func NewV1Api(s server.RevtrServer, mux *http.ServeMux) V1Api {
	api := V1Api{s: s, mux: mux}
	mux.HandleFunc(v1Prefix+"sources", api.sources)
	mux.HandleFunc(v1Prefix+"revtr", api.revtr)
	mux.HandleFunc(v1Prefix+"atlas/clean", api.cleanAtlas)
	mux.HandleFunc(v1Prefix+"atlas/run", api.runAtlas)
	mux.HandleFunc(v1Prefix+"spoof", api.spoof)
	return api
}

type SpoofRequest struct {
	Issuer string
	Destination  string
	Receiver string
}

type SpoofResponse struct {
	Receiver string
	hops []string
}

type VPsRequest struct {
	Destination string
	k int
	method string
}

/*
func (v1 V1Api) getVPs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Debug("testing novel GetVPs")
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)

	var vpr VPsRequest
	err := dec.Decode(&vpr)

	log.Debug(vpr.Destination) // Destination IP
	log.Debug(vpr.k) // k as in "Top k VPs demanded"
	log.Debug(vpr.method) // ranking method: Set Cover or Ingress Cover.
	log.Debug("Done printing request data.")

	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// make the request to DB.
}
*/


type RecordrouteRequest struct {
	Issuer string
	Destination  string
}

type RecordrouteBatchRequest struct {
	Destinations []string
}

type RecordrouteResponse struct {
	hops []string `json:"hops"`
}

func PrintRankedVPs(client rs.RankingClient, req *rs.GetVPsReq) {
	log.Debug("Testing client request.")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Debug("Calling GetVPs to server.")
	resp, err := client.GetVPs(ctx, req)

	if err != nil {
		log.Error(err)
	}
	log.Debug(resp)
	log.Debug(err)
}


func (v1 V1Api) testRankingServiceCall(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Debug("Testing rankingservice call.")


	// below code is to automate finding serverAddr.
	// Inside Docker, it's not docker0. <-- fix this.
	/*
	ief, nerr := net.InterfaceByName("docker0") //
	if nerr != nil {
		log.Error(nerr)
	}
	addrs, nerr := ief.Addrs()
	if nerr != nil {
		log.Error(nerr)
	}
	serverAddr := ""
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip.To4() != nil {
			log.Debug("Found docker0 interface IP:", ip)
			serverAddr += ip.String()
		}
	}
	serverAddr += ":49491" // TODO: this is the port used by rankingservice, change to not be hardcoded.

	 */
	serverAddr := "172.17.0.1:49491" // TODO: docker0 ip COULD change, so implement above automation correctly.
	log.Debug("serverAddr=", serverAddr)

	log.Debug("Building opts.")
	var opts []grpc.DialOption
	/*
		if *tls {
			if *caFile == "" {
				*caFile = testdata.Path("ca.pem")
			}
			creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
			if err != nil {
				log.Fatalf("Failed to create TLS credentials %v", err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}
	*/

	opts = append(opts, grpc.WithInsecure())
	// opts = append(opts, grpc.WithBlock())

	log.Debug("Dialling.")
	conn, ers := grpc.Dial(serverAddr, opts...)

	log.Debug("Done dialling.")
	if ers != nil {
		log.Error("fail to dial: %v", ers)
	}

	log.Debug("Defer conn.Close().")
	defer conn.Close()

	log.Debug("Creating new client.")
	client := rs.NewRankingClient(conn)

	log.Debug("Sending request to server")
	PrintRankedVPs(client, &rs.GetVPsReq{
		Ip:                   "104.245.114.100",
		NumVPs:               3,
		RankingTechnique:     "TK",
		ExecutedMeasurements: []string{"1495083626",},
	})
}

func (v1 V1Api) single_recordroute(w http.ResponseWriter, r *http.Request){
	log.Debug("test single rr!")
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Debug("test single rr!")

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)

	var rr RecordrouteRequest
	err := dec.Decode(&rr)

	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// print spoofrequest
	// hopss, err := v1.s.RunRegularRR(rr.Issuer, rr.Destination) // vp should be equal to recv

	var isss []string
	var dsts []string

	isss = append(isss, rr.Issuer)
	dsts = append(dsts, rr.Destination)

	// hopss, err := v1.s.RunBatchRegularRR(isss, dsts)

	// rrResponse := RecordrouteResponse{
	// 	hops:     hopss[0],
	// }
	// log.Debug("end of test, got hops:", hopss, " destination: ", rr.Destination, " AND rrResponse: ", rrResponse)


	// w.Header().Set("Content-Type", "application/json")
	// err = json.NewEncoder(w).Encode(rrResponse)
	// if err != nil {
	// 	log.Error(err)
	// 	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	// 	return
	// }
}

func (v1 V1Api) recordroute(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Debug("test rr!")


	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)

	var rrr RecordrouteBatchRequest
	err := dec.Decode(&rrr)

	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// print spoofrequest
	//hopss, err := v1.s.RunRegularRR(rrr.Issuer, rrr.Destination) // vp should be equal to recv

	key := r.Header.Get(keyHeader)
	pbr := &pb.GetSourcesReq{
		Auth: key,
	}
	resp, err := v1.s.GetSources(pbr)
	if err == repo.ErrNoRevtrUserFound {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var isss []string
	for _, isr := range resp.Srcs {
		log.Debug("adding VP, Hostname:", isr.Hostname, " IP:", isr.Ip, " Site:", isr.Site)
		isss = append(isss, isr.Ip)
	}

	// hopss, err := v1.s.RunBatchRegularRR(isss, rrr.Destinations)

	// rrResponse := RecordrouteResponse{
	// 	hops:     hopss[0],
	// }
	// log.Debug("end of test, got hops:", hopss, " destinations: ", rrr.Destinations, " AND rrResponse: ", rrResponse)


	// w.Header().Set("Content-Type", "application/json")
	// err = json.NewEncoder(w).Encode(rrResponse)
	// if err != nil {
	// 	log.Error(err)
	// 	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	// 	return
	// }
}

func (v1 V1Api) spoof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Debug("test spoof!")
	// key := r.Header.Get(keyHeader) TODO: Add authentication step.

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	//dec.DisallowUnknownFields()

	var sr SpoofRequest
	err := dec.Decode(&sr)

	log.Debug(sr.Issuer)
	log.Debug("Done printing issuer")
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// print spoofrequest
	// hops, vp, err := v1.s.RunSpoofedRR(sr.Issuer, sr.Destination, sr.Receiver) // vp should be equal to recv
	// spoofResponse := SpoofResponse{
	// 	Receiver: vp,
	// 	hops:     hops,
	// }

	// js, err := json.Marshal(spoofResponse)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.Write(js)
}

func (v1 V1Api) cleanAtlas(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	source := req.Header.Get("source")
	pbr := &pb.CleanAtlasReq{
		Auth: key,
		Source: source,
	}

	resp, err := v1.s.CleanAtlas(pbr)
	if err == repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	r.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(r).Encode(resp)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (v1 V1Api) runAtlas(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	source := req.Header.Get("source")
	pbr := &pb.RunAtlasReq{
		Auth: key,
		Source: source,
	}

	resp, err := v1.s.RunAtlas(pbr)
	if err == repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	r.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(r).Encode(resp)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (v1 V1Api) sources(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	pbr := &pb.GetSourcesReq{
		Auth: key,
	}

	resp, err := v1.s.GetSources(pbr)
	if err == repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	r.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(r).Encode(resp)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (v1 V1Api) revtr(r http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		v1.submitRevtr(r, req)
	case http.MethodGet:
		v1.retreiveRevtr(r, req)
	default:
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (v1 V1Api) retreiveRevtr(r http.ResponseWriter, req *http.Request) {
	key := req.Header.Get(keyHeader)
	ids := req.URL.Query().Get("batchid")
	if len(ids) == 0 {
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(ids, 10, 32)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	grr := &pb.GetRevtrReq{
		BatchId: uint32(id),
		Auth:    key,
	}
	revtrs, err := v1.s.GetRevtr(grr)
	if err == repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	r.Header().Set("Content-Type", "application/json")
	var m jsonpb.Marshaler
	err = m.Marshal(r, revtrs)
	if err != nil {
		log.Debug(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
		return
	}
}

func (v1 V1Api) submitRevtr(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	var revtr pb.RunRevtrReq
	err := jsonpb.Unmarshal(req.Body, &revtr)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	revtr.Auth = key
	resp, err := v1.s.RunRevtr(&revtr)
	if err != nil {
		log.Error(err)
		switch err.(type) {
		case server.SrcError, server.DstError:
			http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		default:
			http.Error(r, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	r.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(r).Encode(struct {
		ResultURI string `json:"result_uri"`
	}{
		ResultURI: fmt.Sprintf("https://%s%s?batchid=%d", "revtr.ccs.neu.edu", v1Prefix+"revtr", resp.BatchId),
	})
}
