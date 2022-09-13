package util

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/NEU-SNS/ReverseTraceroute/log"
)

func GetRIPEProbesIP(useSerialized bool, path string) ([] string, map[string] int) {
	log.Infof("Getting RIPE probes information...")
	destinations := [] string {}
	mappingIPProbeID := map[string]int {}
	probeResponse := [] ProbeResponse {}
	if useSerialized  {	
		// Open our jsonFile
		jsonFile, err := os.Open(path)
		// if we os.Open returns an error then handle it
		if err != nil {
			panic(err)
		}
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()
		byteValue, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(byteValue, &probeResponse)
		for _, probe := range(probeResponse) {
			if probe.Status.Name == "Connected" {
				mappingIPProbeID[probe.AddressV4] = probe.ID 
				destinations = append(destinations, probe.AddressV4)
			}
		}
	} else {
		probesUrl := "https://atlas.ripe.net/api/v2/probes/?status_name=Connected"
		next := &probesUrl
		
		for next != nil{
			probesRespJson, err := http.Get(*next)
			if err != nil {
				panic(err)
			}
			probesResp := ProbesResponse{}
			err = json.NewDecoder(probesRespJson.Body).Decode(&probesResp)
			if err != nil {
				panic(err)
			}
			for _, probeResp := range probesResp.ProbesResults{
				if probeResp.Status.Name == "Connected" && !probeResp.IsAnchor  && probeResp.AddressV4 != "" {
					destinations = append(destinations, probeResp.AddressV4)
					mappingIPProbeID[probeResp.AddressV4] = probeResp.ID
				}
				
			}
			next = probesResp.Next
		}
		// TODO Write the updated probes the json file 
	}
	
	log.Infof("Getting RIPE probes information... done")
	return destinations, mappingIPProbeID
}

func GetRIPEAnchorsIP() ([] string, map[string]int) {
	destinations := [] string {}
	anchorsUrl := "https://atlas.ripe.net/api/v2/anchors"
	next := &anchorsUrl
	mappingIPProbeID := map[string]int {}
	for next != nil{
		anchorsRespJson, err := http.Get(*next)
		if err != nil {
			panic(err)
		}
		anchorsResp := AnchorsResponse{}
		err = json.NewDecoder(anchorsRespJson.Body).Decode(&anchorsResp)
		if err != nil {
			panic(err)
		}
		for _, anchorResp := range anchorsResp.AnchorsResults{
			if !anchorResp.IsDisabled && anchorResp.IPv4 != "" {
				destinations = append(destinations, anchorResp.IPv4)
				mappingIPProbeID[anchorResp.IPv4] = anchorResp.ProbeID
			}
			
		}
		next = anchorsResp.Next
	}
	return destinations, mappingIPProbeID
}
