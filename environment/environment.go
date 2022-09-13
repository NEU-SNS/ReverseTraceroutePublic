package environment

// IsDebug environment variable
var IsDebugRevtr = false
var IsDebugAtlas = true 
var IsDebugVPService = false 
var IsDebugController = false
// IsDebugPLController should never been set to true because 
// we have only one PLController that can run at once on walter. 
var IsDebugPLController = false
var IsDebugRIPEAtlasController = true 
var IsDebugRankingService = true

// RevtrPortProduction setting for the revtr grpc port in production
var RevtrPortProduction = 45454
// RevtrPortDebug setting for the revtr grpc port in debug
var RevtrPortDebug = 45455

// RevtrAPIPortProduction setting for the revtr api port in debug
var RevtrAPIPortProduction = 8080
// RevtrAPIPortDebug setting for the revtr api port in debug
var RevtrAPIPortDebug = 8082

var RevtrGRPCPortProduction = 9999
// RevtrAPIPortDebug setting for the revtr api port in debug
var RevtrGRPCPortDebug = 9998

// AtlasPortProduction setting for the revtr grpc port in production
var AtlasPortProduction = 55000
// AtlasPortDebug setting for the revtr grpc port in debug
var AtlasPortDebug = 55001
// AtlasAPIPortDebug setting for the revtr grpc port in debug
var AtlasAPIPortDebug = 8083

// VPServicePortProduction setting for the revtr grpc port in production
var VPServicePortProduction = 45000 
// VPServicePortDebug setting for the revtr grpc port in debug
var VPServicePortDebug = 45001
// VPServiceAPIPortDebug setting for the revtr grpc port in debug
var VPServiceAPIPortDebug = 8084

// ControllerPortProduction setting for the revtr grpc port in production
var ControllerPortProduction = 4382
// ControllerPortDebug setting for the revtr grpc port in debug
var ControllerPortDebug = 9000 
// ControllerAPIPortDebug setting for the revtr grpc port in debug
var ControllerAPIPortDebug = 8085 

// PLControllerPortProduction setting for the revtr grpc port in production
var PLControllerPortProduction = 4380 
// PLControllerPortDebug setting for the revtr grpc port in debug
var PLControllerPortDebug = 9001 

// PLControllerPortProduction setting for the revtr grpc port in production
var ScamperPortProduction = 4381 
// PLControllerPortDebug setting for the revtr grpc port in debug
var ScamperPortDebug = 4299 

// ControllerPortProduction setting for the revtr grpc port in production
var RIPEAtlasControllerPortProduction = 4384 
// ControllerPortDebug setting for the revtr grpc port in debug
var RIPEAtlasControllerPortDebug = 9003 


// RankingServicePortProduction setting 
var RankingServicePortProduction = 49491 
// RankingServicePortDebug setting for the revtr grpc port in debug
var RankingServicePortDebug = 49492 

type RRVPsSelectionTechnique string
 
const(
	GlobalSetCover   RRVPsSelectionTechnique = "global_cover"
	IngressCover     RRVPsSelectionTechnique = "ingress_cover"
	DestinationCover RRVPsSelectionTechnique = "destination_cover"
	Random           RRVPsSelectionTechnique = "random_cover"
	OldRevtrCover    RRVPsSelectionTechnique = "old_revtr_cover"
	Dummy         	 RRVPsSelectionTechnique = "dummy"
)

const (
	DateFormat = "2006-01-02 15:04:05"
)

// Maximum flying spoofed probes for the overall system
var MaximumFlyingSpoofed = 1000000

const (
	MinSport = 50000
	MaxSport = MinSport + 10000
)
