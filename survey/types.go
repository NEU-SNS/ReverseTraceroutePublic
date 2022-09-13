package survey

import (
	"strconv"

	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
)

type MeasurementPingSurveyParameters struct {

}
 
type MeasurementSurveyParameters struct {
	AuthKey string
	RIPEAPIKey string 
	BatchLen int
	MaximumMeasurements int 
	IsCheckReversePath bool
	MappingIPProbeID map[string]int
	MaxSpoofers int 
	// Parameters for evaluation
	RankingTechnique env.RRVPsSelectionTechnique
	AtlasOptions     revtr.AtlasOptions
	RRHeuristicsOptions revtr.RRHeuristicsOptions
	NParallelRevtr   int
	UseTimestamp    bool
	// Use cache RR and spoofed RR results. 
	UseCache         bool 
	// Time of caching spoofed results in seconds. 
	Cache            int

	// Label of the evaluation
	Label string

	// Dry run, just for debug
	IsDryRun bool

	// For ping surveys
	IsRecordRoute bool
	IsTimestamp bool
	NPackets int 

	// CheckDestinationBasedRouting optoins
	CheckDestinationBasedRouting revtr.CheckDestinationBasedRoutingOptions
	
	// Check if the revtr is already present in the DB, only for huge surveys
	CheckDB bool

	// IsRunForwardTraceroute
	IsRunForwardTraceroute bool 

	// IsRunRTTPings
	IsRunRTTPings bool

	// UseMultipleGoroutines
	IsUseMultipleGoroutines bool 

	// Wait for results before issuing the next batch
	IsWaitBeforeNextBatch bool

	// How long to wait before starting to fetch result
	FirstFetchResult int 

	// Minimum number of revtrs still be running to not start the next batch
	MinimumRunningRevtrsBeforeNextBatch int 

	// Timeout to tell when to stop waiting for a measurement. 
	Timeout int // seconds
}

type TimestampSurveyParameters struct {

	// Types of different timestamp probes to check link A, B on forward path from S to D: 
	// IP TS (D|A,B)
	// IP TS (B|A,B)

	// Types of different timestamp probes to check link B, A on reverse path from D to S:
	// IP TS (D|B,A)
	// IP TS (B|B,A)
	// ICMP TS (B as S), ICMP TS (A as S)
}

type TimestampLink struct {
	src uint32 // Source of the traceroute where the link was found
	dst uint32 // Destination of the traceroute where the link was found
	left uint32 // Left IP part of the link
	right uint32 // Right IP part of the link
	direction string // direction of the link (forward) or reverse
}

const (
	REVERSE = "reverse"
	FORWARD = "forward"
)


func (p *MeasurementSurveyParameters) ToString() string{
	return string(p.RankingTechnique) + "_" + strconv.Itoa(p.MaxSpoofers) + "_" +  string(p.AtlasOptions.ToString()) +
	"_" + strconv.Itoa(p.NParallelRevtr) + "_" + strconv.Itoa(p.Cache) 
}

type SendAndWaitFunc func(client interface{}, controllerClient controller.ControllerClient, 
	batch *[]SourceDestinationPairIndex,
	sources *[] string,
	destinations *[]string, 
	parameters MeasurementSurveyParameters) []uint32

type SourceDestinationPairIndex struct {
	Source uint32
	Destination uint32
}


