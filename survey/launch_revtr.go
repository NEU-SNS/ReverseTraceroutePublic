package survey

import (
	"fmt"
	"strings"
	"sync"
	"time"

	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"golang.org/x/net/context"

	// "github.com/NEU-SNS/ReverseTraceroute/util"
	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	revtrtypes "github.com/NEU-SNS/ReverseTraceroute/revtr/types"
)

func tryFetchReverseTracerouteResults(ctx context.Context, 
	revtrClient revtr.RevtrClient, authKey string, batchId uint32, minimumRunningRevtrsBeforeReturn int) ([]revtr.RevtrStatus, error) {
	revtrReq := revtr.GetRevtrBatchStatusReq{BatchId : batchId, Auth:authKey}
	revtrResp, err:= revtrClient.GetRevtrBatchStatus(ctx ,&revtrReq)
	if err != nil{
		return nil, err
	}
	reverseTracerouteResults := revtrResp.RevtrsStatus
	revtrRunning := 0
	for _, reverseTracerouteStatus := range reverseTracerouteResults{
		if reverseTracerouteStatus == revtr.RevtrStatus_RUNNING{
			revtrRunning += 1
			if revtrRunning > minimumRunningRevtrsBeforeReturn {
				log.Debugf(fmt.Sprintf("Trying to fetch results for batch id %d, status %s", batchId, 
				revtr.RevtrStatus_name[int32(reverseTracerouteStatus)]))
				return nil, nil
			}
		}
	}
	return reverseTracerouteResults, nil
}

func SendAndWaitReverseTracerouteWorker(
	reverseTracerouteClient revtr.RevtrClient,
	controllerClient controller.ControllerClient,
	results chan <- uint32,
	jobs <-chan [] *revtr.RevtrMeasurement,
	parameters MeasurementSurveyParameters,
) {
	// Create some Reverse Traceroutes measurements via gRPC
	for reverseTracerouteMeasurements := range jobs {
		reverseTraceroutes := revtr.RunRevtrReq{
			Revtrs : reverseTracerouteMeasurements,
			Auth: parameters.AuthKey,
			CheckDB: parameters.CheckDB,
		}
		log.Debugf("Starting %d reverse traceroute measurements", len(reverseTracerouteMeasurements))

		reverseTraceroutesResp, err := reverseTracerouteClient.(revtr.RevtrClient).RunRevtr(context.Background(), &reverseTraceroutes)
		if err != nil{
			if strings.Contains(err.Error(), revtrtypes.ErrNoRevtrsToRun.Error()) {
				// Source is probably not valid
				results <- 0 // Dummy value to decrement number of on flying jobs 
				continue
			} else {
				panic(err)
			}
		
		}	
	
		batchID := reverseTraceroutesResp.GetBatchId()
		// Wait that the current batch has finished before sending another batch
		// Start ticking the url every x seconds to find out about results
		ticker := time.NewTicker(time.Duration(10 * time.Second))
		start := time.Now()
		done := false
		for !done {
			select {
			case _ = <-ticker.C:
				if time.Now().Sub(start) < time.Duration(parameters.FirstFetchResult) * time.Second  {
					log.Infof("Starting to try to fetch measurements soon...")
					break
				}
				reverseTracerouteResults, err := tryFetchReverseTracerouteResults(context.Background(), reverseTracerouteClient.(revtr.RevtrClient), 
				parameters.AuthKey, batchID, parameters.MinimumRunningRevtrsBeforeNextBatch)
				if err != nil {
					panic(err)
				}
				if reverseTracerouteResults != nil{
					t := time.Now()
					elapsed := t.Sub(start).Seconds()
					log.Debugf("Took %f seconds to complete %d reverse traceroutes\n", elapsed, len(reverseTracerouteResults))
					log.Infof("curl -H \"Api-Key: 7vg3XfyGIJZL92Ql\" -H \"Revtr-Key: 7vg3XfyGIJZL92Ql\" http://localhost:%d/api/v1/revtr?batchid=%d\n", 
					env.RevtrAPIPortDebug, batchID)
					// log.Infof("Reverse traceroute number %d finished", batchID)
					if parameters.IsCheckReversePath  {
						// Now get the reverse traceroute IDs corresponding to this batch
						revtrMetaResp, err := reverseTracerouteClient.GetRevtrMetaOnly(context.Background(), &revtr.GetRevtrMetaOnlyReq{
							BatchId: batchID,
							Auth: parameters.AuthKey,
						})
						if err != nil {
							panic(err)
						}
						
						var wg sync.WaitGroup
						for _, revtrMeta := range revtrMetaResp.RevtrsMeta {
							wg.Add(1)
							
							// Check if we already have a traceroute measurement for the same day
							source := revtrMeta.Src
							destination := revtrMeta.Dst
							
							// Run a traceroute from the destination to check the results 
							go func()  {
								defer wg.Done()
								checkTraceroutes := [] *dm.TracerouteMeasurement{}
								if !CheckTracerouteInDB(destination, source) {
									// Find the probe that is corresponding to the destination
									probeID := parameters.MappingIPProbeID[destination]
									tracerouteMeasurement := dm.BuildRIPETraceroute(source, []int{probeID},
									parameters.RIPEAPIKey)
									checkTraceroutes = append(checkTraceroutes, &tracerouteMeasurement)
								}

								// Associate the reverse traceroute with the one in the db
								


								// stream, err := controllerClient.Traceroute(context.Background(), &dm.TracerouteArg{
								// 	Traceroutes: checkTraceroutes,
								// })
								// if err != nil {
								// 	log.Error(err)
								// }
								// for {
								// 	tr, err := stream.Recv()
								// 	if err == io.EOF {
								// 		break
								// 	}
								// 	if err != nil {
								// 		log.Error(err)
										
								// 	}
								// 	// Associate the traceroute with its reverse traceroute ID. 
								// 	tracerouteID := tr.Id
								// 	revtrID := revtrMeta.Id
								// 	reverseTracerouteClient.UpdateRevtr(context.Background(), &revtr.UpdateRevtrReq{
								// 		Auth: parameters.AuthKey,
								// 		RevtrId: revtrID, 
								// 		TracerouteId: tracerouteID, 
								// 	})

									// TODO: Check that the traceroutes are done in RIPEAtlas.
								// }
							} ()
							
							wg.Wait()
						}
					}
					results <- batchID
					ticker.Stop()
					done = true
				}
			}	
		}
	}
}

func SendAndWaitReverseTraceroutes(reverseTracerouteClient interface{}, 
	controllerClient controller.ControllerClient, 
	batch *[]SourceDestinationPairIndex,
	sources *[] string,  destinations *[]string, parameters MeasurementSurveyParameters) [] uint32{
	
	// In order to use our pool of workers we need to send
	// them work and collect their results. We make 2
	// channels for this.
	
	// Two implementations to avoid overloading the system with 10K concurrent connections on services
	hasToDoSingleBatch := true
	numJobs := 0
	channelSize := 0
	if  parameters.NParallelRevtr < len(*batch) {
		hasToDoSingleBatch = false
		numJobs = parameters.NParallelRevtr
		channelSize = len(*batch)
	} else {
		numJobs = 1
		channelSize = 1
	}
	
	results := make(chan uint32, channelSize)
	jobs := make(chan [] *revtr.RevtrMeasurement, numJobs)
	defer close(results)
	// Start numJobs workers
	for j := 0; j < numJobs; j++ {
		go SendAndWaitReverseTracerouteWorker(reverseTracerouteClient.(revtr.RevtrClient), controllerClient,
		results, jobs,
		parameters)
	}

	go func() {
		reverseTracerouteMeasurements := [] *revtr.RevtrMeasurement{}
		for _, sourceDestinationPair:= range *batch{
			// TODO Check if the source and destination are valid 
			source := (*sources)[sourceDestinationPair.Source]
			destination := (*destinations)[sourceDestinationPair.Destination] 
			// fmt.Printf("%s, %s\n", source, destination)
			if source == "" || destination == ""{
				continue
			}
			
			reverseTracerouteMeasurements = append(reverseTracerouteMeasurements, &revtr.RevtrMeasurement{
				Src: source,
				Dst: destination,
				// Staleness is for the staleness of the atlas
				RrVpSelectionAlgorithm: string(parameters.RankingTechnique),
				UseTimestamp: parameters.UseTimestamp, 
				UseCache : parameters.UseCache,
				AtlasOptions: &revtr.AtlasOptions{
					UseAtlas: parameters.AtlasOptions.UseAtlas,
					UseRrPings: parameters.AtlasOptions.UseRRPings,
					IgnoreSource: parameters.AtlasOptions.IgnoreSource,
					IgnoreSourceAs: parameters.AtlasOptions.IgnoreSourceAS,
					StalenessBeforeRefresh: parameters.AtlasOptions.StalenessBeforeRefresh,
					Platforms: parameters.AtlasOptions.Platforms,
					Staleness: parameters.AtlasOptions.Staleness,
				},
				CheckDestBasedRoutingOptions: &revtr.CheckDestBasedRoutingOptions{
					CheckTunnel: parameters.CheckDestinationBasedRouting.CheckTunnel,
				},
				HeuristicsOptions: &revtr.RRHeuristicsOptions {
					UseDoubleStamp: parameters.RRHeuristicsOptions.UseDoubleStamp, 
				},
				MaxSpoofers: uint32(parameters.MaxSpoofers),
				Label: parameters.Label,
				IsRunForwardTraceroute: parameters.IsRunForwardTraceroute,
				IsRunRttPings: parameters.IsRunRTTPings,
			})
			if !hasToDoSingleBatch {
				if !parameters.IsDryRun {
					jobs <- reverseTracerouteMeasurements
					reverseTracerouteMeasurements = nil		
				} else {
					// Hack to get the evaluation reverse traceroutes
					print(fmt.Sprintf("%s,%s\n", source, destination))	
					results<-0
				}
				
			}
		}	
		if hasToDoSingleBatch {
			if !parameters.IsDryRun {
				jobs <- reverseTracerouteMeasurements
			} else {
				// Hack to get the evaluation reverse traceroutes
				for _, revtr := range(reverseTracerouteMeasurements) {
					print(fmt.Sprintf("%s,%s\n", revtr.Src, revtr.Dst))	
				}
				results<-0
				
			}
			
		}
		close(jobs)
	}()
	// Finally we collect all the results of the work.
	// This also ensures that the worker goroutines have
	// finished. An alternative way to wait for multiple
	// goroutines is to use a [WaitGroup](waitgroups).
	batchIds := [] uint32{}
	for a := 0; a < channelSize; a++ {
		id := <-results
		batchIds = append(batchIds, id)
	}
	
	return batchIds
}