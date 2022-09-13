package filters

import "github.com/NEU-SNS/ReverseTraceroute/vpservice/types"

// ComposeRRFilter composes the given RRFilters into a single RRFilter
// they are run in the order they are given
func ComposeRRFilter(fs ...RRFilter) RRFilter {
	return func(rrvps []types.RRVantagePoint) []types.RRVantagePoint {
		curr := rrvps
		for _, filter := range fs {
			curr = filter(curr)
		}
		return curr
	}
}

// ComposeTSFilter composes the given TSFilters into a single TSFilter
// they are run in the order they are given
func ComposeTSFilter(fs ...TSFilter) TSFilter {
	return func(rrvps []types.TSVantagePoint) []types.TSVantagePoint {
		curr := rrvps
		for _, filter := range fs {
			curr = filter(curr)
		}
		return curr
	}
}
