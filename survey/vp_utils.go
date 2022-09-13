package survey

import (
	"strings"

	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
)

func addSource(source * revtr.Source, sourcesSite *[] revtr.Source, k int, isStaging bool) {
	hostname := source.GetHostname()
	if isStaging {
		if strings.Contains(hostname, "staging"){ 
			if len(*sourcesSite) < k {
				*sourcesSite = append(*sourcesSite, *source)
			}
		}
	} else {
		if !strings.Contains(hostname, "staging"){ 
			if len(*sourcesSite) < k {
				*sourcesSite = append(*sourcesSite, *source)
			}
		}
	}
	
}

func GetKVPPerSite(sources []*revtr.Source, k int, isStaging bool) []revtr.Source {
	// This function selects one VP per site
	sourcesPerSite := make(map[string] *[] revtr.Source) 
	for _, source := range sources {
		site  := source.GetSite()
		if sourcesSite, ok := sourcesPerSite[site]; ok {
			addSource(source, sourcesSite, k, isStaging)	
		} else {
			sourcesSite := [] revtr.Source{}
			addSource(source, &sourcesSite, k, isStaging)
			sourcesPerSite[site] = &sourcesSite
		}
	} 
	newSources := [] revtr.Source{}
	for _, sourcesSite := range sourcesPerSite {
		newSources = append(newSources, *sourcesSite...)
	}
	return newSources
}