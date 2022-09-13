package api

import (
	"crypto/tls"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateServer creates a grpc server for the vpservice api
func CreateServer(s server.VPServer, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{}
	if !environment.IsDebugVPService {
		opts = append(opts, grpc.Creds(credentials.NewTLS(conf)))
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterVPServiceServer(serv, CreateAPI(s))
	return serv
}

type api struct {
	s server.VPServer
}

// CreateAPI returns a pb.VPService that uses the server.VPServer s
func CreateAPI(s server.VPServer) pb.VPServiceServer {
	return api{s: s}
}

func (a api) GetVPs(ctx context.Context, req *pb.VPRequest) (*pb.VPReturn, error) {
	resp, err := a.s.GetVPs(req)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return resp, err
	}
}

func (a api) GetRRSpoofers(ctx context.Context, req *pb.RRSpooferRequest) (*pb.RRSpooferResponse, error) {
	resp, err := a.s.GetRRSpoofers(req)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return resp, err
	}
}

func (a api) GetTSSpoofers(ctx context.Context, req *pb.TSSpooferRequest) (*pb.TSSpooferResponse, error) {
	resp, err := a.s.GetTSSpoofers(req)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return resp, err
	}
}

func (a api) TestNewVPCapabilities(ctx context.Context, req *pb.TestNewVPCapabilitiesRequest) (*pb.TestNewVPCapabilitiesResponse, error) {
	resp, err := a.s.TestNewVPCapabilities(req)
	return resp, err
}

func (a api) TryUnquarantine(ctx context.Context, req *pb.TryUnquarantineRequest) (*pb.TryUnquarantineResponse, error) {
	resp, err := a.s.TryUnquarantine(req)
	return resp, err
}
