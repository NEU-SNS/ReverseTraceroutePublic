package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"google.golang.org/grpc"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "172.18.0.1:49491", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by the TLS handshake")
)

func PrintRankedVPs(client pb.RakingClient, req *pb.GetVPsReq) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetVPs(ctx, req)
	if err != nil {
		log.Fatalf("%s.GetVPs(_) = _, %s", client, err)
	}
	log.Println(resp)
}

func main() {
	flag.Parse()

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
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)

	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	defer conn.Close()
	client := pb.NewRakingClient(conn)

	PrintRankedVPs(client, &pb.GetVPsReq{
		Ip:                   "104.245.114.100",
		NumVPs:               3,
		RankingTechnique:     "TK",
		ExecutedMeasurements: []string{},
	})

}
