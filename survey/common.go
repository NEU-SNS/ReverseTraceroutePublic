package survey

import (
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	controller "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	rk "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpspb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"golang.org/x/net/context"
)

func CheckTracerouteInDB(source string, destination string) bool {
	// Check if we already have a traceroute from the destination to the source.
	return false
}

func GetVantagePointsPerSite() (map[string][]*vpspb.VantagePoint, []string) {
	vpService, err := CreateVPServiceClient("vpservice/root.crt")
	if err != nil {
		panic(err)
	}
	getVPsResp, err := vpService.GetVPs(context.Background(), &vpspb.VPRequest{})
	if err != nil{
		panic(err)
	}
	vantagePoints := getVPsResp.GetVps()
	vantagePointsPerSite := map[string][]*vpspb.VantagePoint{}
	for _, vp := range(vantagePoints) {
		site := vp.Site
		if _, ok := vantagePointsPerSite[site]; !ok {
			vantagePointsPerSite[site] = []*vpspb.VantagePoint {}
		} 
		// ipI, _ := util.IPStringToInt32(vp.Ip)
		vantagePointsPerSite[site] = append(vantagePointsPerSite[site], vp)
	}

	sites := [] string{}
	for site, _ := range(vantagePointsPerSite) {
		sites = append(sites, site)
	}
	return vantagePointsPerSite, sites
}

func RandomizeSourceDestinationAndSend(client interface{}, controllerClient controller.ControllerClient,
	sources [] string, destinations [] string,
	parameters MeasurementSurveyParameters, sendAndWaitFunc SendAndWaitFunc) error {
	
	lenDestinations := len(destinations)
	lenSources := len(sources)
	bitsForSources := math.Floor(math.Log2(float64(lenSources))) + 1
	bitsForDestinations := math.Floor(math.Log2(float64(lenDestinations))) + 1
	totalBits := bitsForSources + bitsForDestinations
	division := 1
	for totalBits > 32 {
		// Divide the number of destinations by division to make it work in 32 bit-permutation
		division *= 2
		lenDestinations /= division
		bitsForDestinations = math.Floor(math.Log2(float64(lenDestinations))) + 1
		totalBits = bitsForSources + bitsForDestinations
	}
	
	subDestinations :=[][] string {}
	for div := 0; div < len(destinations); div += len(destinations) / division {
		endIndex := div+len(destinations)/division
		if int(endIndex) > len(destinations) {
			// Add the last elements to the previous batch
			subDestinations[len(subDestinations) - 1] = 
			append(subDestinations[len(subDestinations)-1], destinations[div:len(destinations)]...)
		} else {
			subDestinations = append(subDestinations, 
				destinations[div:div+len(destinations)/division]) 
		}
	
	} 
	
	totalSent := 0
	hasReachedMeasurementLimit := false
	var wg sync.WaitGroup
	for k, sendDestinations := range subDestinations{
		permLength := uint32(math.Pow(2, totalBits))
		if totalBits == 32{
			permLength = uint32(math.Pow(2, totalBits) - 1)
		}
		cperm := CPermCreate(permLength)
		batch := []SourceDestinationPairIndex{}
		batchIndex := 0
		for i := uint32(0); i < permLength; i++{
			iPerm := CPermNext(cperm)
			sourceIndex := iPerm & (uint32(math.Pow(2, bitsForSources)) - 1) // Get the first bitsForSources of the number
			destinationIndex := (iPerm >> uint32(bitsForSources)) & uint32(math.Pow(2, bitsForDestinations) - 1) // Get the 
			if sourceIndex >= uint32(len(sources)) || destinationIndex >= uint32(len(sendDestinations)) {
				continue
			}
			
			batch = append(batch, SourceDestinationPairIndex{Source: sourceIndex, Destination:destinationIndex})
			if len(batch) == parameters.BatchLen {
				// So you can restart from where it stopped.
				// if batchIndex >= 90 {
					log.Infof("Sending subdestinations %d batch %d \n", k, batchIndex) 
					if parameters.BatchLen >= 10000 && parameters.IsUseMultipleGoroutines {
						batchCopy := make([]SourceDestinationPairIndex, len(batch))
						copy(batchCopy, batch)
						wg.Add(1)
						go func ()  {
							defer wg.Done()	
							sendAndWaitFunc(client, controllerClient, &batchCopy, &sources, &sendDestinations, parameters)
						} ()
						time.Sleep(180 * time.Second)
					} else {
						sendAndWaitFunc(client, controllerClient, &batch, &sources, &sendDestinations, parameters)
					}	
				// }
				// Reset batch
				batch = nil
				batchIndex ++
			}
			
			totalSent ++
			if totalSent >= parameters.MaximumMeasurements {
				hasReachedMeasurementLimit = true
				break
			}
		}
		// Send last batch
		batchCopy := []SourceDestinationPairIndex {}
		copy(batchCopy, batch)
		if parameters.BatchLen >= 10000 && parameters.IsUseMultipleGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendAndWaitFunc(client, controllerClient, &batchCopy, &sources, &sendDestinations, parameters)		
			} ()
			
		} else {
			sendAndWaitFunc(client, controllerClient, &batch, &sources, &sendDestinations, parameters)	
		}
		CPermDestroy(cperm)
		if hasReachedMeasurementLimit{
			log.Infof("Finished earlier due to maximum %d measurements", totalSent)
			return nil
		}
	}
	wg.Wait()
	return nil 
} 

func BatchSourceDestinationAndSend(client interface{}, controllerClient controller.ControllerClient,
	sources [] string, destinations [] string, sourceDestinationPairIndexes []SourceDestinationPairIndex,
	parameters MeasurementSurveyParameters, sendAndWaitFunc SendAndWaitFunc) error { 
	
	totalSent := 0
	hasReachedMeasurementLimit := false
	var wg sync.WaitGroup
	subBatch := []SourceDestinationPairIndex{}
	subBatchIndex := 0
	for k, sourceDestinationPairIndex :=range(sourceDestinationPairIndexes) {
		
		subBatch = append(subBatch, SourceDestinationPairIndex{
			Source: sourceDestinationPairIndex.Source, 
			Destination: sourceDestinationPairIndex.Destination})
		if len(subBatch) == parameters.BatchLen {
			// So you can restart from where it stopped.
			// if batchIndex >= 90 {
				log.Infof("Sending subdestinations %d batch %d \n", k, subBatchIndex) 
				if parameters.BatchLen >= 10000 && parameters.IsUseMultipleGoroutines {
					subBatchCopy := make([]SourceDestinationPairIndex, len(subBatch))
					copy(subBatchCopy, subBatch)
					wg.Add(1)
					go func ()  {
						defer wg.Done()	
						sendAndWaitFunc(client, controllerClient, &subBatchCopy, &sources, &destinations, parameters)
					} ()
					time.Sleep(180 * time.Second)
				} else {
					sendAndWaitFunc(client, controllerClient, &subBatch, &sources, &destinations, parameters)
				}	
			// }
			// Reset batch
			subBatch = nil
			subBatchIndex ++
		}
		
		totalSent ++
		if totalSent >= parameters.MaximumMeasurements {
			hasReachedMeasurementLimit = true
			sendAndWaitFunc(client, controllerClient, &subBatch, &sources, &destinations, parameters)	
			break
		}
	}
	if hasReachedMeasurementLimit{
		log.Infof("Finished earlier due to maximum %d measurements", totalSent)
		return nil
	}
	// Send last batch
	subBatchCopy := []SourceDestinationPairIndex {}
	copy(subBatchCopy, subBatch)
	if parameters.BatchLen >= 10000 && parameters.IsUseMultipleGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendAndWaitFunc(client, controllerClient, &subBatchCopy, &sources, &destinations, parameters)		
		} ()
		
	} else {
		sendAndWaitFunc(client, controllerClient, &subBatch, &sources, &destinations, parameters)	
	}
	
	wg.Wait()
	return nil 
} 


// SendPings sends pings and just wait for the controller to finish. Use it for short scale surveys (< 1M)
func SendPings(pms []*dm.PingMeasurement, batchSize int) {

	controllerClient, err := CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}
	// Shuffle the pings 
	rand.Shuffle(len(pms), func(i, j int) { pms[i], pms[j] = pms[j], pms[i] })

	// Create batches of 100K TS probes
	batches := [] []*dm.PingMeasurement{}
	for i := 0; i < len(pms); i+=batchSize {
		if i + batchSize > len(pms) {
			batches = append(batches, pms[i:])
		} else {
			batches = append(batches, pms[i:i+batchSize])
		}
		
	}

	for i, batch := range(batches) {
		pingsByVP := map[uint32]uint32 {}
		for _, p := range(batch) {
			var vpI uint32
			if p.Spoof {
				vp := p.SAddr
				vpI, _ = util.IPStringToInt32(vp)
			} else {
				vpI = p.Src
			}
			if _, ok := pingsByVP[vpI]; !ok {
				pingsByVP[vpI] = 0
			}
			pingsByVP[vpI] += 1 
		}
		
		theoreticalSeconds := util.TheoreticalSpoofTimeout(100, pingsByVP)

		if len(batch) > 100 {
			// Small hack here to determine a back of the enveloppe estimation of the measurement time.
			// Basically, check whether we are in an huge batch of pings of just a small RR pings from the revtr
			// system 
			batch[0].IsAbsoluteSpoofTimeout = true
			batch[0].SpoofTimeout = theoreticalSeconds +  10 // time to gather last responses + processing time of the system 	
		}

		log.Infof("Sending %d", i)
		st, err := controllerClient.Ping(context.Background(), &dm.PingArg{
			Pings: batch,
		})
	
		if err != nil {
			panic(err)
		}
	
		// Do nothing on replies, we'll analyze it later. 
		for {
			_, err := st.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// log.Error(err)
				panic(err)
			}
		}

		log.Infof("Finished %d", i)
	} 
	
}

func SendTraceroutes(pms []*dm.TracerouteMeasurement, batchSize int) {

	controllerClient, err := CreateControllerClient("controller/root.crt")
	if err != nil {
		panic(err)
	}
	// Shuffle the measurements 
	rand.Shuffle(len(pms), func(i, j int) { pms[i], pms[j] = pms[j], pms[i] })

	// Create batches of 100K TS probes
	batches := [] []*dm.TracerouteMeasurement{}
	for i := 0; i < len(pms); i+=batchSize {
		if i + batchSize > len(pms) {
			batches = append(batches, pms[i:])
		} else {
			batches = append(batches, pms[i:i+batchSize])
		}
		
	}

	for i, batch := range(batches) {
		log.Infof("Sending %d", i)
		st, err := controllerClient.Traceroute(context.Background(), &dm.TracerouteArg{
			Traceroutes: batch,
		})
	
		if err != nil {
			panic(err)
		}
	
		// Do nothing on replies, we'll analyze it later. 
		for {
			_, err := st.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// log.Error(err)
				panic(err)
			}
		}

		log.Infof("Finished %d", i)
	} 
	
}


func DestinationsFromHitlist(rankingServiceClient *rk.RankingClient, preDestinations *[] string,
	isInitial bool, granularity string) ([] string, error) {
	isOnlyAddresses := false
	ipAddresses := [] *rk.IPAddress{}
	if preDestinations != nil{
		isOnlyAddresses = true
		for _, preDestination := range *preDestinations{
			ipAddress := rk.IPAddress{IpAddress:preDestination}
			ipAddresses = append(ipAddresses, &ipAddress)
		} 
	}
	getTargetsRequest := rk.GetTargetsReq{
			IsInitial : isInitial, 
			IsOnlyAddresses : isOnlyAddresses,
			IpAddresses : ipAddresses,
			Granularity: granularity,
			} 
	resp, err := (*rankingServiceClient).GetTargetsFromHitlist(context.Background(), &getTargetsRequest)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	targets := resp.GetTargets()
	return targets, err
}

func GetOneVPPerSiteRevtr() []string { 

	// Return sources that can run reverse traceroutes

	// Fetch different sources 
	client, err := CreateVPServiceClient("vpservice/root.crt")
	if err != nil {
		panic(err)
	}

	vps, err := client.GetVPs(context.Background(), &vpspb.VPRequest{})
	if err != nil {
		panic(err)
	}
	oneVpPerSite := map[string]string {}
	for _, vp := range(vps.Vps) {
		if _, ok := oneVpPerSite[vp.Site]; !ok {
			if vp.RecordRoute && vp.Spoof {
				ipS, _ :=util.Int32ToIPString(vp.Ip)
				oneVpPerSite[vp.Site] = ipS
			}
		}
	}
	
	sources := [] string{}
	for _, vp := range(oneVpPerSite) {
		sources = append(sources, vp)
	}
	return sources
}