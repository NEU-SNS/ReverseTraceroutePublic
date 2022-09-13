package client

import (
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	pb.AtlasClient
}

// Atlas is the atlas
type Atlas interface {
	GetIntersectingPath(context.Context) (pb.Atlas_GetIntersectingPathClient, error)
	GetPathsWithToken(context.Context) (pb.Atlas_GetPathsWithTokenClient, error)
	InsertTraceroutes(context.Context, []*dm.Traceroute, bool, bool) ([]int64, error)
	CheckIntersectingPath(context.Context, int64, uint32) (*pb.Path, error)
	MarkTracerouteStale(context.Context, uint32, int64, int64) error
	MarkTracerouteStaleSource(ctx context.Context, source uint32) error
	RunTracerouteAtlasToSource(ctx context.Context, source uint32) error
	RunAtlasRRPings(context.Context, []*dm.Traceroute, bool) error 
	GetAvailableHopAtlasPerSource(context.Context) (map[uint32]*pb.AtlasHops, error)

}

// New returns a new atlas
func New(ctx context.Context, cc *grpc.ClientConn) Atlas {
	return client{Context: ctx, AtlasClient: pb.NewAtlasClient(cc)}
}

// GetIntersectingPath gets an intersecting path
func (c client) GetIntersectingPath(ctx context.Context) (pb.Atlas_GetIntersectingPathClient, error) {
	return c.AtlasClient.GetIntersectingPath(ctx)
}

// GetPathsWithToken gets a path from a token
func (c client) GetPathsWithToken(ctx context.Context) (pb.Atlas_GetPathsWithTokenClient, error) {
	return c.AtlasClient.GetPathsWithToken(ctx)
}

func (c client) InsertTraceroutes(ctx context.Context, trs []*dm.Traceroute, isRunRRPings bool, onlyIntersections bool) ([]int64, error) {
	response, err := c.AtlasClient.InsertTraceroutes(ctx, &pb.InsertTraceroutesRequest{Traceroutes: trs, IsRunRrPings: isRunRRPings, OnlyIntersections: onlyIntersections})
	return response.Ids, err
}

func (c client) CheckIntersectingPath(ctx context.Context, tracerouteID int64, hopIntersection uint32) (*pb.Path, error) {
	resp, err := c.AtlasClient.CheckIntersectingPath(ctx, 
		&pb.CheckIntersectionRequest{
			TracerouteId: tracerouteID,
			HopIntersection: hopIntersection,
		})
	return resp.Path, err
}

func (c client) MarkTracerouteStale(ctx context.Context, intersectIP uint32, oldTracerouteID int64, newTracerouteID int64) error {
	_, err := c.AtlasClient.MarkTracerouteStale(ctx, &pb.MarkTracerouteStaleRequest{
		IntersectIP: intersectIP,
		OldTracerouteId: oldTracerouteID,
		NewTracerouteId: newTracerouteID,
	})

	return err
}

func (c client) MarkTracerouteStaleSource(ctx context.Context, source uint32) error {
	_, err := c.AtlasClient.MarkTracerouteStaleSource(ctx, &pb.MarkTracerouteStaleSourceRequest{
		Source: source,
	})

	return err
}

func (c client) RunAtlasRRPings(ctx context.Context, traceroutes []* dm.Traceroute, onlyIntersections bool) error {
	_, err := c.AtlasClient.RunAtlasRRPings(ctx, &pb.RunAtlasRRPingsRequest{
		Traceroutes: traceroutes,
		OnlyIntersections: onlyIntersections,
	})
	return err
}

func (c client) RunTracerouteAtlasToSource(ctx context.Context, source uint32) error {
	_, err := c.AtlasClient.RunTracerouteAtlasToSource(ctx, &pb.RunTracerouteAtlasToSourceRequest{
		Source: source,
	})

	return err 
}

func (c client) GetAvailableHopAtlasPerSource(ctx context.Context) (map[uint32]*pb.AtlasHops, error) {
	res, err := c.AtlasClient.GetAvailableHopAtlasPerSource(ctx, &pb.GetAvailableHopAtlasPerSourceRequest{})
	if err != nil  {
		return nil, err
	}
	return res.HopsPerSource, nil 
}