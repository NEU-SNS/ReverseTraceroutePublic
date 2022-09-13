package atlas

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

func DumpRIPETraceroutes(uuid string, dumpFile string, ripeKey string, ripeAccount string) (string, error) {
	home, err := os.UserHomeDir()
	cmd := fmt.Sprintf("cd %s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/ && python3 -m "+
		"atlas.atlas_service --action=fetch --uuid=%s --dump-file=%s --ripe-key=%s --ripe-account=%s",
		home, uuid, dumpFile, ripeKey, ripeAccount,
	)
	out, err := util.RunCMD(cmd)

	// log.Info(stdout)
	if err != nil {
		log.Error(err)
		return out, err
	}
	return out, nil 
}

type SrcProbeID struct {
	Src string
	ProbeID int
}

func SrcProbeIDToFile(path string, srcsProbeIDs[] SrcProbeID) {
	// For more granular writes, open a file for writing.
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, srcProbeID := range(srcsProbeIDs) {
		_, err := w.WriteString(fmt.Sprintf("%s,%d\n", srcProbeID.Src, srcProbeID.ProbeID))
		if err != nil {
			panic(err)
		}
	}
	// Use `Flush` to ensure all buffered operations have
	// been applied to the underlying writer.
	w.Flush()
	f.Close()
}

func ProbeIDsToFile(path string, probeIDs []int64) {
	// For more granular writes, open a file for writing.
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, probeID := range(probeIDs) {
		_, err := w.WriteString(fmt.Sprintf("%d\n", probeID))
		if err != nil {
			panic(err)
		}
	}
	// Use `Flush` to ensure all buffered operations have
	// been applied to the underlying writer.
	w.Flush()
}

func RunRIPETraceroutesPython (traceroutes [] SrcProbeID, ripeKey string, ripeAccount string, description string) string {
	uuid := util.GenUUID()
	home, err := os.UserHomeDir()
	log.Infof("UUID for RIPE traceroutes atlas %s", uuid)
	description += " " + uuid
	sourcesProbesFileParameter := home + "/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/atlas/resources/refresh/sources_probes_refresh_" + uuid + ".csv"
	SrcProbeIDToFile(sourcesProbesFileParameter, traceroutes)
	// Refresh the traceroutes
	cmd := fmt.Sprintf("cd %s/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/ && python3 -m " + 
	`atlas.atlas_service --action=refresh --sources-probes-file="%s" --uuid="%s" --ripe-key="%s" --ripe-account="%s" --description="%s"`,
	home, sourcesProbesFileParameter, uuid, ripeKey, ripeAccount, description,
 	)
	
	stdout, err := util.RunCMD(cmd)
	
	if err != nil {
		log.Info(stdout)
		log.Error(err)
	}
	return uuid 
}

func RunTraceroutes(controllerClient cclient.Client, 
	traceroutesMeasurements [] *dm.TracerouteMeasurement, platform string) ([]*dm.Traceroute, error) {
   log.Infof("Running %d traceroutes", len(traceroutesMeasurements))

   stream, err := controllerClient.Traceroute(context.Background(), &dm.TracerouteArg{
	   Traceroutes: traceroutesMeasurements,
   })
   if err != nil {
	   log.Error(err)
	   return nil, err
   }
   traceroutes := []*dm.Traceroute {}
   for {
	   tr, err := stream.Recv()
	   if err == io.EOF {
		   break
	   }
	   if err != nil {
		   log.Error(err)
	   }

	   if tr.Error !=  "" {
		   log.Error(tr.Error)
	   }

	   hopsWithSrc := [] * dm.TracerouteHop{{
		   ProbeTtl: 0,
		   Addr: tr.Src,
	   }}

	   hops := tr.GetHops()
	   if len(hops) == 0 {
		   continue
	   }
	   if hops[len(hops)-1].Addr != tr.Dst {
		   srcS := util.Int32ToIP(tr.Src)
		   dstS := util.Int32ToIP(tr.Dst)
		   log.Debugf("Traceroute from %s to %s did not reach destination", srcS, dstS)
		   continue
	   }
	   // Add the source of the traceroute in the traceroute hops	
	   hopsWithSrc = append(hopsWithSrc, hops...)
	   tr.Hops = hopsWithSrc
	   // log.Infof("Traceroute from %d to %d done", tr.Src, tr.Dst)
	   tr.Platform = platform
	   traceroutes = append(traceroutes, tr)
   }
   return traceroutes, nil 
}


