package revtr_test

import (
	// "flag"
	// "net"

	// "net/http/pprof"
	// "os"
	// "strconv"

	// "golang.org/x/net/context"
	// "golang.org/x/net/trace"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/grpclog"

	// "github.com/NEU-SNS/ReverseTraceroute/config"

	"context"
	"strings"
	"sync"
	"testing"

	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	revtrpb "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	revtrtypes "github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	survey "github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/stretchr/testify/require"

	// controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"

	// rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"

	// "github.com/NEU-SNS/ReverseTraceroute/revtr/client"
	"github.com/NEU-SNS/ReverseTraceroute/test"
)
 

func TestUserNonExisting(t *testing.T) {
	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}
	wrongKey := "dummy"
	
	_, err = reverseTracerouteClient.RunRevtr(context.Background(), &revtrpb.RunRevtrReq{Auth: wrongKey, Revtrs: nil})
	require.True(t, strings.Contains(err.Error(), revtrtypes.ErrUserNotFound.Error()))

}

func TestUserExisting(t *testing.T){

	// Create test user 
	db := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer db.Close()
	key := "dummy"
	id := InsertTestUser(db, revtrpb.RevtrUser{Name: "dummy", Key: key, MaxRevtrPerDay: 10, MaxParallelRevtr: 10}) 
	defer DeleteTestUser(db, id)

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	_, err = reverseTracerouteClient.RunRevtr(context.Background(), &revtrpb.RunRevtrReq{Auth: key, Revtrs: nil})
	require.True(t, strings.Contains(err.Error(), revtrtypes.ErrNoRevtrsToRun.Error()))


}

func TestUserNoBudget(t *testing.T) {
	
	// Create test user 
	db := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer db.Close()
	key := "dummy"
	id := InsertTestUser(db, pb.RevtrUser{Name: "dummy", Key: key, MaxRevtrPerDay: 0, MaxParallelRevtr: 10}) 
	defer DeleteTestUser(db, id)


	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	source, err := util.Int32ToIPString(3589431526)
	destination, _ := util.Int32ToIPString(71677608)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)

	parameters := survey.MeasurementSurveyParameters{
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness : 43800 * 12, // a year 
			// StalenessBeforeRefresh: 1,
			Platforms: []string{"ripe"},
		},
		RRHeuristicsOptions: revtr.RRHeuristicsOptions{
			UseDoubleStamp: true,
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,
		IsRunForwardTraceroute: false,
		IsRunRTTPings: true,
	}

	revtr := pb.RevtrMeasurement{
			Src: source,
			Dst: destination,
			// Staleness is for the staleness of the atlas
			RrVpSelectionAlgorithm: string(parameters.RankingTechnique),
			UseTimestamp: parameters.UseTimestamp, 
			UseCache : parameters.UseCache,
			AtlasOptions: &revtrpb.AtlasOptions{
				UseAtlas: parameters.AtlasOptions.UseAtlas,
				UseRrPings: parameters.AtlasOptions.UseRRPings,
				IgnoreSource: parameters.AtlasOptions.IgnoreSource,
				IgnoreSourceAs: parameters.AtlasOptions.IgnoreSourceAS,
				StalenessBeforeRefresh: parameters.AtlasOptions.StalenessBeforeRefresh,
				Platforms: parameters.AtlasOptions.Platforms,
				Staleness: parameters.AtlasOptions.Staleness,
			},
			CheckDestBasedRoutingOptions: &revtrpb.CheckDestBasedRoutingOptions{
				CheckTunnel: parameters.CheckDestinationBasedRouting.CheckTunnel,
			},
			HeuristicsOptions: &revtrpb.RRHeuristicsOptions {
				UseDoubleStamp: parameters.RRHeuristicsOptions.UseDoubleStamp, 
			},
			MaxSpoofers: uint32(parameters.MaxSpoofers),
			Label: parameters.Label,
			IsRunForwardTraceroute: parameters.IsRunForwardTraceroute,
			IsRunRttPings: parameters.IsRunRTTPings,
	}


	_, err = reverseTracerouteClient.RunRevtr(context.Background(), &pb.RunRevtrReq{Auth: key, Revtrs: []*pb.RevtrMeasurement{&revtr}})
	require.True(t, strings.Contains(err.Error(), revtrtypes.ErrTooManyMesaurementsToday.Error()))

}


func TestUserTooManyParallelRevtr(t *testing.T) {
	
	// Create test user 
	db := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer db.Close()
	key := "dummy"
	id := InsertTestUser(db, pb.RevtrUser{Name: "dummy", Key: key, MaxRevtrPerDay: 100, MaxParallelRevtr: 1}) 
	defer DeleteTestUser(db, id)


	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	source, err := util.Int32ToIPString(3589431526)
	destination, _ := util.Int32ToIPString(71677608)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)

	parameters := survey.MeasurementSurveyParameters{
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness : 43800 * 12, // a year 
			// StalenessBeforeRefresh: 1,
			Platforms: []string{"ripe"},
		},
		RRHeuristicsOptions: revtr.RRHeuristicsOptions{
			UseDoubleStamp: true,
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,
		IsRunForwardTraceroute: false,
		IsRunRTTPings: true,
	}

	revtr := pb.RevtrMeasurement{
			Src: source,
			Dst: destination,
			// Staleness is for the staleness of the atlas
			RrVpSelectionAlgorithm: string(parameters.RankingTechnique),
			UseTimestamp: parameters.UseTimestamp, 
			UseCache : parameters.UseCache,
			AtlasOptions: &revtrpb.AtlasOptions{
				UseAtlas: parameters.AtlasOptions.UseAtlas,
				UseRrPings: parameters.AtlasOptions.UseRRPings,
				IgnoreSource: parameters.AtlasOptions.IgnoreSource,
				IgnoreSourceAs: parameters.AtlasOptions.IgnoreSourceAS,
				StalenessBeforeRefresh: parameters.AtlasOptions.StalenessBeforeRefresh,
				Platforms: parameters.AtlasOptions.Platforms,
				Staleness: parameters.AtlasOptions.Staleness,
			},
			CheckDestBasedRoutingOptions: &revtrpb.CheckDestBasedRoutingOptions{
				CheckTunnel: parameters.CheckDestinationBasedRouting.CheckTunnel,
			},
			HeuristicsOptions: &revtrpb.RRHeuristicsOptions {
				UseDoubleStamp: parameters.RRHeuristicsOptions.UseDoubleStamp, 
			},
			MaxSpoofers: uint32(parameters.MaxSpoofers),
			Label: parameters.Label,
			IsRunForwardTraceroute: parameters.IsRunForwardTraceroute,
			IsRunRttPings: parameters.IsRunRTTPings,
	}


	_, err = reverseTracerouteClient.RunRevtr(context.Background(), &pb.RunRevtrReq{Auth: key, Revtrs: []*pb.RevtrMeasurement{&revtr, &revtr}})
	require.True(t, strings.Contains(err.Error(), revtrtypes.ErrTooManyMesaurementsInParallel.Error()))

}

func TestUserTooManyParallelRevtrMultiRequest(t *testing.T) {

	// Create test user 
	db := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer db.Close()
	key := "dummy"
	maxParallelRevtr := uint32(5)
	id := InsertTestUser(db, pb.RevtrUser{Name: "dummy", Key: key, MaxRevtrPerDay: 100, MaxParallelRevtr: maxParallelRevtr}) 
	defer DeleteTestUser(db, id)


	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	source, err := util.Int32ToIPString(68067161)
	destination, _ := util.Int32ToIPString(2967384077)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)

	parameters := survey.MeasurementSurveyParameters{
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness : 43800 * 12, // a year 
			// StalenessBeforeRefresh: 1,
			Platforms: []string{"ripe"},
		},
		RRHeuristicsOptions: revtr.RRHeuristicsOptions{
			UseDoubleStamp: true,
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,
		IsRunForwardTraceroute: false,
		IsRunRTTPings: true,
	}

	revtr := pb.RevtrMeasurement{
			Src: source,
			Dst: destination,
			// Staleness is for the staleness of the atlas
			RrVpSelectionAlgorithm: string(parameters.RankingTechnique),
			UseTimestamp: parameters.UseTimestamp, 
			UseCache : parameters.UseCache,
			AtlasOptions: &revtrpb.AtlasOptions{
				UseAtlas: parameters.AtlasOptions.UseAtlas,
				UseRrPings: parameters.AtlasOptions.UseRRPings,
				IgnoreSource: parameters.AtlasOptions.IgnoreSource,
				IgnoreSourceAs: parameters.AtlasOptions.IgnoreSourceAS,
				StalenessBeforeRefresh: parameters.AtlasOptions.StalenessBeforeRefresh,
				Platforms: parameters.AtlasOptions.Platforms,
				Staleness: parameters.AtlasOptions.Staleness,
			},
			CheckDestBasedRoutingOptions: &revtrpb.CheckDestBasedRoutingOptions{
				CheckTunnel: parameters.CheckDestinationBasedRouting.CheckTunnel,
			},
			HeuristicsOptions: &revtrpb.RRHeuristicsOptions {
				UseDoubleStamp: parameters.RRHeuristicsOptions.UseDoubleStamp, 
			},
			MaxSpoofers: uint32(parameters.MaxSpoofers),
			Label: parameters.Label,
			IsRunForwardTraceroute: parameters.IsRunForwardTraceroute,
			IsRunRttPings: parameters.IsRunRTTPings,
	}

	errChan := make(chan error, 10)
	wg := new(sync.WaitGroup)
	for i := 0; i < int(maxParallelRevtr) + 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err = reverseTracerouteClient.RunRevtr(context.Background(), &pb.RunRevtrReq{Auth: key, Revtrs: []*pb.RevtrMeasurement{&revtr}})
			errChan <- err
		} ()	
	}
	wg.Wait()
	close(errChan)
	nonError := uint32(0)
	errors := uint32(0)
	for err := range(errChan){
		if err == nil {
			nonError += 1
		} else {
			errors += 1
			require.True(t, strings.Contains(err.Error(), revtrtypes.ErrTooManyMesaurementsInParallel.Error()))
		}
	}

	require.Equal(t, maxParallelRevtr, nonError)

	//

}

func TestUserRunRevtr(t *testing.T) {
	
	// Create test user 
	db := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer db.Close()
	key := "dummy"
	id := InsertTestUser(db, pb.RevtrUser{Name: "dummy", Key: key, MaxRevtrPerDay: 100, MaxParallelRevtr: 1}) 
	defer DeleteTestUser(db, id)


	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	source, err := util.Int32ToIPString(68067161)
	destination, _ := util.Int32ToIPString(2967384077)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)

	parameters := survey.MeasurementSurveyParameters{
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness : 43800, // a month 
			// StalenessBeforeRefresh: 1,
			Platforms: []string{"ripe"},
		},
		RRHeuristicsOptions: revtr.RRHeuristicsOptions{
			UseDoubleStamp: true,
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,
		IsRunForwardTraceroute: false,
		IsRunRTTPings: true,
		Label: "test_user",
	}

	revtr := pb.RevtrMeasurement{
			Src: source,
			Dst: destination,
			// Staleness is for the staleness of the atlas
			RrVpSelectionAlgorithm: string(parameters.RankingTechnique),
			UseTimestamp: parameters.UseTimestamp, 
			UseCache : parameters.UseCache,
			AtlasOptions: &revtrpb.AtlasOptions{
				UseAtlas: parameters.AtlasOptions.UseAtlas,
				UseRrPings: parameters.AtlasOptions.UseRRPings,
				IgnoreSource: parameters.AtlasOptions.IgnoreSource,
				IgnoreSourceAs: parameters.AtlasOptions.IgnoreSourceAS,
				StalenessBeforeRefresh: parameters.AtlasOptions.StalenessBeforeRefresh,
				Platforms: parameters.AtlasOptions.Platforms,
				Staleness: parameters.AtlasOptions.Staleness,
			},
			CheckDestBasedRoutingOptions: &revtrpb.CheckDestBasedRoutingOptions{
				CheckTunnel: parameters.CheckDestinationBasedRouting.CheckTunnel,
			},
			HeuristicsOptions: &revtrpb.RRHeuristicsOptions {
				UseDoubleStamp: parameters.RRHeuristicsOptions.UseDoubleStamp, 
			},
			MaxSpoofers: uint32(parameters.MaxSpoofers),
			Label: parameters.Label,
			IsRunForwardTraceroute: parameters.IsRunForwardTraceroute,
			IsRunRttPings: parameters.IsRunRTTPings,
	}


	_, err = reverseTracerouteClient.RunRevtr(context.Background(), &pb.RunRevtrReq{Auth: key, Revtrs: []*pb.RevtrMeasurement{&revtr}})
	require.Equal(t, err, nil)

}





