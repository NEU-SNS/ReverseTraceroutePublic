package filters

import (
	"sort"

	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
)

// RRFilter  is a function for prividing addition filtering the
// VPs returned from the VPProvider for RRSpoofing
type RRFilter func([]types.RRVantagePoint) []types.RRVantagePoint

// TSFilter is a function for providing additional filtering the
// VPs returned from the VPProvider for TSSpoofing
type TSFilter func([]types.TSVantagePoint) []types.TSVantagePoint

// OnePerSiteRR is a filter that returns rrvps with duplicate sites removed
func OnePerSiteRR(rrvps []types.RRVantagePoint) []types.RRVantagePoint {
	var ops []types.RRVantagePoint
	filter := make(map[string]types.RRVantagePoint)
	for _, vp := range rrvps {
		filter[vp.Site] = vp
	}
	for _, vp := range filter {
		ops = append(ops, vp)
	}
	return ops
}

// MakeRRDistanceFilter returns an RRFilter that filters out any RRVantagePoints
// that are in the interval [min, max]
func MakeRRDistanceFilter(min, max uint32) RRFilter {
	return func(rrvps []types.RRVantagePoint) []types.RRVantagePoint {
		var final []types.RRVantagePoint
		for _, vp := range rrvps {
			if vp.Dist >= min && vp.Dist <= max {
				continue
			}
			final = append(final, vp)
		}
		return final
	}
}

// OnePerSiteTS is a filter that returns rrvps with duplicate sites removed
func OnePerSiteTS(tsvps []types.TSVantagePoint) []types.TSVantagePoint {
	var ops []types.TSVantagePoint
	filter := make(map[string]types.TSVantagePoint)
	for _, vp := range tsvps {
		filter[vp.Site] = vp
	}
	for _, vp := range filter {
		ops = append(ops, vp)
	}
	return ops
}

type rrvpsDist []types.RRVantagePoint

func (rrdist rrvpsDist) Len() int           { return len(rrdist) }
func (rrdist rrvpsDist) Swap(i, j int)      { rrdist[i], rrdist[j] = rrdist[j], rrdist[i] }
func (rrdist rrvpsDist) Less(i, j int) bool { return rrdist[i].Dist < rrdist[j].Dist }

// OrderRRDistanceFilter sorts rrvps by distance desc
func OrderRRDistanceFilter(rrvps []types.RRVantagePoint) []types.RRVantagePoint {
	sort.Sort(rrvpsDist(rrvps))
	return rrvps
}
