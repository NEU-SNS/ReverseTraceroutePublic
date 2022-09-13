package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	httputil "github.com/NEU-SNS/ReverseTraceroute/httputils"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/api"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/filters"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/httpapi"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/repo"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
	"github.com/prometheus/client_golang/prometheus"
)

// AppConfig is the config struct for the vpservice
type AppConfig struct {
	ServerConfig types.Config
	DB           repo.Configs
}

const (
	// CONFIG_PATH = "./vpservice/vpservice.config"
	CONFIG_PATH = "/vpservice/vpservice.config" // In docker
)

func init() {
	config.SetEnvPrefix("VPS")
	// if environment.IsDebugVPService {
	config.AddConfigPath(CONFIG_PATH)

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

type errorf func() error

func logError(f errorf) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

func main() {
	apiPort := 0
	if CONFIG_PATH == "./vpservice/vpservice.config" {
		apiPort = 58585
	} else {
		apiPort = 8080
	}
	
	grpcPort := environment.VPServicePortProduction

	if environment.IsDebugVPService {
		apiPort = environment.VPServiceAPIPortDebug
		grpcPort = environment.VPServicePortDebug
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
	da, err := repo.NewRepo(repoOpts...)
	if err != nil {
		log.Fatal(err)
	}

	grpcOptions := grpc.WithInsecure()
	connst := ""
	
	if !environment.IsDebugController {
		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatal(err)
		}
		ccreds, err := credentials.NewClientTLSFromFile(*conf.ServerConfig.RootCA, srvs[0].Target)
		if err != nil {
			log.Fatalf("%s, %s", err, *conf.ServerConfig.RootCA)
		}
		grpcOptions = grpc.WithTransportCredentials(ccreds)
		connst = fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	} else {
		connst = fmt.Sprintf("%s:%d", "localhost", environment.ControllerPortDebug)
	}

	c, err := grpc.Dial(connst, grpcOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer logError(c.Close)

	cl := client.New(context.Background(), c)
	rrf, tsf := makeFilters()
	s, err := server.NewServer(server.WithVPProvider(da),
		server.WithClient(cl),
		server.WithRRFilter(rrf),
		server.WithTSFilter(tsf))

	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	// Register pprof
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	httpapi.NewAPI(s, mux)
	go func() {
		for {
			log.Error(http.ListenAndServe(":"+strconv.Itoa(apiPort), mux))
		}
	}()
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(grpcPort))
	if err != nil {
		log.Fatal(err)
	}
	defer logError(ln.Close)
	tlsc, err := httputil.TLSConfig(*conf.ServerConfig.CertFile, *conf.ServerConfig.KeyFile)
	if err != nil {
		log.Error(err)
		log.Error(conf)
		if err := ln.Close(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}
	apiServ := api.CreateServer(s, tlsc)
	err = apiServ.Serve(ln)
	if err != nil {
		log.Fatal(err)
	}
}

func makeFilters() (filters.RRFilter, filters.TSFilter) {
	rrf := filters.ComposeRRFilter(filters.MakeRRDistanceFilter(9, 9),
		filters.OnePerSiteRR,
		filters.OrderRRDistanceFilter)
	return rrf, filters.OnePerSiteTS
}
