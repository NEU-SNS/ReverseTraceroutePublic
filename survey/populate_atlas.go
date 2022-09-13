package survey

import (
	"encoding/json"
	"fmt"
	"sync"

	atlas "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
)

func CreatePopulateAtlasClient() (revtr.RevtrClient, controller.ControllerClient, atlas.AtlasClient) {
	revtrClient, err := CreateReverseTracerouteClient("atlas/root.crt")  
	if err != nil {
		panic(err)
	}
	controllerClient, err := CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	atlasClient, err := CreateAtlasClient("atlas/root.crt")
	if err != nil {
		panic(err)
	}

	return revtrClient, controllerClient, atlasClient
}


type RIPEMeasurementResponse struct {
	Count int `json:"count,omitempty"`
  	Next string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Results [] RIPEMeasurementResponseResult `json:"result,omitempty"` 
}

type RIPEMeasurementResponseResult struct {

}

func PopulateAtlasPublicRIPEAtlasMeasurement(client *controller.ControllerClient, sources [] string) (error) {

	// This function fetches public traceroutes from RIPE Atlas probes to sources and put it in the DB
	wg := sync.WaitGroup {}

	for _, source := range sources[:]{
		wg.Add(1)
		go func (source string) {
			defer wg.Done()
			url := fmt.Sprintf("https://atlas.ripe.net:443/api/v2/measurements/traceroute/?is_public=true&target_ip=%s", source)
			resp, err := QueryMeasurement(url, "")
			if err != nil {
				return 
			}
			ripeMeasurementResponse := RIPEMeasurementResponse{}
			err = json.NewDecoder(resp.Body).Decode(&ripeMeasurementResponse)
			if err != nil {
				return
			}
			// if ripeMeasurementResponse.Count != 0 {
				fmt.Printf("Results for %s: %d\n", source, ripeMeasurementResponse.Count)
			// }

		}(source)
	}
	wg.Wait()
	return nil
}
