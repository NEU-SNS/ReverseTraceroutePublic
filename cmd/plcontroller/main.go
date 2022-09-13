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
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"golang.org/x/net/trace"

	atpb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/plcontroller"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpspb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/watcher"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	defaultConfig = "/plcontroller/plcontroller.config" // In docker
	// defaultConfig = "plcontroller/plcontroller.config" // Local
	// ConfigPath is the path to the configuration file
	ConfigPath string
	// Build is the build number
	Build string
	// Version is the version number
	Version     string
	showVersion bool
)

var conf = plcontroller.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	if environment.IsDebugPLController{
		ConfigPath = "plcontroller/plcontroller.config" 
	}
	if ConfigPath == "" {
		config.AddConfigPath(defaultConfig)
	} else {
		config.AddConfigPath(ConfigPath)
	}

	flag.Int64Var(conf.Local.Timeout, "t", 60,
		"The default timeout used for measurement requests.")
	flag.StringVar(conf.Local.Addr, "a", "0.0.0.0",
		"The address that the controller will bind to.")

	port := environment.PLControllerPortProduction
	scamperPort :=environment.ScamperPortProduction
	if environment.IsDebugPLController{
		port = environment.PLControllerPortDebug
		scamperPort = environment.ScamperPortDebug
	}


	flag.StringVar(conf.PProf.CPUProfileFile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(conf.PProf.MemoryProfileFile, "memprofile","", "write memory profile to `file`")

	flag.IntVar(conf.Local.Port, "p", port,
		"The port that the controller will use.")
	flag.BoolVar(conf.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed.")
	flag.StringVar(conf.Scamper.Port, "scamper-port", strconv.Itoa(scamperPort),
		"Port that Scamper will use.")
	flag.StringVar(conf.Scamper.SockDir, "socket-dir", "/tmp/scamper_sockets",
		"Directory that scamper will use for its sockets")
	flag.StringVar(conf.Scamper.BinPath, "scamper-bin", "/usr/local/bin/sc_remoted",
		"Path to the scamper binary")
	flag.StringVar(conf.Scamper.ZombieTime, "zombie-time", "2",
	"Path to the scamper binary")
	flag.StringVar(conf.Scamper.ConverterPath, "converter-path", "/usr/local/bin/sc_warts2json",
		"Path for warts parser")
	flag.StringVar(conf.Local.PProfAddr, "pprof", ":55556",
		"The port for pprof")
	flag.StringVar(conf.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.StringVar(conf.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
	flag.StringVar(conf.Local.SSHKeyPath, "sshkey-path", "",
		"The path to the key for connecting to planet-lab")
	flag.StringVar(conf.Local.PLUName, "pluname", "",
		"The username to use for logging into planet-lab nodes")
	flag.StringVar(conf.Db.UName, "db-uname", "",
		"The username for the database")
	flag.StringVar(conf.Db.Password, "db-pass", "",
		"The password for the database")
	flag.StringVar(conf.Db.Db, "db-name", "",
		"The name of the database to use")
	flag.StringVar(conf.Db.Host, "db-host", "localhost",
		"The host of the database")
	flag.StringVar(conf.Db.Port, "db-port", "3306",
		"The port used for the database connection")
	flag.StringVar(conf.Local.UpdateURL, "update-url",
		"http://www.ccs.neu.edu/home/rhansen2/plvp.json",
		"The path for the version info of the plvps")
	flag.BoolVar(&showVersion, "version",
		false, "Show version info")
	flag.StringVar(conf.Local.CControllerAddr, "ccontroller-addr", 
	"localhost", "The address of the central controller")
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
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Fatalf("Failed to parse config: %v\n", err)
	}
	if showVersion {
		fmt.Printf("Build: %s\nVersion: %s\n", Build, Version)
		return
	}

	util.CloseStdFiles(*conf.Local.CloseStdDesc)
	var sc scamper.Config
	if environment.IsDebugPLController {
		*conf.Scamper.Port = strconv.Itoa(environment.ScamperPortDebug)
		*conf.Scamper.SockDir = "/tmp/scamper_sockets_debug"
	}
	
	sc.Port = *conf.Scamper.Port
	sc.Path = *conf.Scamper.SockDir
	sc.ScPath = *conf.Scamper.BinPath
	sc.ScParserPath = *conf.Scamper.ConverterPath
	sc.ZombieTime = *conf.Scamper.ZombieTime
	err = scamper.ParseConfig(sc)
	if err != nil {
		log.Fatalf("Invalid scamper configuration: %v\n", err)
	}
	proc := scamper.GetProc(sc.Path, sc.Port, sc.ScPath, sc.ZombieTime)
	// proc := scamper.GetProc(sc.Path, "4299", sc.ScPath, sc.ZombieTime)
	mp := mproc.New()
	_, err = mp.ManageProcess(proc, true, 1000)
	if err != nil {
		log.Fatalf("Could not start scamper: %v\n", err)
	}

	// scamperNewPath := "/scamper-cvs-20211212x/utils/sc_remoted/sc_remoted"
	// proc = scamper.GetProc(sc.Path, "4299", scamperNewPath, sc.ZombieTime)
	// mp = mproc.New()
	// _, err = mp.ManageProcess(proc, true, 1000)
	// if err != nil {
	// 	log.Fatalf("Could not start scamper: %v\n", err)
	// }

	db, err := da.New(da.DbConfig{
		WriteConfigs: []da.Config{
			da.Config{
				User:     *conf.Db.UName,
				Password: *conf.Db.Password,
				Host:     *conf.Db.Host,
				Port:     *conf.Db.Port,
				Db:       *conf.Db.Db,
			},
		},
		ReadConfigs: []da.Config{
			da.Config{
				User:     *conf.Db.UName,
				Password: *conf.Db.Password,
				Host:     *conf.Db.Host,
				Port:     *conf.Db.Port,
				Db:       *conf.Db.Db,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create db: %v\n", err)
	}
	fw, err := watcher.New(*conf.Scamper.SockDir)
	if err != nil {
		log.Fatalf("Failed to created file watcher: %v\n", err)
	}

	// Initiate the connection to VP service
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Fatalf("Failed to connect to vp service: %s", err)
	}
	vpcreds, err := credentials.NewClientTLSFromFile(*conf.Local.RootCA, srvs[0].Target)
	if err != nil {
		log.Fatalf("Failed to connect to vp service: %s", err)
	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		log.Fatalf("Failed to connect to vp service: %s", err)
	}
	vpscl := vpspb.NewVPServiceClient(c)

	// Initiate the connection to Atlas service
	var atcl atpb.AtlasClient
	if !environment.IsDebugAtlas{
		_, srvs, err := net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Fatalf(err.Error())
		}
		atcreds, err := credentials.NewClientTLSFromFile(*conf.Local.RootCA, srvs[0].Target)
		if err != nil {
			log.Fatalf(err.Error())
		}
		connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c, err = grpc.Dial(connstrat, grpc.WithTransportCredentials(atcreds))
		if err != nil {
			log.Fatalf(err.Error())
		}
		atcl = atpb.NewAtlasClient(c)
		
	} else {
		// Atlas 
		insecureOptionGrpc := grpc.WithInsecure()
		connstrat := fmt.Sprintf("%s:%d", "localhost", environment.AtlasPortDebug) // 55000 is the default
		c2, err := grpc.Dial(connstrat, insecureOptionGrpc)
		if err != nil {
			log.Fatalf(err.Error())
		}
		atcl = atpb.NewAtlasClient(c2)
	}


	plc, err := plcontroller.New(plcontroller.WithConfig(conf), plcontroller.WithVPStore(db),
	plcontroller.WithClient(scamper.NewClient()),
	plcontroller.WithAtlasClient(atcl), 
	plcontroller.WithVPserviceClient(vpscl),
	plcontroller.WithWatcher(fw))
	if err != nil {
		log.Fatalf("Failed to create plcontroller: %v\n", err)
	}

	if *conf.PProf.CPUProfileFile != "" {
        f, err := os.Create(*conf.PProf.CPUProfileFile)
        if err != nil {
            log.Fatal("could not create CPU profile: ", err)
        }
        defer f.Close() // error handling omitted for example
        if err := pprof.StartCPUProfile(f); err != nil {
            log.Fatal("could not start CPU profile: ", err)
		}
        defer pprof.StopCPUProfile()
	}
	/*********************************/


	if err != nil {
		log.Errorf("Controller Start returned with error: %v", err)
		util.Exit(1)
	}
	http.Handle("/metrics", prometheus.Handler())
	go func() {
		log.Error(http.ListenAndServe(*conf.Local.PProfAddr, nil))
	}()
	var sigHandle = func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
			syscall.SIGQUIT, syscall.SIGSTOP)
		for sig := range c {
			log.Infof("Got signal: %v", sig)
			pprof.StopCPUProfile()
			mp.IntAll()
			fw.Close()
			plc.Stop()
			db.Close()
		}
	}
	go sigHandle()
	err = plc.Start()
	if err != nil {
		log.Error(err)
	}
}
