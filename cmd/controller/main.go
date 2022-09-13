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
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"google.golang.org/grpc/grpclog"

	"golang.org/x/net/trace"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/router"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var conf = controller.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	port := 0
	if environment.IsDebugController {
		config.AddConfigPath("./controller/controller.config")
		port = environment.ControllerPortDebug
	} else {
		config.AddConfigPath("/controller/controller.config") // In docker
		port = environment.ControllerPortProduction
	}
	flag.StringVar(conf.PProf.CPUProfileFile, "cpuprofile", "cpu_controlller.prof", "write cpu profile to `file`")
	flag.StringVar(conf.PProf.MemoryProfileFile, "memprofile","mem.prof", "write memory profile to `file`")

	flag.StringVar(conf.Local.Addr, "a", "0.0.0.0",
		"The address that the controller will bind to.")
	flag.IntVar(conf.Local.Port, "p", port,
		"The port that the controller will use.")
	flag.BoolVar(conf.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed")
	flag.StringVar(conf.Local.PProfAddr, "pprof", ":55555",
		"The port for pprof")
	flag.StringVar(conf.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.StringVar(conf.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
	flag.Int64Var(conf.Local.ConnTimeout, "conn-timeout", 60,
		"How long to wait for an rpc connection to timeout")
	flag.Var(conf.Cache.Addrs, "cache-list",
		"The list of cache servers.")
	flag.IntVar(conf.Cache.CacheTime, "cache-time", 0, "The cache time of the controller")
	
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
	/**
	Profiling
	**/

	

	go sigHandle()
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Errorf("Failed to parse config: %v", err)
		util.Exit(1)
	}
	util.CloseStdFiles(*conf.Local.CloseStdDesc)
	db, err := da.New(conf.Db)
	if err != nil {
		log.Errorf("Failed to create db: %v", err)
		util.Exit(1)
	}

	if environment.IsDebugController{
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
	}
	/*********************************/

	err = <-controller.Start(conf, db, cache.New(*conf.Cache.Addrs), router.New(*conf.Local.RootCA))

	if err != nil {
		log.Errorf("Controller Start returned with error: %v", err)
		util.Exit(1)
	}

	/**
	Profiling
	**/
	if environment.IsDebugController {
		if *conf.PProf.MemoryProfileFile != "" {
			f, err := os.Create(*conf.PProf.MemoryProfileFile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close() // error handling omitted for example
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		}
	}
	/**********************************/
}


func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		log.Infof("Got signal: %v", sig)
		controller.HandleSig(sig)
		pprof.StopCPUProfile()
		util.Exit(1)
	}
}
