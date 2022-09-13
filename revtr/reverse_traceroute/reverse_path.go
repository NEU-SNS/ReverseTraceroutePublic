package reversetraceroute

import (
	"fmt"

	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
)

// ReversePath is a reverse path
type ReversePath struct {
	Src, Dst string
	Path     *[]Segment
}

func (rp *ReversePath) len() int {
	return len(*rp.Path)
}

// NewReversePath creates a reverse path
func NewReversePath(src, dst string, path []Segment) *ReversePath {
	ret := ReversePath{
		Src: src,
		Dst: dst,
	}
	if len(path) == 0 {
		// No measurement ID as it only contains the destination
		ret.Path = &[]Segment{NewDstRevSegment([]string{dst}, src, dst, 0, false)} 
	} else {
		ret.Path = &path
	}
	return &ret
}

// Clone clones a ReversePath
func (rp *ReversePath) Clone() *ReversePath {
	ret := ReversePath{
		Src:  rp.Src,
		Dst:  rp.Dst,
		Path: new([]Segment),
	}
	for _, seg := range *rp.Path {
		*ret.Path = append(*ret.Path, seg.Clone())
	}
	return &ret
}

// Hops gets the hops from each segment
func (rp *ReversePath) Hops() []string {
	var segs [][]string
	for _, p := range *rp.Path {
		segs = append(segs, p.Hops())
	}
	var hops []string
	for _, seg := range segs {
		for _, h := range seg {
			hops = append(hops, h)
		}
	}
	return hops
}

// Length returns the length of all the segments
func (rp *ReversePath) Length() int {
	var length int
	for _, seg := range *rp.Path {
		length += seg.Length(false)
	}
	return length
}

// LastHop gets the last hop of the last segment
func (rp *ReversePath) LastHop() string {
	return rp.LastSeg().LastHop()
}

// LastSeg gets the last segment
func (rp *ReversePath) LastSeg() Segment {
	return (*rp.Path)[rp.len()-1]
}

// Pop pops a segment off of the path
func (rp *ReversePath) Pop() Segment {
	length := rp.len()
	last := (*rp.Path)[length-1]
	*rp.Path = (*rp.Path)[:length-1]
	return last
}

// Reaches returns weather or not the last segment reaches
func (rp *ReversePath) Reaches(cm clustermap.ClusterMap) bool {
	return rp.LastSeg().Reaches(cm)
}

// SymmetricAssumptions returns the number of symmetric assumptions
func (rp *ReversePath) SymmetricAssumptions() int {
	var total int
	for _, seg := range *rp.Path {
		total += seg.SymmetricAssumptions()
	}
	return total
}

// Add adds a segment to the path
func (rp *ReversePath) Add(s Segment) {
	*rp.Path = append(*rp.Path, s)
}

func (rp *ReversePath) String() string {
	return fmt.Sprintf("RevPath_D%s_S%s_%v", rp.Dst, rp.Src, rp.Path)
}
