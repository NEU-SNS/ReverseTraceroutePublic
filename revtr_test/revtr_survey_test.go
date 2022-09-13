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
	"fmt"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	survey "github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/stretchr/testify/require"

	// controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	// rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	// "github.com/NEU-SNS/ReverseTraceroute/revtr/client"
	"github.com/NEU-SNS/ReverseTraceroute/test"
)
 
var now = time.Now()
var date string = fmt.Sprintf("'%s'", now.Format("2006-01-02 15:04:05"))

func TestRevtrTRIntersection(t *testing.T){

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	db := test.GetMySQLDB("../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	id := test.InsertTestTraceroute(1, 2, platform, date, db) 
	defer test.DeleteTestTraceroute(db, id)

	source, err := util.Int32ToIPString(2)
	destination, _ := util.Int32ToIPString(1)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	_, mappingIPProbeID := util.GetRIPEAnchorsIP()
	mappingIPProbeID[destination] = 0
	_, ok := mappingIPProbeID[destination]
	if !ok {
		panic(fmt.Errorf("Destination %s not in RIPE Anchors", destination))
	}
	parameters := survey.MeasurementSurveyParameters{
		MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
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
			Platforms: []string{platform},
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
	}
	label := "debug_" + util.GenUUID()
	parameters.Label = label
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
	}
	
	// Retrieve the reverse traceroute in the DB 
	resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
	if err != nil  {
		panic(err)
	}
	require.EqualValues(t, len(resp.Revtrs), 1)
	expectedHops := [] uint32 {1, 3093903169, 3093905493, 3093902558, 71683070, 2}
	for i, hop := range(resp.Revtrs[0].Path)  {
		hopI, _ := util.IPStringToInt32(hop.Hop)
		require.EqualValues(t, hopI, expectedHops[i])
		if i == 0 {
			continue
		}
		require.EqualValues(t, hop.Type, pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT)
	}
	// revtrDB := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	// defer revtrDB.Close()
	// test.DeleteReverseTracerouteByLabel(revtrDB, label)

}


func TestRevtrRRIntersection(t *testing.T){

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	db := test.GetMySQLDB("../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	id := test.InsertTestTraceroute(1, 2, platform, date, db) 
	defer test.DeleteTestTraceroute(db, id)

	source, err := util.Int32ToIPString(2)
	destination, _ := util.Int32ToIPString(71677608)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	_, mappingIPProbeID := util.GetRIPEAnchorsIP()
	mappingIPProbeID[destination] = 0
	_, ok := mappingIPProbeID[destination]
	if !ok {
		panic(fmt.Errorf("Destination %s not in RIPE Anchors", destination))
	}
	parameters := survey.MeasurementSurveyParameters{
		MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
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
			Platforms: []string{platform},
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
	}
	label := "debug_" + util.GenUUID()
	parameters.Label = label
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
	}
	
	// Retrieve the reverse traceroute in the DB 
	resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
	if err != nil  {
		panic(err)
	}
	require.EqualValues(t, len(resp.Revtrs), 1)
	expectedHops := [] uint32 {71677608, 1, 3093903169, 3093905493, 3093902558, 71683070, 2}
	for i, hop := range(resp.Revtrs[0].Path)  {
		hopI, _ := util.IPStringToInt32(hop.Hop)
		require.EqualValues(t, hopI, expectedHops[i])
		if i == 0 {
			continue
		}
		if 1 <= i && i <= 5{
			require.EqualValues(t, hop.Type, pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT_BETWEEN)
		} else {
			require.EqualValues(t, hop.Type, pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT)
		}
		
	}
	// revtrDB := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	// defer revtrDB.Close()
	// test.DeleteReverseTracerouteByLabel(revtrDB, label)

}


func TestRevtrRTT(t *testing.T){

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	db := test.GetMySQLDB("../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	srcI := uint32(68067161)
	id := test.InsertTestTraceroute(1, srcI, platform, date, db) 
	defer test.DeleteTestTraceroute(db, id)

	source, err := util.Int32ToIPString(srcI)
	destination, _ := util.Int32ToIPString(71677608)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	_, mappingIPProbeID := util.GetRIPEAnchorsIP()
	mappingIPProbeID[destination] = 0
	_, ok := mappingIPProbeID[destination]
	if !ok {
		panic(fmt.Errorf("Destination %s not in RIPE Anchors", destination))
	}
	parameters := survey.MeasurementSurveyParameters{
		MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
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
			Platforms: []string{platform},
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
	label := "debug_" + util.GenUUID()
	parameters.Label = label
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
	}
	
	// Retrieve the reverse traceroute in the DB 
	resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
	if err != nil  {
		panic(err)
	}
	require.EqualValues(t, len(resp.Revtrs), 1)
	expectedHops := [] uint32 {71677608, 1, 3093903169, 3093905493, 3093902558, 71683070, srcI}
	for i, hop := range(resp.Revtrs[0].Path)  {
		hopI, _ := util.IPStringToInt32(hop.Hop)
		require.EqualValues(t, hopI, expectedHops[i])
		if i == 0 {
			continue
		}
		if 1 <= i && i <= 5{
			require.EqualValues(t, hop.Type, pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT_BETWEEN)
		} else {
			require.EqualValues(t, hop.Type, pb.RevtrHopType_TR_TO_SRC_REV_SEGMENT)
		}
		require.True(t, hop.RttMeasurementId != 0)
	}
	// revtrDB := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	// defer revtrDB.Close()
	// test.DeleteReverseTracerouteByLabel(revtrDB, label)

}



func TestRevtrNonZeroRTT(t *testing.T){

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	srcI := uint32(3367879859)

	source, err := util.Int32ToIPString(srcI)
	destination, _ := util.Int32ToIPString(1137835009)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	_, mappingIPProbeID := util.GetRIPEAnchorsIP()
	mappingIPProbeID[destination] = 0
	_, ok := mappingIPProbeID[destination]
	if !ok {
		panic(fmt.Errorf("Destination %s not in RIPE Anchors", destination))
	}
	parameters := survey.MeasurementSurveyParameters{
		MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
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
			Platforms: []string{"debug"},
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
	label := "debug_" + util.GenUUID()
	parameters.Label = label
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
	}
	
	// Retrieve the reverse traceroute in the DB 
	resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
	if err != nil  {
		panic(err)
	}
	require.EqualValues(t, len(resp.Revtrs), 1)
	for _, hop := range(resp.Revtrs[0].Path)  {
		
		require.True(t, hop.RttMeasurementId != 0)
	}
	// revtrDB := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	// defer revtrDB.Close()
	// test.DeleteReverseTracerouteByLabel(revtrDB, label)

}

func TestRevtrIntermediateHopCache(t *testing.T){

	
	revtrDB := test.GetMySQLDB("../cmd/revtr/revtr.config", "revtr")
	defer revtrDB.Close()
	label := "debug_intermediate_hop_cache"
	test.DeleteReverseTracerouteByLabel(revtrDB, label)
	intermediateLabel := "debug_intermediate_hop_cache_2" 
	test.DeleteReverseTracerouteByLabel(revtrDB, intermediateLabel)
	for i := 11213 ; i <= 11220; i++ {
		err := test.ClearCache(i)
		if err != nil {
			panic(err)
		}
	} 

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	srcI := uint32(3279003918)

	source, err := util.Int32ToIPString(srcI)
	destination, _ := util.Int32ToIPString(1042834101)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	_, mappingIPProbeID := util.GetRIPEAnchorsIP()
	mappingIPProbeID[destination] = 0
	_, ok := mappingIPProbeID[destination]
	if !ok {
		panic(fmt.Errorf("Destination %s not in RIPE Anchors", destination))
	}
	parameters := survey.MeasurementSurveyParameters{
		MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
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
			Platforms: []string{"debug"},
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
		FirstFetchResult: 1,
		UseCache: true,
	}
	 
	parameters.Label = label
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
	}

	
	
	// Retrieve the reverse traceroute in the DB 
	
	resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
	if err != nil  {
		panic(err)
	}
	status := resp.Revtrs[0].Status.String()
	for status != "COMPLETED" {
		resp, err = reverseTracerouteClient.GetRevtrByLabel(context.Background(),
	 &pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
		if err != nil  {
			panic(err)
		}	
		status = resp.Revtrs[0].Status.String()
		time.Sleep(1) // Wait for the revtr to be inserted in the DB
	}
	

	

	// Get one of the intermediate hops and run a reverse traceroute to it. 
	require.EqualValues(t, len(resp.Revtrs), 1)
	for _, hop := range(resp.Revtrs[0].Path)  {

		// Run a reverse traceroute from an intermediate hop 
		if hop.Hop == destination {
			continue
		}
		intermediateDestination := hop.Hop
		log.Infof("Debugging intermediate hop reverse traceroute from %s to %s", source, intermediateDestination)
		sources := [] string {source}
		destinations := [] string {intermediateDestination}

		batch := []survey.SourceDestinationPairIndex {}
		batch = append(batch, survey.SourceDestinationPairIndex {
			Source: 0,
			Destination: 0,
		})

		parameters.Label = intermediateLabel
		// First step of debugging, try to run a traceroute from the destination to the source
		for i := 0; i < 1; i++ {
			survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
				&sources, &destinations,
				parameters)
		}
		
		// Retrieve the reverse traceroute in the DB 
		resp, err := reverseTracerouteClient.GetRevtrByLabel(context.Background(),
		&pb.GetRevtrByLabelReq{Label: intermediateLabel, Auth: parameters.AuthKey})
		if err != nil  {
			panic(err)
		}

		status := resp.Revtrs[0].Status.String()
		for status != "COMPLETED" {
			resp, err = reverseTracerouteClient.GetRevtrByLabel(context.Background(),
		&pb.GetRevtrByLabelReq{Label: label, Auth: parameters.AuthKey})
			if err != nil  {
				panic(err)
			}	
			status = resp.Revtrs[0].Status.String()
			time.Sleep(1) // Wait for the revtr to be inserted in the DB
		}

		require.Equal(t, resp.Revtrs[0].FailReason, "")

		// Check that the hops on this revtr are all from cache
		require.EqualValues(t, len(resp.Revtrs), 1)
		for _, hopIntermediateRevtr := range(resp.Revtrs[0].Path)  {
			if hopIntermediateRevtr.Hop != intermediateDestination {
				require.True(t, hopIntermediateRevtr.FromCache)
			}
			
		}
		break	
		
	}
}