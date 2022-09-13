package v2api

import (
	"crypto/tls"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	repo "github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// CreateServer creates a grpc Server that serves the v2api
func CreateServer(s server.RevtrServer, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{}
	opts = append(opts, grpc.MaxRecvMsgSize(2147483647))
	opts = append(opts, grpc.MaxSendMsgSize(2147483647))
	if environment.IsDebugRevtr{	
		serv := grpc.NewServer(opts...)
		pb.RegisterRevtrServer(serv, CreateAPI(s))
		return serv
	} else {
		opts = append(opts, grpc.Creds(credentials.NewTLS(conf)))
		serv := grpc.NewServer(opts...)
		pb.RegisterRevtrServer(serv, CreateAPI(s))
		return serv
	}
}

// CreateAPI returns pb.RevtrServer that uses the revtr.Server s
func CreateAPI(s server.RevtrServer) pb.RevtrServer {
	return V2Api{s: s}
}

type V2Api struct {
	s server.RevtrServer
}

const (
	authHeader = "Revtr-key"
)

var (
	ErrUnauthorizedRequest = grpc.Errorf(codes.Unauthenticated, "unauthorized request")
	ErrInvalidBatchId      = grpc.Errorf(codes.FailedPrecondition, "invalid batch id")
	ErrNoRevtrsToRun       = grpc.Errorf(codes.FailedPrecondition, "no revtrs to run")
	ErrFailedToCreateBatch = grpc.Errorf(codes.Internal, "failed to create batch")
)

func checkAuth(m metadata.MD) (string, bool) {
	if val, ok := m[authHeader]; ok {
		if len(val) != 1 {
			return "", false
		}
		return val[0], true
	}
	return "", false
}

func (v2 V2Api) RunRevtr(ctx context.Context, req *pb.RunRevtrReq) (*pb.RunRevtrResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.RunRevtr(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) GetRevtr(ctx context.Context, req *pb.GetRevtrReq) (*pb.GetRevtrResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.GetRevtr(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) GetRevtrByLabel(ctx context.Context, req *pb.GetRevtrByLabelReq) (*pb.GetRevtrByLabelResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.GetRevtrByLabel(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) GetSources(ctx context.Context, req *pb.GetSourcesReq) (*pb.GetSourcesResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.GetSources(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) GetRevtrMetaOnly(ctx context.Context, req *pb.GetRevtrMetaOnlyReq) (*pb.GetRevtrMetaOnlyResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}

	ret, err := v2.s.GetRevtrsMetaOnly(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) GetRevtrBatchStatus(ctx context.Context, req *pb.GetRevtrBatchStatusReq) (*pb.GetRevtrBatchStatusResp, error) {	
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.GetRevtrBatchStatus(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) UpdateRevtr(ctx context.Context, req *pb.UpdateRevtrReq) (*pb.UpdateRevtrResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.UpdateRevtr(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) CleanAtlas(ctx context.Context, req *pb.CleanAtlasReq) (*pb.CleanAtlasResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.CleanAtlas(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (v2 V2Api) RunAtlas(ctx context.Context, req *pb.RunAtlasReq) (*pb.RunAtlasResp, error) {
	if md, hasMD := metadata.FromIncomingContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := v2.s.RunAtlas(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func rpcError(err error) error {
	switch err {
	case repo.ErrNoRevtrUserFound:
		return ErrUnauthorizedRequest
	default:
		return err
	}
}
