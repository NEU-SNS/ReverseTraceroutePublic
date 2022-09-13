package test

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local"

// Config is the config struct for the controller
type Config struct {
	Db    da.DbConfig
}
// NewConfig returns a new blank Config
func NewConfig() Config {
	c := Config{
		Db:    da.DbConfig{},
	}

	return c
}

func ClearCache(port int) error {
	// Clean the cache after the debug test
	cmd := fmt.Sprintf("echo 'flush_all' | nc -q0 localhost %d", port)
	_, err := util.RunCMD(cmd)	
	return err
}

func GetMySQLDB(configPath string, dbName string)  (db *sql.DB) {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	fmt.Println(path)
	var conf = NewConfig()
	// config.AddConfigPath("../../cmd/atlas/atlas.config")
	config.AddConfigPath(configPath)
	err = config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Errorf("Failed to parse config: %v", err)
		panic(err)
	}
	db, err = sql.Open("mysql", fmt.Sprintf(conFmt, conf.Db.WriteConfigs[0].User, conf.Db.WriteConfigs[0].Password,
		 conf.Db.WriteConfigs[0].Host, conf.Db.WriteConfigs[0].Port, dbName))
	if err != nil {
		panic(err)
	}
	return db
}

func InsertTestTraceroute(src uint32, dst uint32, platform string, date string, db *sql.DB) int64 {

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	// Create a fake traceroute and fake RR pings
	insertTrQuery := fmt.Sprintf("INSERT INTO atlas_traceroutes (dest, src, source_asn, platform, date, stale) VALUES (%d,%d,%d,\"%s\",%s,%d)",
	dst, src, 6939, platform, date, 0)
	
	res, err := tx.Exec(insertTrQuery)
	fmt.Printf("%s", insertTrQuery)
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		panic(err)
	}

	insertRRPingsQuery := "INSERT INTO atlas_rr_pings(traceroute_id, ping_id, tr_hop, rr_hop, rr_hop_index, date) VALUES "
	rowsToInsert := fmt.Sprintf(`
	(%[1]d, 2915894233, %[2]d, %[2]d,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d, %[2]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,   71677608,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,   71826305,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,   %[3]d,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894233, %[2]d,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d, %[2]d,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d, %[2]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,   71677608,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,   71826305,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,   %[3]d,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426982, %[2]d,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d, %[2]d,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d, %[2]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,   71677608,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,   71826305,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,   %[3]d,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613299, %[2]d,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d, %[2]d,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d, %[2]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,   71677608,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,   71826305,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,   %[3]d,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221606, %[2]d,          0,   8 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d, %[2]d,   0 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d, %[2]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d,   71677608,   2 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d,   71826305,   3 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d,   %[3]d,   4 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351961, %[2]d,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558, 3093902558,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,   71826305,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,   %[3]d,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3628221580, 3093902558,          0,   8 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558, 3093902558,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,   71826305,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,   %[3]d,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,          0,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,          0,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915894259, 3093902558,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558, 3093902558,   0 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,   71826305,   1 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,   %[3]d,   2 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,          0,   3 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,          0,   4 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d,   69351987, 3093902558,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558, 3093902558,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,   71826305,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,   %[3]d,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,          0,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,          0,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3277426995, 3093902558,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558, 3093902558,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,   71826305,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,   %[3]d,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,          0,   3 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,          0,   4 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,          0,   5 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,          0,   6 , '2021-03-31 09:41:44' ),
	(%[1]d, 3561613286, 3093902558,          0,   7 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915893478,   71683070,   71683070,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915893478,   71683070,   %[3]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 2915893478,   71683070,          0,   2 , '2021-03-31 09:41:44' ),
	(%[1]d, 3509823923,   71683070,   71683070,   0 , '2021-03-31 09:41:44' ),
	(%[1]d, 3509823923,   71683070,   %[3]d,   1 , '2021-03-31 09:41:44' ),
	(%[1]d, 3509823923,   71683070,          0,   2 , '2021-03-31 09:41:44')
	`, id, src, dst)
	insertRRPingsQuery += rowsToInsert
	fmt.Printf("%s\n", insertRRPingsQuery)
	_, err = tx.Exec(insertRRPingsQuery)
	
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	insertTRHopRRHopIntersectionQuery := fmt.Sprintf(`
	INSERT INTO atlas_rr_intersection (traceroute_id, rr_hop, tr_ttl_start, tr_ttl_end) VALUES 
	(%[1]d, 71677608, 0, 4),
	(%[1]d, 71826305, 4, 4)
	`, id)

	_, err = tx.Exec(insertTRHopRRHopIntersectionQuery)
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	insertTrHopQuery := fmt.Sprintf(`
		INSERT INTO atlas_traceroute_hops (trace_id, hop, ttl, mpls) VALUES 
		(%[1]d, %[2]d, 0, 0),
		(%[1]d, 3093903169, 1, 0),
		(%[1]d, 3093905493, 2, 0),
		(%[1]d, 3093902558, 3, 0),
		(%[1]d,  71683070, 4, 0),
		(%[1]d,  %[3]d, 5, 0)
		`, id, src, dst)

	_, err = tx.Exec(insertTrHopQuery)
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	return id
}

func InsertTestAlias(db *sql.DB, ip uint32, alias uint32) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	// Create a fake traceroute and fake RR pings
	insertAliasQuery := fmt.Sprintf(`INSERT INTO ip_aliases (cluster_id, ip_address) VALUES 
	(%[1]d, %[2]d), 
	(%[1]d, %[3]d)`,
 	0, ip, alias)
	 fmt.Printf("%s", insertAliasQuery)
	_, err = tx.Exec(insertAliasQuery)
	
	if err != nil {
		tx.Rollback()
		panic(err)
	}
}

func DeleteTestAlias(db *sql.DB, clusterID int64) {
	deleteQuery := fmt.Sprintf("DELETE FROM ip_aliases where cluster_id = %d", clusterID)
	print("%s\n", deleteQuery)
	_, err := db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
}

func DeleteTestTraceroute(db *sql.DB, id int64) {
	deleteQuery := fmt.Sprintf(`DELETE at, ath, arri, arrp FROM 
	atlas_traceroutes at 
	LEFT JOIN  atlas_traceroute_hops ath ON ath.trace_id=at.id
	LEFT JOIN atlas_rr_pings arrp ON at.id=arrp.traceroute_id 
	LEFT JOIN atlas_rr_intersection arri ON at.id=arri.traceroute_id
	WHERE at.id=%d`, id)
	fmt.Printf("%s\n", deleteQuery)
	_, err := db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
}

func DeleteTestTracerouteByLabel(db *sql.DB, platform string) {
	deleteQuery := fmt.Sprintf(`DELETE at, ath, arri, arrp FROM 
	atlas_traceroutes at 
	LEFT JOIN  atlas_traceroute_hops ath ON ath.trace_id=at.id
	LEFT JOIN atlas_rr_pings arrp ON at.id=arrp.traceroute_id 
	LEFT JOIN atlas_rr_intersection arri ON at.id=arri.traceroute_id
	WHERE at.platform="%s"`, platform)
	print("%s\n", deleteQuery)
	_, err := db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
}


// TODO Does not work because of FK constraint.
func DeleteReverseTracerouteByLabel(db *sql.DB, label string) {
	deleteQuery := fmt.Sprintf(`
	DELETE brevt, revth, revtrs, revts FROM reverse_traceroutes revt 
	LEFT JOIN reverse_traceroute_hops revth ON revt.id = revth.reverse_traceroute_id 
	LEFT JOIN reverse_traceroute_ranked_spoofers revtrs ON  revt.id = revtrs.revtr_id
	LEFT JOIN reverse_traceroute_stats revts ON revt.id= revts.revtr_id
	INNER JOIN batch_revtr brevt ON revt.id = brevt.revtr_id 
	WHERE revt.label='%s';
	`, label)
	_, err := db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}

	deleteQuery = fmt.Sprintf(`
	DELETE revt FROM reverse_traceroutes revt WHERE revt.label='%s';
	`, label)
	_, err = db.Exec(deleteQuery)
	if err != nil {
		panic(err)
	}
}