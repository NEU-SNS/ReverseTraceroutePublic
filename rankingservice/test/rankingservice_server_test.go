package rankingservice_server_test

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	pb "github.com/NEU-SNS/ReverseTraceroute/rankingservice/protos"
	"github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/stretchr/testify/require"
)


func TestRankingServiceServerLoad(t *testing.T) {
	// Make sure the ranking service is running properly

	rkcl, err :=  survey.CreateRankingClient()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	var wg sync.WaitGroup
	
	for i := 0; i < 50000; i++ {
		// time.Sleep(5 * time.Millisecond)
		wg.Add(1)
		go func (k int) {
			defer wg.Done()
			ip, _ := util.Int32ToIPString(rand.Uint32())
			resp, err := rkcl.GetVPs(ctx,  &pb.GetVPsReq{
				Ip:                   ip,
				NumVPs:               250,
				RankingTechnique:     "old_revtr_cover",
				ExecutedMeasurements: nil, // Initial batch, so nothing executed yet.
			})
			if err != nil {

				panic(err)
			}
			if k % 100 == 0 {
				log.Infof("Query answered %d\n", k)
				require.True(t, len(resp.Vps) > 0)
			}
			
		} (i)
	}
	wg.Wait()
	

}