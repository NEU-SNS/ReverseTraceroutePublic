package runner_test

import (
	"database/sql"
	"fmt"
	"testing"

	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	revtr "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	"github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/test"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/stretchr/testify/require"
)

func DeleteTestReverseTraceroute(db *sql.DB, label string, batchID uint32) {
	deleteQuery := fmt.Sprintf(`DELETE brevt, revth, revtrs, revts FROM reverse_traceroutes revt 
	LEFT JOIN reverse_traceroute_hops revth ON revt.id = revth.reverse_traceroute_id 
	LEFT JOIN reverse_traceroute_ranked_spoofers revtrs ON  revt.id = revtrs.revtr_id
	LEFT JOIN reverse_traceroute_stats revts ON revt.id= revts.revtr_id
	INNER JOIN batch_revtr brevt ON revt.id = brevt.revtr_id 
	WHERE revt.label='%s';`, label)
	// print(deleteQuery)
	_, err := db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
	deleteQuery = fmt.Sprintf(`DELETE  FROM reverse_traceroutes WHERE label='%s';`, label)
	_, err = db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
}

func TestCheckTraceroute(t *testing.T) {
	// Run ccontroller, revtr and atlas before running this test
	
	// First insert a stale traceroute from a non existing vp so we can intersect at first hop 
	atlasDB := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer atlasDB.Close()

	src := uint32(71815910)
	source, err := util.Int32ToIPString(src)
	dst := uint32(68067148)
	destination, err := util.Int32ToIPString(dst)
	id := test.InsertTestTraceroute(src, dst, "dummy", "", atlasDB)
	defer test.DeleteTestTraceroute(atlasDB, id)

	
	err = test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("")
	if err != nil {
		panic(err)
	}

	// source := "190.98.158.51"
	// destination := "132.227.123.9"
	// source := "195.89.146.12"
	// destination := "160.99.1.254"
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {destination}
	destinations := [] string {source}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	// _, mappingIPProbeID := util.GetRIPEAnchorsIP()
	parameters := survey.MeasurementSurveyParameters{
		// MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness: 43800, // Staleness of the atlas traceroutes, in minutes, correspond to a month 
			Platforms: []string{"dummy"}, // Do not allow tr to source as it is not releaant for caching
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,

	}
	parameters.Label = "debug_test_check_traceroute"
	revtrDB := test.GetMySQLDB("../../cmd/revtr/revtr.config", "revtr")
	defer revtrDB.Close()
	
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		ids := survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
		id := ids[0]
		// Check the reverse traceroute hops
		expectedHops := [] uint32 {src, src, 3093903169, 3093905493, 3093902558, 71683070, dst}
		query := fmt.Sprintf("SELECT hop FROM reverse_traceroute_hops revtrh INNER JOIN batch_revtr brevtr ON revtrh.reverse_traceroute_id=brevtr.revtr_id WHERE brevtr.batch_id=%d ORDER BY `order` ",
		id)

		rows, err := revtrDB.Query(query)
		if err !=nil {
			panic(err)
		}
		c := 0
		for rows.Next() {
			var hop uint32
			err = rows.Scan(&hop)
			if err != nil {
				panic(err)
			}
			require.EqualValues(t, hop, expectedHops[c])
			c++
		}
		defer DeleteTestReverseTraceroute(revtrDB, parameters.Label, id)
	}	

}

func TestCheckTracerouteNoAtlasRRHop(t *testing.T) {
	// Run ccontroller, revtr and atlas before running this test
	
	// First insert a stale traceroute from a non existing vp so we can intersect at first hop 
	atlasDB := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer atlasDB.Close()

	src := uint32(71815910)
	source, err := util.Int32ToIPString(src)
	dst := uint32(68067148)
	destination, err := util.Int32ToIPString(dst)
	id := test.InsertTestTraceroute(src, dst, "dummy", "", atlasDB)
	defer test.DeleteTestTraceroute(atlasDB, id)

	
	err = test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("")
	if err != nil {
		panic(err)
	}

	// source := "190.98.158.51"
	// destination := "132.227.123.9"
	// source := "195.89.146.12"
	// destination := "160.99.1.254"
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {destination}
	destinations := [] string {source}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	// _, mappingIPProbeID := util.GetRIPEAnchorsIP()
	parameters := survey.MeasurementSurveyParameters{
		// MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: false,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness: 43800, // Staleness of the atlas traceroutes, in minutes, correspond to a month 
			Platforms: []string{"dummy"}, // Do not allow tr to source as it is not releaant for caching
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,

	}
	parameters.Label = "debug_test_check_traceroute"
	revtrDB := test.GetMySQLDB("../../cmd/revtr/revtr.config", "revtr")
	defer revtrDB.Close()
	
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 1; i++ {
		ids := survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
		id := ids[0]
		// Check the reverse traceroute hops
		expectedHops := [] uint32 {src, src, 3093903169, 3093905493, 3093902558, 71683070, dst}
		query := fmt.Sprintf("SELECT hop FROM reverse_traceroute_hops revtrh INNER JOIN batch_revtr brevtr ON revtrh.reverse_traceroute_id=brevtr.revtr_id WHERE brevtr.batch_id=%d ORDER BY `order` ",
		id)

		rows, err := revtrDB.Query(query)
		if err !=nil {
			panic(err)
		}
		c := 0
		for rows.Next() {
			var hop uint32
			err = rows.Scan(&hop)
			if err != nil {
				panic(err)
			}
			require.EqualValues(t, hop, expectedHops[c])
			c++
		}
		defer DeleteTestReverseTraceroute(revtrDB, parameters.Label, id)
	}	

}

func TestCheckRevtrCache(t *testing.T) {
	err := test.ClearCache(11211)
	err = test.ClearCache(11212)
	err = test.ClearCache(11213)
	if err != nil {
		panic(err)
	}

	reverseTracerouteClient, err := survey.CreateReverseTracerouteClient("")
	if err != nil {
		panic(err)
	}

	controllerClient, err := survey.CreateControllerClient("")
	if err != nil {
		panic(err)
	}

	ip := uint32(68067148)
	source, err := util.Int32ToIPString(71815910)
	destination, err := util.Int32ToIPString(ip)
	log.Infof("Debugging reverse traceroute from %s to %s", source, destination)
	sources := [] string {source}
	destinations := [] string {destination}

	batch := []survey.SourceDestinationPairIndex {}
	batch = append(batch, survey.SourceDestinationPairIndex {
		Source: 0,
		Destination: 0,
	})	
	// Get the mapping from the RIPE Atlas anchors.
	// _, mappingIPProbeID := util.GetRIPEAnchorsIP()
	parameters := survey.MeasurementSurveyParameters{
		// MappingIPProbeID: mappingIPProbeID,
		AuthKey: "",
		RIPEAPIKey: "",
		BatchLen: 1, 
		NParallelRevtr: 1,
		RankingTechnique: env.IngressCover,
		IsCheckReversePath: false,
		Cache: 15,
		AtlasOptions: revtr.AtlasOptions{
			UseAtlas: true,
			UseRRPings: true,
			IgnoreSourceAS: false,
			IgnoreSource: false,
			Staleness: 43800, // Staleness of the atlas traceroutes, in minutes, correspond to a month 
			Platforms: []string{"dummy"}, 
		},
		CheckDestinationBasedRouting: revtr.CheckDestinationBasedRoutingOptions{
			CheckTunnel: false,
		},
		MaxSpoofers: 10,
		IsDryRun: false,

	}
	parameters.Label = parameters.ToString()
	parameters.Label = "debug_test_check_revtr_from_cache"
	revtrDB := test.GetMySQLDB("../../cmd/revtr/revtr.config", "revtr")
	defer revtrDB.Close()
	// First step of debugging, try to run a traceroute from the destination to the source
	for i := 0; i < 2; i++ {
		// The first one should not come from cache, and the second one should come from cache.
		ids := survey.SendAndWaitReverseTraceroutes(reverseTracerouteClient, controllerClient, &batch, 
			&sources, &destinations,
			parameters)
		id := ids[0]
		defer DeleteTestReverseTraceroute(revtrDB, parameters.Label, id)
		// check that the first one is not from cache
		query  := fmt.Sprintf(
		`SELECT rth.reverse_traceroute_id, hop, hop_type, measurement_id, from_cache FROM reverse_traceroute_hops rth
		INNER JOIN batch_revtr brevtr on brevtr.revtr_id = rth.reverse_traceroute_id 
		WHERE brevtr.batch_id = %d ` + "order by rth.reverse_traceroute_id ASC, rth.`order` ASC", id)
		print(query)
		rows, err := revtrDB.Query(query)
		if err != nil {
			panic(err)
		}
		// require.True(t, rows.Next())
		rowIndex := 0
		for rows.Next() {
			var revtrID int64
			var hop uint32 
			var hopType uint32
			var measurementId sql.NullInt64 
			var fromCache bool 
			err := rows.Scan(&revtrID, &hop, &hopType, & measurementId, &fromCache)
			if err != nil {
				panic(err)
			}
			if i == 0 {
				require.EqualValues(t, fromCache, false)
			} else if i == 1 {
				if rowIndex == 0 {
					require.EqualValues(t, fromCache, false)
				} else {
					require.EqualValues(t, fromCache, true)
				}
				
			}
			rowIndex += 1	
		}
		
	}
}