package survey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"github.com/NEU-SNS/ReverseTraceroute/util"

	// "github.com/NEU-SNS/ReverseTraceroute/util"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	// controllerApi "github.com/NEU-SNS/ReverseTraceroute/controller"
	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
)

func CreateRRSurveyClients() (revtr.RevtrClient, rk.RankingClient, controller.ControllerClient) {
	revtrClient, err := CreateReverseTracerouteClient("revtr/root.crt")  
	if err != nil {
		panic(err)
	}
	rankingClient, err := CreateRankingClient()
	if err != nil {
		panic(err)
	}
	controllerClient, err := CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}

	return revtrClient, rankingClient, controllerClient
}

	
type ResponsesCount struct{
	Count uint32 `json:"count,omitempty"`
}
func tryFetchMeasurementResults (resultUrl string, authKey string) error{
	tickerTime := 1
	timeout := 1600
	errorChan := make(chan error, 1)
	ticker := time.NewTicker(time.Duration(tickerTime) * time.Second) 
	defer ticker.Stop()
	go func(){
		tickerCountLimit := timeout / tickerTime
		tickerCount := 0
		previousNResponses := -1
		for {
			select {
			case t := <-ticker.C:
				resp, err := QueryMeasurement(resultUrl, authKey)
				if err != nil {
					log.Errorf(fmt.Sprintf("%s querying results at %s: %s", t, resultUrl, err))
					errorChan <- err
					return  
				}
				if resp != nil{
					// Check if the body is empty or not 
					responseCount := ResponsesCount{}
					err = json.NewDecoder(resp.Body).Decode(&responseCount)
					if err != nil {
						errorChan <- err
						return
					}
					if int(responseCount.Count) == previousNResponses && responseCount.Count != 0{
						log.Infof(fmt.Sprintf("Batch finished, fetched %d results at %s", responseCount.Count, resultUrl))
						errorChan <- nil
						return 
					} else {
						log.Debugf(fmt.Sprintf("Batch ongoing: fetched %d results at %s", responseCount.Count, resultUrl))
						previousNResponses = int(responseCount.Count)
					}
				}
				tickerCount += 1
				if tickerCount >= tickerCountLimit {
					log.Errorf(fmt.Sprintf("Timeout for fetching results at %s", resultUrl))
					errorChan <- nil
					return
				}

			}
		}
	}()
	err := <-errorChan
	return err
}


func sendBatch(client controller.ControllerClient, batch *[]SourceDestinationPairIndex, 
	sources *[] string, destinations * [] string, parameters MeasurementSurveyParameters) error {

	// currentTime := time.Now()
	// currentTime_str := currentTime.Format("2006_01_02")
	pms := []*dm.PingMeasurement {}
	for _, sourceDestinationPair:= range *batch{
		source := (*sources)[sourceDestinationPair.Source]
		destination := (*destinations)[sourceDestinationPair.Destination] 
		srcI, _ := util.IPStringToInt32(source)
		dstI, _ := util.IPStringToInt32(destination)
		pm := &dm.PingMeasurement{
			Src:        srcI,
			Dst:        dstI,
			Sport: 		strconv.Itoa(40000),
			Timeout:    int64(parameters.Timeout),
			Count:      strconv.Itoa(parameters.NPackets),
			CheckCache: false,
			FromRevtr:  true,
			RR:         parameters.IsRecordRoute,
			Label: 		parameters.Label, 
			SaveDb :    true, 
			Staleness:  24 * 60,
		}
		// pingMeasurement := controllerApi.Ping{
		// 	Src:        source,
		// 	Dst:        destination,
		// 	Timeout:    180,
		// 	Count:      3,
		// 	IsCheckCache: true,
		// 	IsRecordRoute:         false,
		// 	Label: 		label, 
		// }

		pms = append(pms, pm)
	}
	

	st, err := client.Ping(context.Background(), &dm.PingArg{
		Pings: pms,
	})
	if err != nil {
		return err
	}

	for {
		_, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// log.Error(err)
			return err
		}
	}
	return nil
}

func sendBatchAndWaitResults(client interface{}, controllerClient controller.ControllerClient,
	batch * []SourceDestinationPairIndex, sources * [] string, batchDestinations * []string,
	// batchIndex int,
	parameters MeasurementSurveyParameters,
	) [] uint32 {
	err := sendBatch(client.(controller.ControllerClient), batch, sources, batchDestinations,
	parameters)	
	if err != nil{
		log.Error(err)
	}
	// log.Infof(res + " batch index: %d", batchIndex)
	// Wait that the current batch has finished before sending another batch
	return nil 
}

func RRPingsRandomSources(client *controller.ControllerClient,
	nSources int, sources [] string, destinations [] string,
	authKey string, label string) (error) {
	/**
		This function takes a list of destinations, a list of sources and run RR from 5 random sources  
	**/
	
	parameters := MeasurementSurveyParameters{
		AuthKey: authKey,
		Label: label,
	}

	bitsForDestinations := math.Floor(math.Log2(float64(len(destinations)))) + 1
	permLength := uint32(math.Pow(2, bitsForDestinations))
	if bitsForDestinations == 32{
		permLength = uint32(math.Pow(2, bitsForDestinations) - 1)
	}

	for k := 0; k < nSources; k++ {
		cperm := CPermCreate(permLength)
		batch := []SourceDestinationPairIndex{}
		batchLen := 30000
		batchIndex := 0
		for i := uint32(0); i < permLength; i++{
			iPerm := CPermNext(cperm)
			destinationIndex := iPerm & uint32(math.Pow(2, bitsForDestinations) - 1)
			sourceIndex := uint32(rand.Intn(len(sources)))
			if sourceIndex >= uint32(len(sources)) {
				panic(errors.New("rand panicing"))
			}
			if destinationIndex >= uint32(len(destinations)) {
				continue
			}
			batch = append(batch, SourceDestinationPairIndex{Source: sourceIndex, Destination:destinationIndex})
			if len(batch) == batchLen {
				fmt.Printf("%d, %s, %s\n", i, sources[sourceIndex], destinations[destinationIndex])
				sendBatchAndWaitResults(client, nil, &batch, &sources, &destinations, parameters)
				// Reset batch
				batch = nil
				batchIndex += 1
			}
		}
		// Send last batch
		sendBatchAndWaitResults(client, nil,  &batch, &sources, &destinations, parameters)
		CPermDestroy(cperm)
	}

	
	return nil
}
func Pings(client *controller.ControllerClient, 
	sources [] string, destinations [] string, 
	authKey string, label string, nPacket int) (error) {
	// Create batches of 200000 measurements to not overload server memory
	// Randomize it to not trigger rate limiting either close to the source or on the destination
	parameters := MeasurementSurveyParameters{
		AuthKey: authKey,
		BatchLen: 200000,
		MaximumMeasurements: len(sources) * len(destinations),
		NPackets: nPacket,
		Label: label,
		IsRecordRoute: false,
		IsUseMultipleGoroutines: false,

		
	}
	RandomizeSourceDestinationAndSend(*client, nil, sources, destinations, parameters, sendBatchAndWaitResults)
	return nil
}

func RRPings(client *controller.ControllerClient, 
	sources [] string, destinations [] string, 
	authKey string, label string) (error) {
	// Create batches of 10000 measurements to not overload server memory
	// Randomize it to not trigger rate limiting either close to the source or on the destination
	batchLen := 1000000
	pps := 100
	// Number of sources 
	packetsPerSource := batchLen / len(sources)
	estimatedTime := packetsPerSource / pps // in seconds
	timeout := estimatedTime + 20 // with a 20 seconds margin
	parameters := MeasurementSurveyParameters{
		AuthKey: authKey,
		BatchLen: batchLen,
		MaximumMeasurements: len(sources) * len(destinations),
		Label: label,
		IsRecordRoute: true,
		NPackets: 1,
		Timeout: timeout,
	}
	RandomizeSourceDestinationAndSend(*client, nil, sources, destinations, parameters, sendBatchAndWaitResults)
	return nil
}