package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/api"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	httputil "github.com/NEU-SNS/ReverseTraceroute/httputils"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	bgpradix "github.com/NEU-SNS/ReverseTraceroute/radix"
	rkclient "github.com/NEU-SNS/ReverseTraceroute/rankingservice/client"
	vpsclient "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	config.SetEnvPrefix("ATLAS")
	if environment.IsDebugAtlas {
		config.AddConfigPath("./atlas/atlas.config")

	} else {
		config.AddConfigPath("/atlas/atlas.config") // In docker
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

type errorf func() error

func logError(f errorf) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

func main() {

	apiPort := 8080
	grpcPort := environment.AtlasPortProduction

	if environment.IsDebugAtlas {
		apiPort = environment.AtlasAPIPortDebug
		grpcPort = environment.AtlasPortDebug
	}

	conf := server.Config{}
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
	r, err := repo.NewRepo(repoOpts...)
	if err != nil {
		log.Fatal(err)
	}
	vps, err := makeVPS(conf.RootCA)
	if err != nil {
		log.Fatal(err)
	}
	cc, err := makeClient(conf.RootCA)
	if err != nil {
		log.Fatal(err)
	}
	rkcl, err := makeRankingClient(conf.RootCA)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/metrics", prometheus.Handler())
	go func() {
		for {
			log.Error(http.ListenAndServe(":"+strconv.Itoa(apiPort), nil))
		}
	}()
	cache, err := lru.New(1000000)
	if err != nil {
		panic("Could not create cache " + err.Error())
	}

	rt := bgpradix.New(conf.IP2AS)
	serv := server.NewServer(conf, server.WithVPS(vps),
		server.WithTRS(r),
		server.WithClient(cc),
		server.WithCache(cache),
		server.WithRadix(*rt),
		server.WithRanking(rkcl))
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(grpcPort))
	if err != nil {
		log.Fatal(err)
	}
	defer logError(ln.Close)
	tlsc, err := httputil.TLSConfig(conf.CertFile, conf.KeyFile)
	s := api.CreateServer(serv, tlsc)
	err = s.Serve(ln)
	if err != nil {
		log.Fatal(err)
	}
}

func makeVPS(rootCA string) (vpsclient.VPSource, error) {

	grpcOptions := grpc.WithInsecure()
	conn := ""
	if !environment.IsDebugVPService {
		_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatal(err)
		}
		vpcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
		}
		conn = fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		grpcOptions = grpc.WithTransportCredentials(vpcreds)
	} else {
		conn = fmt.Sprintf("%s:%d", "localhost", environment.VPServicePortDebug)
	}
	c, err := grpc.Dial(conn, grpcOptions)
	if err != nil {
		log.Fatal(err)
	}

	return vpsclient.New(context.Background(), c), nil
}

func makeClient(rootCA string) (cclient.Client, error) {

	grpcOptions := grpc.WithInsecure()
	connst := ""
	if !environment.IsDebugController {
		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatal(err)
		}
		ccreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
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
	return cclient.New(context.Background(), c), nil
}

func makeRankingClient(rootCA string) (rkclient.RankingSource, error){
	connrs := ""
	insecureOptionGrpc := grpc.WithInsecure()
	grpcDialOptions := []grpc.DialOption{}
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(1024 * 1024 * 100))) 
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 100)))
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
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(rscreds))
		c4, err := grpc.Dial(connrs, grpcDialOptions...)
		if err != nil {
			return nil, err
		} 
		rks := rkclient.New(context.Background(), c4)
		// ret.rks = rks
		return rks, nil

	} else {
		connrs = fmt.Sprintf("%s:%d", "localhost", environment.RankingServicePortDebug)
		grpcDialOptions = append(grpcDialOptions, insecureOptionGrpc)
		c4, err := grpc.Dial(connrs, grpcDialOptions...)
		if err != nil {
			return nil, err
		}
		rks := rkclient.New(context.Background(), c4)
		return rks, nil
	}
}
