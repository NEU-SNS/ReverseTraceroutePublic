package ripe

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// GetProbesASN Fetch the mapping between the ASN and the probe (necessary to store in the DB)
func GetProbesASN(useSerialized bool, serializedFile string) (map[uint32]int) {
	mappingIPASN := map[uint32]int {}
	if useSerialized {
		// Parse the csv
		file, err := os.Open(serializedFile)
 
		if err != nil {
			log.Fatalf("failed opening file: %s", err)
		}
 
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			tokens := strings.Split(line, " ")
			ip, _ := strconv.Atoi(tokens[0])
			asn, _ := strconv.Atoi(tokens[1])
			mappingIPASN[uint32(ip)] = asn
		}
		return mappingIPASN
	}
	probesURL := "https://atlas.ripe.net/api/v2/probes"
	next := &probesURL
	
	i := 0
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
		// if i > 1000 {
		// 	break
		// }
		for _, probeResp := range probesResp.ProbesResults{
			i += 1
			if i % 1000 == 0 {
				log.Infof("Mapping IP to ASN for RIPE probes done for %d", i)
			}
			if probeResp.IPv4 == "" {
				continue
			}
			ip, err :=  util.IPStringToInt32(probeResp.IPv4)
			if err != nil {
				log.Errorf(fmt.Sprintf("Cannot convert RIPE probe IP into int, %s, %d", probeResp.IPv4, probeResp.ID))
				continue
			}
			mappingIPASN[ip] = probeResp.ASv4
		}
		next = probesResp.Next
	}

	// Serialize the results
	f, err := os.Create(serializedFile)
	if err != nil {
        panic(err)
	}
	defer f.Close()
	for ip, asn := range(mappingIPASN) {
		line := []byte(fmt.Sprintf("%d %d\n", ip, asn))
		_, err := f.Write(line)
		if err != nil {
			panic(err)
		}
	}
	f.Sync()

	return mappingIPASN
}

func FetchRIPEMeasurementResults(t time.Time, msrID int, raAPIKey string) (*http.Response, error) {
	// url := fmt.Sprintf("https://atlas.ripe.net/api/v2/measurements/%d?key=%s", 
	// 				msrID, raAPIKey)
	// resp, err := http.Get(url)
	// if err != nil{
	// 	return nil, err
	// }
	// if resp.StatusCode != http.StatusOK {
	// 	message := fmt.Sprintf("Bad http status code while trying to fetch %s", url)
	// 	log.Error(message)
	// 	return nil, errors.New(message)
	// }
	// isFinished, status, err:= pollRIPEMeasurement(msrID, raAPIKey)
	// log.Info(fmt.Sprintf("Tick at %s, polling %s, status %s", t, url, status))
	// if err != nil{
	// 	return nil, err
	// }
	
	// Now query the measurement result
	url := fmt.Sprintf("https://atlas.ripe.net/api/v2/measurements/%d/results?key=%s",
	msrID, raAPIKey)
	log.Info(fmt.Sprintf("Trying to fetch results at %s", url))
	resp, err := http.Get(url)
	if resp.StatusCode != http.StatusOK {
		message := fmt.Sprintf("Bad http status code while trying to fetch results %s", url)
		return nil, errors.New(message) 
	}
	return resp, err 
}


func FetchRIPEMeasurementsToTraceroutes(ripeTraceroutesJsonFile string, 
	serializedRIPEProbesFile string,
	platform string) ([]* dm.Traceroute, error) {
	ripeSourceASN := GetProbesASN(true, serializedRIPEProbesFile)
	ripeMeasurements := []dm.RIPEMeasurementTracerouteResult{}
	
	// Open our jsonFile
	jsonFile, err := os.Open(ripeTraceroutesJsonFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(byteValue, &ripeMeasurements)

	traceroutes := [] * dm.Traceroute{}
	// Flush what we have and return.
	log.Info(fmt.Sprintf("Loaded results of %s, %d traceroutes",
		ripeTraceroutesJsonFile, len(ripeMeasurements)))
	for i, ripeMeasurement := range(ripeMeasurements){
		src, _ := util.IPStringToInt32(ripeMeasurement.Src)
		dst, _ := util.IPStringToInt32(ripeMeasurement.Dst)
		if src == 0  || dst == 0 {
			continue
		}
		traceroute := dm.ConvertRIPETraceroute(ripeMeasurement)
		traceroute.Platform = platform
		// Test if traceroute reached the destination
		if traceroute.Hops[len(traceroute.Hops)-1].Addr != traceroute.Dst {
			continue
		}

		if asn, ok := ripeSourceASN[traceroute.Src]; ok {
			traceroute.SourceAsn = int32(asn)
		} 
		// c.opts.trs.StoreAtlasTraceroute(&traceroute)
		traceroutes = append(traceroutes, &traceroute)
		if i % 1000 == 0 {
			log.Infof("Converted %d ripe traceroutes ", i)
		}
		
	}
	return traceroutes, nil 
}


