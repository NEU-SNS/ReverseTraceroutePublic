package types

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

// IntersectionQuery represents a request to the TRStore for an intersecting traceroute
type IntersectionQuery struct {
	Addr, Dst, Src uint32
	Alias          bool
	Stale          int64 // in Minutes
	UseAtlasRR     bool
	IgnoreSource   bool
	IgnoreSourceAS bool
	SourceAS       int 
	Platforms      []string
}

type IntersectionResponse struct {
	TracerouteID int64
	Path *pb.Path
	Src uint32 // Source of the traceroute 
	Platform string
	Timestamp int64 
	DistanceToSource uint32
}

type RefreshTracerouteIDChan struct  {
	TracerouteID int64
	Meta RefreshTracerouteMeta
	Channel chan RefreshStillIntersect 
}

type RefreshTracerouteMeta struct {
	Src uint32 // The source of the traceroute
	Dst uint32 // The revtr source (M-Lab vp)
	IntersectionIP uint32 
	DistanceToSource uint32
	Platform string 
	Timestamp time.Time
}

type RefreshStillIntersect struct {
	IsRefreshed bool 
	IsStillIntersect bool
}

type IntersectionResponseWithError struct {
	Ir IntersectionResponse
	Err error 
}

type TrIDRRHop struct {
	ID int64 // Corresponding traceroute
	PingID int64 // Corresponding ping
	RRHop uint32
	TrHop uint32
	RRHopIndex int
	Date time.Time
}

type ByTracerouteID []TrIDRRHop

func (a ByTracerouteID) Len() int           { return len(a) }
func (a ByTracerouteID) Less(i, j int) bool { 
	// if a[i].ID != a[j].ID {
		return a[i].ID < a[j].ID
	// }
}
func (a ByTracerouteID) Swap(i, j int)      {
	 a[i], a[j] = a[j], a[i] 
}


func RemoveDuplicateValues(slice []TrIDRRHop) []TrIDRRHop { 
    keys := make(map[TrIDRRHop]bool) 
    list := []TrIDRRHop{} 
  
    // If the key(values of the slice) is not equal 
    // to the already present value in new slice (list) 
    // then we append it. else we jump on another element. 
    for _, entry := range slice { 
        if _, value := keys[entry]; !value { 
            keys[entry] = true
            list = append(list, entry) 
        } 
    } 
    return list 
} 

// TRStore is the interface required by the
type TRStore interface {
	FindIntersectingTraceroute(IntersectionQuery) (IntersectionResponse, error)
	StoreAtlasTraceroute(*datamodel.Traceroute) (int64, error)
	StoreAtlasTracerouteBulk([]*datamodel.Traceroute) ([]int64, error)
	GetAtlasSources(uint32, time.Duration) ([]uint32, error)
	GetIPAliases([]uint32) (map[uint32]int64, map[int64][]uint32,  error)
	StoreTrRRHop([]TrIDRRHop) error 
	CheckIntersectingPath(int64, uint32) (*pb.Path, error)
	MarkTracerouteStale(uint32, int64, int64) error
	MarkTraceroutesStale([]int64) error
	MarkTracerouteStaleSource(uint32) error
	CompareOldAndNewTraceroute(uint32, int64, int64) (bool, bool, error)
	UpdateRRIntersectionTracerouteNonStale(oldID int64, newID int64) error
	MonitorDB() 
	GetAvailableHopAtlasPerSource() (map[uint32]map[uint32]struct{}, error)
}
