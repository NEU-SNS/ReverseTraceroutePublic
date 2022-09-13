package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	ca "github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	httputil "github.com/NEU-SNS/ReverseTraceroute/httputils"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	repo "github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/runner"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/v1api"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/v2api"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus"
)

var (
	templates = &template.Template{}
)

// AppConfig for the app
type AppConfig struct {
	ServerConfig types.Config
	DB           repo.Configs
	CacheConfig  ca.Config
}

func init() {
	config.SetEnvPrefix("REVTR")
	if environment.IsDebugRevtr {
		config.AddConfigPath("./revtr/revtr.config")
		templates = template.Must(template.ParseGlob("revtr/webroot/templates/*.html"))
	} else {
		config.AddConfigPath("/revtr/revtr.config") // In docker
		templates = template.Must(template.ParseGlob("/revtr/webroot/templates/*.html"))
	}
	
	

	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		switch {
		case err != nil:
			return false, false
		case host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "rhansen2.local" || host == "rhansen2.revtr.ccs.neu.edu" || host == "129.10.113.189":
			return true, true
		default:
			return false, false
		}
	}
	grpclog.SetLogger(log.GetLogger())
}

func main() {

	apiPort := 8080
	// grpcPort := environment.RevtrPortProduction

	if environment.IsDebugRevtr {
		apiPort = environment.RevtrAPIPortDebug
	}

	conf := AppConfig{
		ServerConfig: types.NewConfig(),
	}
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Fatal(err)
	}
	var repoOpts []repo.Option
	
	for _, c := range conf.DB.WriteConfigs {
		repoOpts = append(repoOpts, repo.WithWriteConfig(c))
	}
	for _, c := range conf.DB.ReadConfigs {
		repoOpts = append(repoOpts, repo.WithReadConfig(c))
	}
	repoOpts = append(repoOpts, repo.WithEnvironment(conf.DB.Environment))

	da, err := repo.NewRepo(repoOpts...)
	if err != nil {
		log.Fatal(err)
	}

	grpcOptions := grpc.WithInsecure()
	connvp := ""
	if !environment.IsDebugVPService {
		_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatal(err)
		}
		vpcreds, err := credentials.NewClientTLSFromFile(*conf.ServerConfig.RootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
		}
		connvp = fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		grpcOptions = grpc.WithTransportCredentials(vpcreds)
	} else {
		connvp = fmt.Sprintf("%s:%d", "localhost", environment.VPServicePortDebug)
	}
	c3, err := grpc.Dial(connvp, grpcOptions)
	if err != nil {
		log.Fatal(err)
	}
	vps := vpservice.New(context.Background(), c3)

	tlsConf, err := httputil.TLSConfig(*conf.ServerConfig.CertFile, *conf.ServerConfig.KeyFile)
	if err != nil {
		log.Fatal(err)
	}

	serv := server.NewRevtrServer(server.WithVPSource(vps),
		server.WithAdjacencySource(da),
		server.WithClusterSource(da),
		server.WithRTStore(da),
		server.WithRootCA(*conf.ServerConfig.RootCA),
		server.WithCertFile(*conf.ServerConfig.CertFile),
		server.WithKeyFile(*conf.ServerConfig.KeyFile),
		// server.WithCache(cache.New(time.Minute*30, time.Minute*30)),
		server.WithCache(cache.New(time.Duration(*conf.CacheConfig.CacheTime)*time.Minute, 
		time.Duration(*conf.CacheConfig.CacheTime)*time.Minute)),
		server.WithCacheTime(*conf.CacheConfig.CacheTime),
		server.WithRunner(runner.New()),
	)

	mux := http.NewServeMux()
	RegisterHome(vps, *conf.ServerConfig.WebsiteSourceFile, mux)
	RegisterRunRevtr(serv, mux)
	mux.Handle("/styles/", http.StripPrefix("/styles", http.FileServer(http.Dir("webroot/style"))))
	v1api.NewV1Api(serv, mux)
	v2serv := v2api.CreateServer(serv, tlsConf)
	gatewayMux := runtime.NewServeMux()
	selfCreds := credentials.NewClientTLSFromCert(nil, "www.revtr.ccs.neu.edu")
	dialOpts := []grpc.DialOption{}
	if environment.IsDebugRevtr{
		dialOpts = []grpc.DialOption{grpc.WithInsecure()}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(selfCreds)}
	}
	err = pb.RegisterRevtrHandlerFromEndpoint(context.Background(), gatewayMux, ":"+strconv.Itoa(apiPort), dialOpts)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
	s := &http.Server{
		Addr:      fmt.Sprintf(":%d", apiPort),
		TLSConfig: tlsConf,
		Handler:   directRequest(v2serv, mux),
	}

	http.Handle("/metrics", prometheus.Handler())
	go func() {
		for {
			log.Error(http.ListenAndServe(":45454", nil))
			// log.Error(http.ListenAndServe(":8888", nil))
		}
	}()

	go func() {
		for {
			log.Error(http.ListenAndServe(":8181", http.HandlerFunc(redirect)))
		}
	}()
	
	log.Infof("Starting http server on port %d\n\n\n", apiPort)
	conn, _ := net.Listen("tcp4", ":"+strconv.Itoa(apiPort))
	go func() {
		for {
			log.Error(s.Serve(tls.NewListener(conn, tlsConf)))
		}
	} ()
	
	grpcPort := environment.RevtrGRPCPortProduction
	if environment.IsDebugRevtr {
		grpcPort = environment.RevtrGRPCPortDebug
	}
	grpcConn, err := net.Listen("tcp", ":" + strconv.Itoa(grpcPort))
	if err != nil {
		log.Error(err)
		panic(err)
	}
	log.Infof("Starting grpc server on port %d", grpcPort)
	v2serv.Serve(grpcConn)
	
	// go profiler.SigHandle()
	// f, err := profiler.CPUProfile("revtr_cpu_profile")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	// defer pprof.StopCPUProfile()
	// go func() {
	// 	for {
	// 		log.Error(http.ListenAndServe(":8181", http.HandlerFunc(redirect)))
	// 	}
	// }()
	// go func() {
	// 	if environment.IsDebugRevtr{
	// 		for {
	// 			log.Error(v2serv.Serve(conn))
	// 		}
	// 	} else {
	// 		for {
	// 			log.Error(v2serv.Serve(tls.NewListener(conn, tlsConf)))
	// 		}
	// 	}
	// } ()
	// if environment.IsDebugRevtr{
	// 	conn, _ := net.Listen("tcp", ":"+strconv.Itoa(apiPort))
		
	// 	// s := &http.Server{
	// 	// 	Addr:      ":" + strconv.Itoa(apiPort),
	// 	// 	Handler:   directRequest(v2serv, mux),
	// 	// }
	// 	for {
	// 		log.Error(v2serv.Serve(conn))
	// 	}
		
	// } else {
	// 	conn, err := net.Listen("tcp", ":"+strconv.Itoa(apiPort))
	// 	s := &http.Server{
	// 		Addr:      ":" + strconv.Itoa(apiPort),
	// 		TLSConfig: tlsConf,
	// 		Handler:   directRequest(v2serv, mux),
	// 	}
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	for {
	// 		log.Error(s.Serve(tls.NewListener(conn, tlsConf)))
	// 	}
	// }
	
}

func directRequest(grpcServer *grpc.Server, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func redirect(w http.ResponseWriter, req *http.Request) {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		host = req.Host
	}
	http.Redirect(w, req, "https://"+host+":443"+req.RequestURI, http.StatusMovedPermanently)
}

type homeModel struct {
	Nodes []vpModel
}

type vpModel struct {
	Host string
	IP   string
}

type vpModelSort []vpModel

func (a vpModelSort) Len() int           { return len(a) }
func (a vpModelSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a vpModelSort) Less(i, j int) bool { return a[i].Host < a[j].Host }

// Home handles the home route for revtr
type Home struct {
	vps vpservice.VPSource
	vpsWithAtlasFile string 
}

// RegisterHome wires up a new home handler
func RegisterHome(vps vpservice.VPSource, vpsWithAtlasFile string, mux *http.ServeMux) {
	h := Home{
		vps: vps,
		vpsWithAtlasFile: vpsWithAtlasFile,
	}
	mux.HandleFunc("/", h.Home)
}

// Home handles the "/" route
func (h Home) Home(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// Only provides VPs where there is an atlas
	vpsWithAtlasSet := map[string] struct{} {}

	vpsWithAtlas := util.IPsFromFile(h.vpsWithAtlasFile)
	for _, vp := range(vpsWithAtlas) {
		vpsWithAtlasSet[vp] = struct{}{}
	}

	vps, err := h.vps.GetVPs()
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
		return
	}
	var model homeModel
	nodes := vps.GetVps()
	sites := make(map[string]bool)
	var vpl []vpModel
	for _, node := range nodes {
		ipS, _ := util.Int32ToIPString(node.Ip)
		if _, ok := vpsWithAtlasSet[ipS]; !ok {
			continue
		}

		if !node.RecordRoute || !node.Timestamp {
			continue
		}

		if strings.Contains(node.Hostname, "staging") {
			continue
		}

		if sites[node.Site] {
			continue
		}
		sites[node.Site] = true
		var vp vpModel
		vp.Host = node.Hostname
		vp.IP, err = util.Int32ToIPString(node.Ip)
		if err != nil {
			log.Error(err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
			return
		}
		vpl = append(vpl, vp)
	}
	sort.Sort(vpModelSort(vpl))
	model.Nodes = vpl
	templates.ExecuteTemplate(rw, "home", &model)
}

type runningModel struct {
	Key string
	URL string
}

type msgFunc func(msg server.Status) error
type completeFunc func()

type outputChan struct {
	c <-chan server.Status
	// protects the following
	mu        sync.Mutex
	last      server.Status
	funcs     map[*msgFunc]bool
	compl     completeFunc
	completed bool
}

func (oc *outputChan) complete() {
	if oc.completed {
		// Already done
		return
	}
	oc.completed = true
	if oc.compl != nil {
		oc.compl()
	}
}

func (oc *outputChan) onComplete(cf completeFunc) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.compl = cf
}

func (oc *outputChan) register(mf msgFunc) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if oc.funcs == nil {
		oc.funcs = make(map[*msgFunc]bool)
	}
	oc.funcs[&mf] = true
	zero := server.Status{}
	if oc.last != zero {
		oc.callAll(oc.last)
	}
}

func (oc *outputChan) remove(mf msgFunc) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if oc.funcs == nil {
		return
	}
	delete(oc.funcs, &mf)
}

func (oc *outputChan) callAll(s server.Status) {
	for f := range oc.funcs {
		err := (*f)(s)
		if err != nil {
			oc.removeLocked(f)
		}
	}
}

// call in a goroutine
func (oc *outputChan) monitor() {
	for {
		select {
		case s, ok := <-oc.c:
			if !ok {
				return
			}
			oc.mu.Lock()
			oc.callAll(s)
			oc.last = s
			oc.complete()
			oc.mu.Unlock()
		}
	}
}

func (oc *outputChan) removeLocked(mf *msgFunc) {
	delete(oc.funcs, mf)
}

// RunRevtr handles /runrevtr
type RunRevtr struct {
	s server.RevtrServer
	// protects map of id -> Status chan
	mu  *sync.Mutex
	rts map[uint32]*outputChan
}

// RegisterRunRevtr wires up a new runrevtr
func RegisterRunRevtr(s server.RevtrServer, mux *http.ServeMux) {
	rr := RunRevtr{
		s:   s,
		mu:  &sync.Mutex{},
		rts: make(map[uint32]*outputChan),
	}
	mux.HandleFunc("/ws", rr.WS)
	mux.HandleFunc("/runrevtr", rr.RunRevtr)
}

type wsConnection struct {
	c *websocket.Conn
}

func (ws wsConnection) Close() error {
	if ws.c == nil {
		return nil
	}
	return ws.c.Close()
}

type wsMessage struct {
	HTML   string
	Status bool
	Error  string
}

func (ws wsConnection) Write(mess wsMessage) error {
	if ws.c == nil {
		return nil
	}
	res, err := json.Marshal(&mess)
	if err != nil {
		return err
	}
	err = ws.c.SetWriteDeadline(time.Now().Add(time.Second))
	if err != nil {
		return err
	}
	return ws.c.WriteMessage(websocket.TextMessage, res)

}

// WS is the endpoint for websockets
func (rr RunRevtr) WS(rw http.ResponseWriter, req *http.Request) {
	var upgrader websocket.Upgrader
	// There were issues with the origin header not passing the default check
	// so check after putting this behind the proxy
	// So the proxy now passes through the Origin and we check that its what we expect
	upgrader.CheckOrigin = func(r *http.Request) bool {
		orig := req.Header.Get("Origin")
		if len(orig) == 0 {
			return false
		}
		u, err := url.Parse(orig)
		if err != nil {
			log.Error(err)
			return false
		}
		return u.Host == "www.revtr.ccs.neu.edu" || u.Host == "revtr.ccs.neu.edu"
	}
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	key := req.URL.Query().Get("key")
	log.Debug("WS request for key: ", key)
	if key == "" {
		defer ws.Close()
		err = ws.WriteMessage(websocket.TextMessage, []byte("Missing key."))
		if err != nil {
			log.Error(err)
		}
		return
	}
	c := wsConnection{c: ws}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	keyint, err := strconv.ParseUint(key, 10, 32)
	if err != nil {
		log.Error(err)
		err = ws.WriteMessage(websocket.TextMessage, []byte("Invalid key."))
		if err != nil {
			log.Error(err)
		}
		return
	}
	rtrs, ok := rr.rts[uint32(keyint)]
	if !ok {
		defer ws.Close()
		log.Errorf("Invalid Key: %d", keyint)
		err = ws.WriteMessage(websocket.TextMessage, []byte("Invalid key."))
		if err != nil {
			log.Error(err)
		}
		return
	}
	var oc *outputChan
	if rtrs == nil {
		out, err := rr.s.StartRevtr(context.Background(), uint32(keyint))
		if err != nil {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		oc = &outputChan{c: out}
		// When the reverse traceroute is done, remove it from the map
		oc.onComplete(func() {
			rr.mu.Lock()
			defer rr.mu.Unlock()
			delete(rr.rts, uint32(keyint))
		})
		rr.rts[uint32(keyint)] = oc
	} else {
		oc = rtrs
	}
	mf := msgFunc(func(s server.Status) error {
		err := c.Write(wsMessage{
			HTML:   s.Rep,
			Status: s.Status,
			Error:  s.Error,
		})
		if err != nil {
			log.Error(err)
			c.Close()
			return err
		}
		return nil
	})
	oc.register(mf)
	go oc.monitor()
}

// RunRevtr handles /runrevtr
func (rr RunRevtr) RunRevtr(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	src := req.FormValue("src")
	dst := req.FormValue("dst")
	if src == "" || dst == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Errorf("bad request, src: %s, dst: %s", src, dst)
		return
	}
	rm := pb.RevtrMeasurement{
		Src:       src,
		Dst:       dst,

		Staleness: 180,
	}
	id, err := rr.s.AddRevtr(rm)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var rt runningModel
	rt.Key = fmt.Sprintf("%d", id)
	rt.URL = "revtr.ccs.neu.edu"
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.rts[id] = nil
	templates.ExecuteTemplate(rw, "running", &rt)
	return
}
