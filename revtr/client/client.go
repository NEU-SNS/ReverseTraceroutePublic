package client

import (
	revtrapi "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	// "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	revtrapi.RevtrClient
}

// Client is a client for the controller
type Client interface {
	RunRevtr(context.Context, *revtrapi.RunRevtrReq) (*revtrapi.RunRevtrResp, error)
	GetRevtr(context.Context, *revtrapi.GetRevtrReq) (*revtrapi.GetRevtrResp, error)
	GetRevtrBatchStatus(context.Context, *revtrapi.GetRevtrBatchStatusReq) (*revtrapi.GetRevtrBatchStatusResp, error)
	GetSources(context.Context, *revtrapi.GetSourcesReq) (*revtrapi.GetSourcesResp, error)
	RunRecordRoute(context.Context, *revtrapi.RunRecordRouteReq) (*revtrapi.RunRecordRouteResp, error)
}

// New creates a new controller client
func New(ctx context.Context, cc *grpc.ClientConn) Client {
	return client{Context: ctx, RevtrClient: revtrapi.NewRevtrClient(cc)}
}

func (c client) RunRevtr(ctx context.Context, req *revtrapi.RunRevtrReq) (*revtrapi.RunRevtrResp, error) {
	return c.RevtrClient.RunRevtr(ctx, req)
}

func (c client) GetRevtr(ctx context.Context, req *revtrapi.GetRevtrReq) (*revtrapi.GetRevtrResp, error) {
	return c.RevtrClient.GetRevtr(ctx, req)
}

func (c client) GetRevtrBatchStatus(ctx context.Context, req *revtrapi.GetRevtrBatchStatusReq) (*revtrapi.GetRevtrBatchStatusResp, error) {
	return c.RevtrClient.GetRevtrBatchStatus(ctx, req)
}

func (c client) GetSources(ctx context.Context, req *revtrapi.GetSourcesReq) (*revtrapi.GetSourcesResp, error) {
	return c.RevtrClient.GetSources(ctx, req)
}

func (c client) RunRecordRoute(ctx context.Context, req *revtrapi.RunRecordRouteReq) (*revtrapi.RunRecordRouteResp, error){
	return c.RevtrClient.RunRecordRoute(ctx, req)
}