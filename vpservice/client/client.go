package client

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	pb.VPServiceClient
}

// VPSource is the inteface to something that gives vps
type VPSource interface {
	GetVPs() (*pb.VPReturn, error)
	GetOneVPPerSite() (*pb.VPReturn, error)
	GetRRSpoofers(addr, max uint32) ([]*pb.VantagePoint, error)
	GetTSSpoofers(max uint32) ([]*pb.VantagePoint, error)
}

// New returns a VPSource
func New(ctx context.Context, cc *grpc.ClientConn) VPSource {
	return client{Context: ctx, VPServiceClient: pb.NewVPServiceClient(cc)}
}

func (c client) GetVPs() (*pb.VPReturn, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
	defer cancel()
	vpr, err := c.VPServiceClient.GetVPs(ctx, &pb.VPRequest{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return vpr, nil
}

func (c client) GetOneVPPerSite() (*pb.VPReturn, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
	defer cancel()
	vpr, err := c.VPServiceClient.GetVPs(ctx, &pb.VPRequest{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	set := make(map[string]*pb.VantagePoint)
	vps := vpr.GetVps()
	for _, vp := range vps {
		set[vp.Site] = vp
	}
	var ret []*pb.VantagePoint
	for _, val := range set {
		ret = append(ret, val)
	}
	vpr.Vps = ret
	return vpr, nil
}

func (c client) GetRRSpoofers(addr, max uint32) ([]*pb.VantagePoint, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
	defer cancel()
	arg := &pb.RRSpooferRequest{
		Addr: addr,
		Max:  max,
	}
	sr, err := c.VPServiceClient.GetRRSpoofers(ctx, arg)
	if err != nil {
		return nil, err
	}
	return sr.Spoofers, nil
}

func (c client) GetTSSpoofers(max uint32) ([]*pb.VantagePoint, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
	defer cancel()
	arg := &pb.TSSpooferRequest{
		Max: max,
	}
	sr, err := c.VPServiceClient.GetTSSpoofers(ctx, arg)
	if err != nil {
		return nil, err
	}
	return sr.Spoofers, nil
}
