package client

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	pb "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	pb.RankingClient
}

// RankingSource is the interface to something that gives vps
type RankingSource interface {
	GetVPs(string, uint32, environment.RRVPsSelectionTechnique) (*pb.GetVPsResp, error)
	GetTargetsFromHitlist(in * pb.GetTargetsReq) (*pb.GetTargetsResp, error)
	GetRRSpoofers(addr string, max uint32, selectionTechnique environment.RRVPsSelectionTechnique) ([]*pb.Source, error) 
}

// New returns a VPSource
func New(ctx context.Context, cc *grpc.ClientConn) RankingSource {
	return client{Context: ctx, RankingClient: pb.NewRankingClient(cc)}
}

func (c client) GetRRSpoofers(addr string, max uint32, selectionTechnique environment.RRVPsSelectionTechnique) ([]*pb.Source, error) {
	resp, err := c.GetVPs(addr, max, selectionTechnique)
	if err != nil {
		return nil, err
	}
	// Convert these objects into vantage points for compatibility
	// ret := []*vpservicepb.VantagePoint{}
	// for _, vp := range resp.Vps {
	// 	ip, _ := util.IPStringToInt32(vp.Ip)
	// 	ret = append(ret, &vpservicepb.VantagePoint{
	// 		Hostname: vp.Hostname,
	// 		Ip: ip,
	// 	})
	// }
	return resp.Vps, nil 
}

func (c client) GetVPs(addr string, max uint32, selectionTechnique environment.RRVPsSelectionTechnique) (*pb.GetVPsResp, error) {
	
	// Wait until we have a response in case the ranking service is overloaded
	
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
	defer cancel()
	resp, err := c.RankingClient.GetVPs(ctx,  &pb.GetVPsReq{
		Ip:                   addr,
		NumVPs:               int32(max),
		RankingTechnique:     string(selectionTechnique),
		ExecutedMeasurements: nil, // Initial batch, so nothing executed yet.
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return resp, nil 
	
	// return nil, nil
}

func (c client) GetTargetsFromHitlist(in *pb.GetTargetsReq) (*pb.GetTargetsResp, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*60)
	defer cancel()
	targetsResp, err := c.RankingClient.GetTargetsFromHitlist(ctx, in)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return targetsResp, nil
}
