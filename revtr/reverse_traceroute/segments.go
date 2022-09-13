package reversetraceroute

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	atlaspb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	stringutil "github.com/NEU-SNS/ReverseTraceroute/util/string"
)

const (
	dstRevSegment = iota + 1
	dstSymRevSegment
	trToSrcRevSegment
	trToSrcRevSegmentBetween
	rrRevSegment
	spoofRRRevSegment
	tsAdjRevSegment
	spoofTSAdjRevSegment
	spoofTSAdjRevSegmentTSZero
	spoofTSAdjRevSegmentTSZeroDoubleStamp
)

// Segment is the interface for a segment
type Segment interface {
	Hops() []string
	LastHop() string
	Length(bool) int
	Reaches(clustermap.ClusterMap) bool
	SymmetricAssumptions() int
	Order(Segment, clustermap.ClusterMap) int
	RemoveHops([]string, clustermap.ClusterMap) error
	Clone() Segment
	RemoveAt(int)
	Type() int
	GetMeasurementID() int64
	FromCache() bool
}

// RevSegment is a segment in a reverse path
type RevSegment struct {
	Segment  []string
	Src, Hop string
	MeasurementID int64 // Measurement ID corresponding to this RevSegment
	IsFromCache bool
}

// Type returns the type of the segment
func (rv *RevSegment) Type() int {
	return 0
}

// GetMeasurementID returns the measurement ID corresponding to this RevSegment.
func (rv * RevSegment) GetMeasurementID() int64 {
	return rv.MeasurementID
}

func (rv * RevSegment) FromCache() bool {
	return rv.IsFromCache
}

// RemoveAt removes a hop at a given index
func (rv *RevSegment) RemoveAt(idx int) {
	rv.Segment, rv.Segment[len(rv.Segment)-1] = append(rv.Segment[:idx], rv.Segment[idx+1:]...), ""
}

func (rv *RevSegment) clone() *RevSegment {
	ret := RevSegment{
		Src: rv.Src,
		Hop: rv.Hop,
		MeasurementID: rv.MeasurementID,
		IsFromCache: rv.IsFromCache,
	}
	ret.Segment = append(ret.Segment, rv.Segment...)
	return &ret
}

// Clone is for the interface
func (rv *RevSegment) Clone() Segment {
	return rv.clone()
}

func (rv *RevSegment) String() string {
	return fmt.Sprintf("RevSegment_%v_S%s_H%s", rv.Segment, rv.Src, rv.Hop)
}

// Hops gets the hops of the segment
func (rv *RevSegment) Hops() []string {
	return rv.Segment
}

// SetHop sets the hop for the segment
func (rv *RevSegment) SetHop(hop string) {
	rv.Hop = hop
}

func rIndex(ss []string, s string, cm clustermap.ClusterMap) int {
	index := -1
	for i, st := range ss {
		if cm.Get(s) == cm.Get(st) {
			index = i
		}
	}
	return index
}

// RemoveHops removes the given hops from the segment
func (rv *RevSegment) RemoveHops(toDel []string, cm clustermap.ClusterMap) error {
	var noZeros []string
	segAsSet := stringutil.StringSet(rv.Segment)
	for _, ip := range toDel {
		if ip != "0.0.0.0" {
			noZeros = append(noZeros, ip)
		}
	}
	common := segAsSet.Union(stringutil.StringSet(noZeros))
	if len(common) > 0 {
		log.Debug("Removing loopy hops ", common, " from ", rv.Segment)
		maxIndex := -1
		for _, h := range common {
			tmp := rIndex(rv.Segment, h, cm)
			if tmp > maxIndex {
				maxIndex = tmp
			}
		}
		log.Debug("MaxIndex: ", maxIndex)
		if maxIndex == len(rv.Segment) {
			rv.Segment = nil
		} else {
			rv.Segment = rv.Segment[maxIndex+1 : len(rv.Segment)]
		}
	}
	log.Debug("Segment after removing loopy hops: ", rv.Segment)
	common = stringutil.StringSet(rv.Segment).Union(stringutil.StringSet(toDel))
	if len(common) > 0 {
		return fmt.Errorf("Still a loop, %v, %v, %v", toDel, rv.Segment, common)
	}
	return nil
}

// RemoveLocalHops removes all private hops
func (rv *RevSegment) RemoveLocalHops() {
	var ns []string
	for _, h := range rv.Segment {
		ip := net.ParseIP(h)
		if ip != nil && ip.IsGlobalUnicast() {
			ns = append(ns, h)
		}
	}
	rv.Segment = ns
}

// SymmetricAssumptions returns the number of symmetric assumptions in the segment
func (rv *RevSegment) SymmetricAssumptions() int {
	return 0
}

// Length returns the length of the segment. If excNullHops is given,
// hops that are 0.0.0.0 aren't included
func (rv *RevSegment) Length(excNullHops bool) int {
	if excNullHops {
		var length int
		for _, h := range rv.Segment {
			if h != "0.0.0.0" {
				length++
			}
		}
		return length
	}
	return len(rv.Segment)
}

// LastHop returns the last hop in the segment
// returns "" if the segment is empty
func (rv *RevSegment) LastHop() string {
	if len(rv.Segment) == 0 {
		return ""
	}
	return rv.Segment[len(rv.Segment)-1]
}

// Reaches returns true if the revsegment reaches the src
func (rv *RevSegment) Reaches(cm clustermap.ClusterMap) bool {
	return rv.LastHop() == rv.Src || cm.Get(rv.LastHop()) == cm.Get(rv.Src)
}

// Order ...
func (rv *RevSegment) Order(b Segment, cm clustermap.ClusterMap) int {
	if reflect.TypeOf(rv) == reflect.TypeOf(&DstSymRevSegment{}) &&
		reflect.TypeOf(b) != reflect.TypeOf(&DstSymRevSegment{}) {
		return -1
	} else if reflect.TypeOf(b) == reflect.TypeOf(&DstSymRevSegment{}) &&
		reflect.TypeOf(rv) != reflect.TypeOf(&DstSymRevSegment{}) {
		return 1
	} else if stringutil.StringArrayEquals(rv.Segment, b.Hops()) {
		return 0
	} else if rv.LastHop() == b.LastHop() {
		if rv.Length(true) != b.Length(true) {
			if rv.Length(true) < b.Length(true) {
				return -1
			}
			if rv.Length(true) == b.Length(true) {
				return 0
			}
			return 1
		}
		return stringutil.OrderStringArray(rv.Segment, b.Hops())
	} else if rv.Reaches(cm) {
		return 1
	} else if b.Reaches(cm) {
		return -1
	} else if rv.Length(true) != b.Length(true) {
		ll := rv.Length(true)
		rl := b.Length(true)
		if ll < rl {
			return -1
		}
		if ll > rl {
			return 1
		}
		return 0
	}
	return strings.Compare(rv.LastHop(), b.LastHop())
}

// NewRevSegment creates a new RevSegment
func NewRevSegment(segment []string, src, hop string, measurementID int64, fromCache bool) *RevSegment {
	ret := RevSegment{
		Src:     src,
		Hop:     hop,
		Segment: segment,
		MeasurementID: measurementID,
		IsFromCache: fromCache,
	}
	for i, h := range ret.Segment {
		if !net.ParseIP(h).IsGlobalUnicast() {
			ret.Segment[i] = "0.0.0.0"
		}
	}
	return &ret
}

// DstRevSegment is when the reverse hop was verified by virtue of being the destination
type DstRevSegment struct {
	*RevSegment
}

// Type satisfies the Segment Interface
func (d *DstRevSegment) Type() int {
	return dstRevSegment
}

// NewDstRevSegment creates a new DstRevSegment
func NewDstRevSegment(segment []string, src, hop string, measurementID int64, fromCache bool) *DstRevSegment {
	ret := DstRevSegment{
		RevSegment: NewRevSegment(segment, src, hop, measurementID, fromCache),
	}
	return &ret
}

func (d *DstRevSegment) String() string {
	return fmt.Sprintf("%s_Dst", d.RevSegment.String())
}

// Clone is for the Segment interface
func (d *DstRevSegment) Clone() Segment {
	return d.clone()
}

func (d *DstRevSegment) clone() *DstRevSegment {
	ret := &DstRevSegment{
		RevSegment: d.RevSegment.clone(),
	}
	return ret
}

// DstSymRevSegment when the reverse hop is
type DstSymRevSegment struct {
	*RevSegment
	tr      []string
	numHops int
}

// Type ...
func (d *DstSymRevSegment) Type() int {
	return dstSymRevSegment
}

func (d *DstSymRevSegment) clone() *DstSymRevSegment {
	ret := DstSymRevSegment{
		RevSegment: d.RevSegment.clone(),
		numHops:    d.numHops,
	}
	ret.tr = append(ret.tr, d.tr...)
	return &ret
}

// Clone is for the interface
func (d *DstSymRevSegment) Clone() Segment {
	return d.clone()
}

func (d *DstSymRevSegment) String() string {
	return fmt.Sprintf("%s_AssumeSym", d.RevSegment.String())
}

// This mimics the functionality of the static method select_nonzero_hops
func ndsrsSelectNonzeroHops(tr []string, hops int, hopsToIgnore []string) []string {
	log.Debugf("Selecting %d non-zero hops from %v, ignoring %v", hops, tr, hopsToIgnore)
	var i, found int
	for found < hops && i < len(tr) {
		if tr[i] != "0.0.0.0" && (!stringutil.InArray(hopsToIgnore, tr[i])) {
			found++
		} else if stringutil.InArray(hopsToIgnore, tr[i]) {
			log.Debug("Skipping Deadend ", tr[i])
		}
		i++
	}
	if found == hops {
		log.Debug("Rev Seg is ", tr[:i])
		return tr[:i]
	}
	log.Debug("Only able to find ", found)
	lastValidHop := len(tr) - 1
	for tr[lastValidHop] == "0.0.0.0" {
		lastValidHop--
	}
	return tr[:lastValidHop+1]
}

// NewDstSymRevSegment creates a new NewDstSymRevSegment
// tr is an array of hops along the forward path, not including the source
// numhops is number of nonzero hops to assume
// even if we are passing in the array of hops to store as the segment
// still need to include numhops, since we don't know how many of those are being ignored
// hop to ignore does no persist
func NewDstSymRevSegment(src, hop string, tr []string, numhops int, hopsToIgnore []string,
	measurementID int64, fromCache bool) *DstSymRevSegment {
	ntr := append([]string{src}, tr[:len(tr)-1]...)
	log.Debug("New TR: ", ntr)
	rev := stringutil.StringSliceReverse(ntr)
	log.Debug("The reversed slice is: ", rev)
	segment := ndsrsSelectNonzeroHops(rev, numhops, hopsToIgnore)
	log.Debug("The segment is: ", segment)
	ret := DstSymRevSegment{
		tr:         tr,
		numHops:    numhops,
		RevSegment: NewRevSegment(segment, src, hop, measurementID, fromCache),
	}
	return &ret
}

// AddHop ...
// assume an addition hop is symmetric
// keeps going until finding one that isn't 0
// note: right now, this starts from scratch every time and counts hops,
// replacing the curring segments with a new one.
func (d *DstSymRevSegment) AddHop(hopsToIgnore []string) error {
	tr := append([]string{d.Src}, d.tr[:len(d.tr)-1]...)
	rev := stringutil.StringSliceReverse(tr)
	d.numHops++
	d.Segment = ndsrsSelectNonzeroHops(rev, d.numHops, hopsToIgnore)
	return nil
}

// TRtoSrcRevSegment is a ....
type TRtoSrcRevSegment struct {
	*RevSegment
	IntersectHopTypes [] atlaspb.IntersectHopType
}

// Type ...
func (d *TRtoSrcRevSegment) Type() int {
	return trToSrcRevSegment
}


func (d *TRtoSrcRevSegment) clone() *TRtoSrcRevSegment {
	ret := TRtoSrcRevSegment{
		RevSegment: d.RevSegment.clone(),
	}
	ret.IntersectHopTypes = append(ret.IntersectHopTypes, d.IntersectHopTypes...)
	return &ret
}

// Clone is for the interface
func (d *TRtoSrcRevSegment) Clone() Segment {
	return d.clone()
}



// NewTrtoSrcRevSegment creates a new TRtoSrcRevSegment
func NewTrtoSrcRevSegment(segment []string, intersectHopTypes [] atlaspb.IntersectHopType, src string, hop string, measurementID int64) *TRtoSrcRevSegment {
	ret := TRtoSrcRevSegment{
		RevSegment: NewRevSegment(segment, src, hop, measurementID, false),
		IntersectHopTypes: intersectHopTypes,
	}
	return &ret
}

func (d *TRtoSrcRevSegment) String() string {
	return fmt.Sprintf("%s_TRtoSrc", d.RevSegment.String())
}

// RRRevSegment when the reverse hop was found with a non-spoofed RR probe
type RRRevSegment struct {
	*RevSegment
}

// Type ...
func (d RRRevSegment) Type() int {
	return rrRevSegment
}

// NewRRRevSegment creates a new RRRevSegment
func NewRRRevSegment(segment []string, src, hop string, measurementID int64, fromCache bool) *RRRevSegment {
	ret := RRRevSegment{
		RevSegment: NewRevSegment(segment, src, hop, measurementID, fromCache),
	}
	return &ret
}

func (d *RRRevSegment) clone() *RRRevSegment {
	ret := RRRevSegment{
		RevSegment: d.RevSegment.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *RRRevSegment) Clone() Segment {
	return d.clone()
}

func (d *RRRevSegment) String() string {
	return fmt.Sprintf("%s_RR", d.RevSegment.String())
}

// SpoofRRRevSegment when the reverse hop was found with a spoofed RR probe
type SpoofRRRevSegment struct {
	*RRRevSegment
	SpoofSource string
}

// Type ...
func (d *SpoofRRRevSegment) Type() int {
	return spoofRRRevSegment
}

func (d *SpoofRRRevSegment) clone() *SpoofRRRevSegment {
	ret := SpoofRRRevSegment{
		RRRevSegment: d.RRRevSegment.clone(),
		SpoofSource:  d.SpoofSource,
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofRRRevSegment) Clone() Segment {
	return d.clone()
}

func (d *SpoofRRRevSegment) String() string {
	return fmt.Sprintf("%s_SpfSrc", d.RRRevSegment.String())
}

// NewSpoofRRRevSegment creates a new SpoofRRRevSegment
func NewSpoofRRRevSegment(segment []string, src, hop, spfsrc string, measurementID int64, fromCache bool) *SpoofRRRevSegment {
	ret := SpoofRRRevSegment{
		RRRevSegment: NewRRRevSegment(segment, src, hop, measurementID, fromCache),
		SpoofSource:  spfsrc,
	}
	return &ret
}

// TSAdjRevSegment when the reverse hop identified as potentially adjacent to another hop,
// then verified using timestamp
type TSAdjRevSegment struct {
	*RevSegment
	LinuxBug bool
}

// Type ...
func (d *TSAdjRevSegment) Type() int {
	return tsAdjRevSegment
}

func (d *TSAdjRevSegment) clone() *TSAdjRevSegment {
	ret := TSAdjRevSegment{
		RevSegment: d.RevSegment.clone(),
		LinuxBug:   d.LinuxBug,
	}
	return &ret
}

// Clone is for the interface
func (d *TSAdjRevSegment) Clone() Segment {
	return d.clone()
}

func (d *TSAdjRevSegment) String() string {
	if d.LinuxBug {
		return fmt.Sprintf("%s_TSAdj_LinuxBug", d.RevSegment.String())
	}
	return fmt.Sprintf("%s_TSAdj", d.RevSegment.String())
}

// NewTSAdjRevSegment creates a new TSAdjRevSegment
func NewTSAdjRevSegment(segment []string, src, hop string, linuxbug bool, measurementID int64, fromCache bool) *TSAdjRevSegment {
	ret := TSAdjRevSegment{
		LinuxBug:   linuxbug,
		RevSegment: NewRevSegment(segment, src, hop, measurementID, fromCache),
	}
	return &ret
}

// SpoofTSAdjRevSegment is when the reverse hop was found with a spoofed TS probe
type SpoofTSAdjRevSegment struct {
	*TSAdjRevSegment
	SpoofSource string
}

// Type ...
func (d *SpoofTSAdjRevSegment) Type() int {
	return spoofTSAdjRevSegment
}

func (d *SpoofTSAdjRevSegment) clone() *SpoofTSAdjRevSegment {
	ret := SpoofTSAdjRevSegment{
		TSAdjRevSegment: d.TSAdjRevSegment.clone(),
		SpoofSource:     d.SpoofSource,
	}
	return &ret
}

// Clone is for the iterface
func (d *SpoofTSAdjRevSegment) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegment) String() string {
	return fmt.Sprintf("%s_SpfSrc_%s", d.TSAdjRevSegment.String(), d.SpoofSource)
}

// NewSpoofTSAdjRevSegment creates a new SpoofTSAdjRevSegment
func NewSpoofTSAdjRevSegment(segment []string, src, hop, spfsrc string, linuxbug bool, measurementID int64, fromCache bool) *SpoofTSAdjRevSegment {
	ret := SpoofTSAdjRevSegment{
		SpoofSource:     spfsrc,
		TSAdjRevSegment: NewTSAdjRevSegment(segment, src, hop, linuxbug, measurementID, fromCache),
	}
	return &ret
}

// SpoofTSAdjRevSegmentTSZero is when the reverse hop was found with a spoofed TS probe
// and the destination is not a timestamper
// in general this means we found it by first issuing a spoofed probe and
// finding it was on either fwd or rev
// then issuing a non-spoofed and findit it wasn't on that
type SpoofTSAdjRevSegmentTSZero struct {
	*SpoofTSAdjRevSegment
}

// Type ...
func (d *SpoofTSAdjRevSegmentTSZero) Type() int {
	return spoofTSAdjRevSegmentTSZero
}

func (d *SpoofTSAdjRevSegmentTSZero) clone() *SpoofTSAdjRevSegmentTSZero {
	ret := SpoofTSAdjRevSegmentTSZero{
		SpoofTSAdjRevSegment: d.SpoofTSAdjRevSegment.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofTSAdjRevSegmentTSZero) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegmentTSZero) String() string {
	return fmt.Sprintf("%s_Dst0", d.SpoofTSAdjRevSegment.String())
}

// NewSpoofTSAdjRevSegmentTSZero creates a new SpoofTSAdjRevSegmentTSZero
func NewSpoofTSAdjRevSegmentTSZero(segment []string, src, hop, spfsrc string, linuxbug bool, measurementID int64, fromCache bool) *SpoofTSAdjRevSegmentTSZero {
	ret := SpoofTSAdjRevSegmentTSZero{
		SpoofTSAdjRevSegment: NewSpoofTSAdjRevSegment(segment, src, hop, spfsrc, linuxbug, measurementID, fromCache),
	}
	return &ret
}

// SpoofTSAdjRevSegmentTSZeroDoubleStamp is when
// the reverse hop was found with a spoofed TS probe
// and the destination is not a timestamper
// and we assume one was fwd, one rev
type SpoofTSAdjRevSegmentTSZeroDoubleStamp struct {
	*SpoofTSAdjRevSegmentTSZero
}

// Type ...
func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) Type() int {
	return spoofTSAdjRevSegmentTSZeroDoubleStamp
}

func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) clone() *SpoofTSAdjRevSegmentTSZeroDoubleStamp {
	ret := SpoofTSAdjRevSegmentTSZeroDoubleStamp{
		SpoofTSAdjRevSegmentTSZero: d.SpoofTSAdjRevSegmentTSZero.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) String() string {
	return fmt.Sprintf("%s_DblStamp", d.SpoofTSAdjRevSegmentTSZero.String())
}

// NewSpoofTSAdjRevSegmentTSZeroDoubleStamp creates a new SpoofTSAdjRevSegmentTSZeroDoubleStamp
func NewSpoofTSAdjRevSegmentTSZeroDoubleStamp(segment []string, src, hop, spfsrc string, linuxbug bool, measurementID int64, fromCache bool) *SpoofTSAdjRevSegmentTSZeroDoubleStamp {
	ret := SpoofTSAdjRevSegmentTSZeroDoubleStamp{
		SpoofTSAdjRevSegmentTSZero: NewSpoofTSAdjRevSegmentTSZero(segment, src, hop, spfsrc, linuxbug, measurementID, fromCache),
	}
	return &ret
}
