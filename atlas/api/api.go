package api

import (
	"context"
	"crypto/tls"
	"io"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateServer creates a grpc server fro the Atlas api
func CreateServer(s server.AtlasServer, conf *tls.Config) *grpc.Server {
	opts :=[]grpc.ServerOption{}
	opts = append(opts, grpc.MaxRecvMsgSize(10010241024))
	opts = append(opts, grpc.MaxSendMsgSize(10010241024))
	if !environment.IsDebugAtlas{
		opts = append(opts, grpc.Creds(credentials.NewTLS(conf)))
		
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterAtlasServer(serv, CreateAPI(s))
	return serv
}

type api struct {
	s server.AtlasServer
}

// CreateAPI returns a pb.AtlasServer that uses the given server
func CreateAPI(s server.AtlasServer) pb.AtlasServer {
	return api{s: s}
}

func (a api) GetIntersectingPath(stream pb.Atlas_GetIntersectingPathServer) error {
	ctx := stream.Context()
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}
		resp, err := a.s.GetIntersectingPath(req)
		if err != nil {
			log.Error(err)
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

func (a api) GetPathsWithToken(stream pb.Atlas_GetPathsWithTokenServer) error {
	ctx := stream.Context()
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}
		resp, err := a.s.GetPathsWithToken(req)
		if err != nil {
			log.Error(err)
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}


func (a api) InsertTraceroutes(context context.Context, req *pb.InsertTraceroutesRequest) (*pb.InsertTraceroutesResponse, error) {
	return a.s.InsertTraceroutes(req) 
}


func (a api) CheckIntersectingPath(ctx context.Context, req *pb.CheckIntersectionRequest) (*pb.CheckIntersectionResponse, error) {
	return a.s.CheckIntersectingPath(req)
}

func (a api) MarkTracerouteStale(ctx context.Context, req *pb.MarkTracerouteStaleRequest) (*pb.MarkTracerouteStaleResponse, error) {
	return a.s.MarkTracerouteStale(req)
}

func (a api) MarkTracerouteStaleSource(ctx context.Context, req *pb.MarkTracerouteStaleSourceRequest) (*pb.MarkTracerouteStaleSourceResponse, error) {
	return a.s.MarkTracerouteStaleSource(req)
}

func (a api) RunAtlasRRPings(ctx context.Context, req *pb.RunAtlasRRPingsRequest) (*pb.RunAtlasRRPingsResponse, error) {
	return a.s.RunAtlasRRPings(req)
}

func (a api) RunTracerouteAtlasToSource(ctx context.Context, req *pb.RunTracerouteAtlasToSourceRequest) (*pb.RunTracerouteAtlasToSourceResponse, error) {
	return a.s.RunTracerouteAtlasToSource(req)
}

func (a api) GetAvailableHopAtlasPerSource(ctx context.Context, req *pb.GetAvailableHopAtlasPerSourceRequest) (*pb.GetAvailableHopAtlasPerSourceResponse, error) {
	return a.s.GetAvailableHopAtlasPerSource(req)
}

