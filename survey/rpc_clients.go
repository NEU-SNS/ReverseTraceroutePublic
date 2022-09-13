package survey

import (
	"crypto/tls"
	"fmt"
	"net"

	atlas "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	vps "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func CreateReverseTracerouteClient(rootCA string) (revtr.RevtrClient, error) {
	grpcDialOptions := []grpc.DialOption{}
	port := environment.RevtrGRPCPortProduction
	if environment.IsDebugRevtr{
		port = environment.RevtrGRPCPortDebug
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
		connst := fmt.Sprintf("%s:%d", "localhost", port)
	// grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(2147483647))) 
	// grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2147483647)))
		c, err := grpc.Dial(connst, grpcDialOptions...)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		client := revtr.NewRevtrClient(c)
		return client, nil 
	} else {
		// _, srvs, err := net.LookupSRV("revtr", "tcp", "revtr.ccs.neu.edu")
		// if err != nil {
		// 	return nil, err
		// }
		// revtrAddr := "www.revtr.ccs.neu.edu"
		// creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		// creds, err := credentials.NewClientTLSFromFile(rootCA, "")
		// certPool := tls.Config.
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,

		})
		// if err != nil {
		// 	log.Fatal(err)
		// 	return nil, err
		// }
		revtrPort := environment.RevtrGRPCPortProduction
		if environment.IsDebugRevtr {
			revtrPort = environment.RevtrGRPCPortDebug
		}
		
		// connStr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		connStr := fmt.Sprintf("%s:%d", "revtr.ccs.neu.edu", revtrPort)
		// grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(creds))
		conn, err := grpc.Dial(connStr, grpcDialOptions...)
		if err != nil {
			return nil, err
		}
		client := revtr.NewRevtrClient(conn)
		return client, nil
	}
}

func CreateVPServiceClient(rootCA string) (vps.VPServiceClient, error) {
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return nil, err
	}
	vpcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		log.Fatal(err)
	}
	connStr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	grpcOptions := grpc.WithTransportCredentials(vpcreds)
	

	conn, err := grpc.Dial(connStr, grpcOptions)
	client := vps.NewVPServiceClient(conn)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CreateRankingClient() (rk.RankingClient, error) {
	grpcDialOptions := []grpc.DialOption{}
	port := environment.RankingServicePortProduction
	if environment.IsDebugRankingService{
		port = environment.RankingServicePortDebug
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
	}
	connst := fmt.Sprintf("%s:%d", "localhost", port)
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(2147483647))) 
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2147483647)))
	c, err := grpc.Dial(connst, grpcDialOptions...)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	client := rk.NewRankingClient(c)
	return client, nil
}
	
func CreateControllerClient(rootCA string) (controller.ControllerClient, error) {
	grpcDialOptions := []grpc.DialOption{}
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(2147483647))) 
	grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2147483647)))
	if environment.IsDebugController{
		port := environment.ControllerPortDebug
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
		connst := fmt.Sprintf("%s:%d", "localhost", port)
		c, err := grpc.Dial(connst, grpcDialOptions...)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		client := controller.NewControllerClient(c)
		return client, nil
	} else {
	
		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			return nil, err
		}
		creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
		}
		connStr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(creds))
		conn, err := grpc.Dial(connStr, grpcDialOptions...)
		client := controller.NewControllerClient(conn)
		return client, nil
	}
}


func CreateAtlasClient(rootCA string) (atlas.AtlasClient, error) {
	grpcDialOptions := []grpc.DialOption{}
	port := environment.AtlasPortProduction
	if environment.IsDebugAtlas{
		port = environment.AtlasPortDebug
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
		connst := fmt.Sprintf("%s:%d", "localhost", port)
		grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(2147483647))) 
		grpcDialOptions = append(grpcDialOptions, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2147483647)))
		c, err := grpc.Dial(connst, grpcDialOptions...)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		client := atlas.NewAtlasClient(c)
		return client, nil
	} else {
		_, srvs, err := net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			return nil, err
		}
		creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
		if err != nil {
			log.Fatal(err)
		}
		connStr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(creds))
		conn, err := grpc.Dial(connStr, grpcDialOptions...)
		client := atlas.NewAtlasClient(conn)
		return client, nil
	}
	
}