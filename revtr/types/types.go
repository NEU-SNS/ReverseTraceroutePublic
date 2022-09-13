package types

import (
	"fmt"
	"time"
)

// AdjacencySource is the interface for something that provides adjacnecies
type AdjacencySource interface {
	GetAdjacenciesByIP1(uint32) ([]Adjacency, error)
	GetAdjacenciesByIP2(uint32) ([]Adjacency, error)
	GetAdjacencyToDestByAddrAndDest24(uint32, uint32) ([]AdjacencyToDest, error)
}

// ClusterSource is the interface for something that provides cluster data
type ClusterSource interface {
	GetClusterIDByIP(uint32) (int, error)
	GetIPsForClusterID(int) ([]uint32, error)
}

// Config represents the config options for the revtr service.
type Config struct {
	RootCA   *string `flag:"root-ca"`
	CertFile *string `flag:"cert-file"`
	KeyFile  *string `flag:"key-file"`
	WebsiteSourceFile  *string `flag:"website-source-file"`
	Staleness *int   `flag:"staleness"` // window to refresh the traceroutes, in minutes. 
}

// NewConfig creates a new config struct
func NewConfig() Config {
	return Config{
		RootCA:   new(string),
		CertFile: new(string),
		KeyFile:  new(string),
		WebsiteSourceFile: new(string),
	}
}

// Adjacency represents the adjacency of 2 ips
type Adjacency struct {
	IP1, IP2 uint32
	Cnt      uint32
}

// AdjacencyToDest is ...
type AdjacencyToDest struct {
	Dest24   uint32
	Address  uint32
	Adjacent uint32
	Cnt      uint32
}

// Cache is the interface for the cache that revtr uses
type Cache interface {
	Set(string, interface{}, time.Duration)
	Get(string) (interface{}, bool)
}

// SpoofRRHops is the structure related to spoof record routes matched with their measurement IDs.
type SpoofRRHops struct {
	Hops []string
	VP   string
	MeasurementID int64
	FromCache bool
}

// RRHops is the structure related to record route matched with their measurement IDs.
type RRHops struct {
	Hops [] string
	MeasurementID int64
	FromCache bool
}

type DestinationBasedRoutingViolation int

const (
	NoCheck DestinationBasedRoutingViolation = iota
	NoViolation 
	Tunnel
	LoadBalancing 
)

type SrcDstPair struct {
	Src uint32
	Dst uint32
}

var  (

	// ErrNoRevtrsToRun is returned when there are no revtrs given in a batch
	ErrNoRevtrsToRun = fmt.Errorf("no runnable revtrs in the batch")
	// ErrIncorrectSource is returned when trying to run a revtr to a not valid source (either unknown or inactive)
	ErrIncorrectSource = fmt.Errorf("Not a valid source")
	// ErrConnectFailed is returned when connecting to the services failed
	ErrConnectFailed = fmt.Errorf("could not connect to services")
	// ErrFailedToCreateBatch is returned when creating the batch of revtrs fails
	ErrFailedToCreateBatch = fmt.Errorf("could not create batch")

	
	ErrUserNotFound = fmt.Errorf("No user found associated to this key")
	ErrTooManyMesaurementsToday = fmt.Errorf("Too many revtr in a day")
	ErrTooManyMesaurementsInParallel = fmt.Errorf("Too many revtr in parallel")

)
