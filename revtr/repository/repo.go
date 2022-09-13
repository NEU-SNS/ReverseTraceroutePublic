package repo

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	dasql "github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/protobuf/ptypes"
)

var reverseTraceroutesTable = "reverse_traceroutes"
var reverseTracerouteHopsTable = "reverse_traceroute_hops"
var reverseTracerouteRankedSpoofersTable = "reverse_traceroute_ranked_spoofers"
var reverseTracerouteStatsTable = "reverse_traceroute_stats" 

var tables = []*string{&reverseTraceroutesTable, &reverseTracerouteHopsTable, &reverseTracerouteRankedSpoofersTable,
	 &reverseTracerouteStatsTable, 
	}

var revtrStoreRevtr, revtrInitRevtr, revtrInitRevtrBulk, revtrUpdateRevtrStatus, revtrUpdateRevtrTracerouteID,
 revtrStoreRevtrHop, revtrStoreRevtrHopBulk, revtrStoreStats, revtrStoreRankedSpoofersBulk, revtrGetUserByKey,
 revtrCanAddTraces, revtrAddBatch, revtrAddBatchRevtr, revtrAddBatchRevtrBulk, revtrGetRevtrsInBatch, revtrGetRevtrsByLabel,
 revtrGetRevtrsBatchStatus, revtrGetHopsForRevtr, revtrGetStatsForRevtr, revtrUpdateRevtr, revtrExistsRevtr string

func initRevtrQueries() {
	revtrStoreRevtr = fmt.Sprintf(`INSERT INTO %s(src, dst, runtime, stop_reason, status, fail_reason) VALUES
	(?, ?, ?, ?, ?, ?)`, reverseTraceroutesTable)
	revtrInitRevtr         = fmt.Sprintf(`INSERT INTO %s(src, dst) VALUES (?, ?)`, reverseTraceroutesTable)
	revtrInitRevtrBulk     = fmt.Sprintf(`INSERT INTO %s(src, dst, label) VALUES `, reverseTraceroutesTable)
	revtrUpdateRevtrStatus = fmt.Sprintf(`UPDATE %s SET status = ? WHERE id = ?`, reverseTraceroutesTable)
	revtrUpdateRevtrTracerouteID = fmt.Sprintf(`UPDATE %s SET traceroute_id = ? WHERE id = ?`, reverseTraceroutesTable)
	
	revtrStoreRevtrHop     = fmt.Sprintf("INSERT INTO %s(reverse_traceroute_id, hop, hop_type, dest_based_routing_type, `order`, measurement_id, from_cache) VALUES (?, ?, ?, ?, ?, ?, ?)", reverseTracerouteHopsTable)
	revtrStoreRevtrHopBulk = fmt.Sprintf("INSERT INTO %s(reverse_traceroute_id, hop, hop_type, dest_based_routing_type, `order`, measurement_id, from_cache, rtt, rtt_measurement_id) VALUES ", reverseTracerouteHopsTable)
	revtrStoreStats        = fmt.Sprintf(`INSERT INTO 
                                  %s(revtr_id, rr_probes, spoofed_rr_probes, 
                                                           ts_probes, spoofed_ts_probes, rr_round_count, 
                                                           rr_duration, ts_round_count, ts_duration, tr_to_src_round_count, 
                                                           tr_to_src_duration, assume_symmetric_round_count, assume_symmetric_duration, 
                                                           background_trs_round_count, background_trs_duration) 
														   values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, reverseTracerouteStatsTable)
	revtrStoreRankedSpoofersBulk = 	fmt.Sprintf("INSERT INTO %s(revtr_id, hop, `rank`, ip, ping_id, ranking_technique) VALUES ", reverseTracerouteRankedSpoofersTable)												   
	revtrGetUserByKey = "SELECT " +
		"`id`, `name`, `email`, `max`, `delay`, `key`, max_revtr_per_day, revtr_run_today, max_parallel_revtr " +
		"FROM " +
		"users " +
		"WHERE " +
		"`key` = ?"
	revtrCanAddTraces = fmt.Sprintf("SELECT " +
		"	CASE WHEN COUNT(*) + ? < u.max THEN TRUE ELSE FALSE END AS Valid " +
		"	FROM " +
		" 	users u INNER JOIN batch b ON u.id = b.user_id " +
		"	INNER JOIN batch_revtr brtr ON brtr.batch_id = b.id " +
		"	INNER JOIN %s rt ON rt.id = brtr.revtr_id " +
		"	WHERE " +
		"	u.`key` = ? AND b.created >= DATE_SUB(NOW(), INTERVAL u.delay MINUTE) " +
		"	GROUP BY " +
		"		u.max ", reverseTraceroutesTable)
	revtrAddBatch         = "INSERT INTO batch(user_id) SELECT id FROM users WHERE users.`key` = ?"
	revtrAddBatchRevtr    = "INSERT INTO batch_revtr(batch_id, revtr_id) VALUES (?, ?)"
	revtrAddBatchRevtrBulk = "INSERT INTO batch_revtr(batch_id, revtr_id) VALUES "
	revtrGetRevtrsInBatch = fmt.Sprintf("SELECT rt.id, rt.src, rt.dst, rt.runtime, rt.stop_reason, rt.status, rt.date, rt.fail_reason " +
		"FROM users u INNER JOIN batch b ON u.id = b.user_id INNER JOIN batch_revtr brt ON b.id = brt.batch_id " +
		"INNER JOIN %s rt ON brt.revtr_id = rt.id WHERE u.id = ? AND b.id = ?", reverseTraceroutesTable)
	revtrGetRevtrsByLabel = fmt.Sprintf("SELECT rt.id, rt.src, rt.dst, rt.runtime, rt.stop_reason, rt.status, rt.date, rt.fail_reason " +
	"FROM %s rt WHERE rt.label = ? ", reverseTraceroutesTable)
	revtrGetRevtrsBatchStatus = fmt.Sprintf("SELECT rt.status " +
	"FROM users u INNER JOIN batch b ON u.id = b.user_id INNER JOIN batch_revtr brt ON b.id = brt.batch_id " +
	"INNER JOIN %s rt ON brt.revtr_id = rt.id WHERE u.id = ? AND b.id = ?", reverseTraceroutesTable)
	revtrGetHopsForRevtr  = fmt.Sprintf("SELECT hop, hop_type, measurement_id, from_cache, rtt, rtt_measurement_id FROM %s rth WHERE rth.reverse_traceroute_id = ? ORDER BY rth.`order`", reverseTracerouteHopsTable)
	revtrGetStatsForRevtr = fmt.Sprintf(`SELECT rr_probes, spoofed_rr_probes, ts_probes, spoofed_ts_probes, rr_round_count, rr_duration, 
                             ts_round_count, ts_duration, tr_to_src_round_count, tr_to_src_duration, assume_symmetric_round_count, 
                             assume_symmetric_duration, background_trs_round_count, background_trs_duration 
                             from %s rts where rts.revtr_id = ? limit 1`, reverseTracerouteStatsTable)
	revtrUpdateRevtr = fmt.Sprintf(`UPDATE %s 
	SET 
	date = ?,	
	runtime = ?,
	stop_reason = ?,
	status = ?,
	fail_reason = ?,
	label=?,
	traceroute_id=?
	WHERE
		%s.id = ?;`, reverseTraceroutesTable, reverseTraceroutesTable)
	
	revtrExistsRevtr = fmt.Sprintf(`SELECT src, dst, status FROM %s WHERE label=?`, reverseTraceroutesTable)
}

var (
	// ErrInvalidUserID is returned when the user id provided is not in the system
	ErrInvalidUserID = fmt.Errorf("Invalid User Id")
	// ErrNoRow is returned when a query that should return row doesn't
	ErrNoRow = fmt.Errorf("No rows returned when one should have been")
	// ErrCannotAddRevtrBatch is returned if the user is not allowed to add more revtrs
	ErrCannotAddRevtrBatch = fmt.Errorf("Cannot add more revtrs")
	// ErrFailedToStoreBatch is returned when storing a batch of revtrs failed
	ErrFailedToStoreBatch = fmt.Errorf("Failed to store batch of revtrs")
	// ErrFailedToGetBatch is returned when a batch cannot be fetched
	ErrFailedToGetBatch = fmt.Errorf("Failed to get batch of revtrs")
	// ErrFailedToUpdateRevtr is returned when a revtr cannot be updated 
	ErrFailedToUpdateRevtr = fmt.Errorf("Failed to update revtr")
)

// Repo is a repository for storing and retreiving reverse traceroutes
type Repo struct {
	repo *repository.DB
}

// Configs is the configuration for the repo
type Configs struct {
	WriteConfigs []Config
	ReadConfigs  []Config
	Environment string 
}

// Config is an individual reader/writer config
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string

}

type repoOptions struct {
	writeConfigs []Config
	readConfigs  []Config
	environment string 
}

// Option sets up the Repo
type Option func(*repoOptions)

// WithWriteConfig configures the repo with the given config used as a writer
// multiples may be provided
func WithWriteConfig(c Config) Option {
	return func(ro *repoOptions) {
		ro.writeConfigs = append(ro.writeConfigs, c)
	}
}

// WithReadConfig configures the repo with the given config used as a reader
// multiples may be provided
func WithReadConfig(c Config) Option {
	return func(ro *repoOptions) {
		ro.readConfigs = append(ro.readConfigs, c)
	}
}

func WithEnvironment(environment string) Option {
	return func(ro *repoOptions) {
		ro.environment = environment
	}
}

// NewRepo creates a new Repo configured with the given options
func NewRepo(options ...Option) (*Repo, error) {
	ro := &repoOptions{}
	for _, opt := range options {
		opt(ro)
	}
	var dbc repository.DbConfig
	for _, wc := range ro.writeConfigs {
		var c repository.Config
		c.User = wc.User
		c.Password = wc.Password
		c.Host = wc.Host
		c.Port = wc.Port
		c.Db = wc.Db
		dbc.WriteConfigs = append(dbc.WriteConfigs, c)
	}
	for _, rc := range ro.readConfigs {
		var c repository.Config
		c.User = rc.User
		c.Password = rc.Password
		c.Host = rc.Host
		c.Port = rc.Port
		c.Db = rc.Db
		dbc.ReadConfigs = append(dbc.ReadConfigs, c)
	}
	db, err := repository.NewDB(dbc)
	if err != nil {
		return nil, err
	}

	if ro.environment == "debug" {
		for _, table :=range(tables) {
			*table +=  "_debug"
		}
	}

	initRevtrQueries()

	return &Repo{repo: db}, nil
}

func (r *Repo) StoreRankedSpoofersBulk(tx * sql.Tx, rt pb.ReverseTraceroute) error {
	if len(rt.RankedSpoofersByHop) == 0{
		return nil
	}
	query := bytes.Buffer{}
	query.WriteString(revtrStoreRankedSpoofersBulk)
	for hop, rankedSpoofers := range rt.RankedSpoofersByHop {
		// if i == 2 {
		// 	break
		// }
		for _, rankedSpoofer := range rankedSpoofers.RankedSpoofers {
			if rankedSpoofer.MeasurementId == 0 {
				query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %s, \"%s\"),", 
				rt.Id, hop, rankedSpoofer.Rank, rankedSpoofer.Ip, "NULL", rankedSpoofer.RankingTechnique))
			} else  {
				query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, \"%s\"),", 
				rt.Id, hop, rankedSpoofer.Rank, rankedSpoofer.Ip, rankedSpoofer.MeasurementId, rankedSpoofer.RankingTechnique))
			}
			
		}
	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_ , err := tx.Exec(queryStr)
	if err != nil {
		log.Errorf(queryStr)
		return err
	}
	return nil
}

// StoreBatchedRevtrs stores a batch of Revtrs
func (r *Repo) StoreBatchedRevtrs(batch []pb.ReverseTraceroute) error {
	con := r.repo.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return ErrFailedToStoreBatch
	}
	for _, rt := range batch {
		startTime := time.Unix(rt.StartTime, 0).Format(environment.DateFormat)
		_, err = tx.Exec(revtrUpdateRevtr, startTime, rt.Runtime,
			rt.StopReason, rt.Status.String(),
			rt.FailReason, rt.Label, rt.ForwardTracerouteId, rt.Id)
		if err != nil {
			log.Error(err)
			if err := tx.Rollback(); err != nil {
				log.Error(err)
			}
			return ErrFailedToStoreBatch
		}

		// bulk insert reverse path
		if len(rt.Path) > 0{
			query := bytes.Buffer{}
			query.WriteString(revtrStoreRevtrHopBulk)
			
			for i, hop := range rt.Path {
				hopi, _ := util.IPStringToInt32(hop.Hop)
 	  			var fromCache string
				if hop.FromCache {
					fromCache = "TRUE"
				} else  {
					fromCache = "FALSE"
				}
				query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, %d, %s, %d, %d),", 
				rt.Id, hopi, uint32(hop.Type), uint32(hop.DestBasedRoutingType), i, hop.MeasurementId, fromCache, hop.Rtt, hop.RttMeasurementId))
			}
			
			//trim the last ,
			queryStr := query.String()
			queryStr = queryStr[0:len(queryStr)-1]
			//format all vals at once
			_ , err := tx.Exec(queryStr)
			if err != nil {
				tx.Rollback()
				log.Error(err)
				return ErrFailedToStoreBatch
			}
		}
		

		// for i, hop := range rt.Path {
		// 	hopi, _ := util.IPStringToInt32(hop.Hop)
		// 	_, err = tx.Exec(revtrStoreRevtrHop, rt.Id, hopi, uint32(hop.Type), i)
		// 	if err != nil {
		// 		log.Error(err)
		// 		if err := tx.Rollback(); err != nil {
		// 			log.Error(err)
		// 		}
		// 		return ErrFailedToStoreBatch
		// 	}
		// }

		err = r.StoreRankedSpoofersBulk(tx, rt)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error(err)
			}
			return ErrFailedToStoreBatch
		}

		rrdur, _ := ptypes.Duration(rt.Stats.RrDuration)
		tsdur, _ := ptypes.Duration(rt.Stats.TsDuration)
		trdur, _ := ptypes.Duration(rt.Stats.TrToSrcDuration)
		asdur, _ := ptypes.Duration(rt.Stats.AssumeSymmetricDuration)
		bgtrdur, _ := ptypes.Duration(rt.Stats.BackgroundTrsDuration)
		_, err = tx.Exec(revtrStoreStats, rt.Id, rt.Stats.RrProbes, rt.Stats.SpoofedRrProbes,
			rt.Stats.TsProbes, rt.Stats.SpoofedTsProbes, rt.Stats.RrRoundCount,
			rrdur.Nanoseconds(), rt.Stats.TsRoundCount, tsdur.Nanoseconds(),
			rt.Stats.TrToSrcRoundCount, trdur.Nanoseconds(),
			rt.Stats.AssumeSymmetricRoundCount, asdur.Nanoseconds(),
			rt.Stats.BackgroundTrsRoundCount, bgtrdur.Nanoseconds())
		if err != nil {
			log.Error(err)
			if err := tx.Rollback(); err != nil {
				log.Error(err)
			}
			return ErrFailedToStoreBatch
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		if err := tx.Rollback(); err != nil {
			log.Error(err)
		}
		return ErrFailedToStoreBatch
	}
	return nil
}

type rtid struct {
	rt pb.ReverseTraceroute
	id uint32
}

func (r * Repo) GetRevtrsMetaOnly(uid uint32, bid uint32) ([] *pb.ReverseTracerouteMetaOnly, error){
	con := r.repo.GetReader()
	res, err := con.Query(revtrGetRevtrsInBatch, uid, bid)
	defer func() {
		if err := res.Close(); err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	
	var final []*pb.ReverseTracerouteMetaOnly
	for res.Next() {
		var r pb.ReverseTraceroute
		var src, dst, id uint32
		var t time.Time
		var status string
		err = res.Scan(&id, &src, &dst, &r.Runtime,
			&r.StopReason, &status, &t, &r.FailReason)
		if err != nil {
			log.Error(err)
			return nil, ErrFailedToGetBatch
		}
		r.Src, _ = util.Int32ToIPString(src)
		r.Dst, _ = util.Int32ToIPString(dst)
		final = append(final, &pb.ReverseTracerouteMetaOnly{
			Src: r.Src,
			Dst: r.Dst,
			Id: id,
		})
	}
	if err := res.Err(); err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	return final, nil 
}

func (r * Repo) ExistsRevtr(label string) (map[types.SrcDstPair]struct{}, error){
	con := r.repo.GetReader()
	res, err := con.Query(revtrExistsRevtr, label)
	defer func() {
		if err := res.Close(); err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	srcDstPairs := map[types.SrcDstPair]struct{} {}
	for res.Next() {
		var srcDstPair types.SrcDstPair
		status := ""
		err = res.Scan(&srcDstPair.Src, &srcDstPair.Dst, &status)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if status == "COMPLETED" {
			srcDstPairs[srcDstPair] = struct{}{}
		}
	}
	return srcDstPairs, nil
}

func (r * Repo) GetRevtrsBatchStatus(uid uint32, bid uint32) ([]pb.RevtrStatus, error) {
	con := r.repo.GetReader()
	res, err := con.Query(revtrGetRevtrsBatchStatus, uid, bid)
	defer func() {
		if err := res.Close(); err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}

	batchStatus := []pb.RevtrStatus{}
	for res.Next() {
		revtrStatusStr := ""
		err = res.Scan(&revtrStatusStr)
		if err != nil {
			log.Error(err)
			return nil, ErrFailedToGetBatch
		}
		revtrStatus := pb.RevtrStatus(pb.RevtrStatus_value[revtrStatusStr])
		batchStatus = append(batchStatus, revtrStatus)
	}
	return batchStatus, nil
}

// GetRevtrsInBatch gets the reverse traceroutes in batch bid
func (r *Repo) GetRevtrsInBatch(uid, bid uint32) ([]*pb.ReverseTraceroute, error) {
	con := r.repo.GetReader()
	res, err := con.Query(revtrGetRevtrsInBatch, uid, bid)
	defer func() {
		if err := res.Close(); err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	final, err := addRevtrInfos(res, con)
	if err != nil {
		log.Error(err)
		return nil, err 
	}
	return final, nil
}

// GetRevtrsByLabel gets the reverse traceroutes per label
func (r *Repo) GetRevtrsByLabel(uid uint32, label string) ([]*pb.ReverseTraceroute, error) {
	con := r.repo.GetReader()
	res, err := con.Query(revtrGetRevtrsByLabel, label)
	defer res.Close()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	
	final, err := addRevtrInfos(res, con)
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}

	return final, nil
}

func addRevtrInfos(res *sql.Rows, con *sql.DB) ([]*pb.ReverseTraceroute, error) {
	var ret []rtid
	var final []*pb.ReverseTraceroute
	for res.Next() {
		var r pb.ReverseTraceroute
		var src, dst, id uint32
		var t time.Time
		var status string
		err := res.Scan(&id, &src, &dst, &r.Runtime,
			&r.StopReason, &status, &t, &r.FailReason)
		if err != nil {
			log.Error(err)
			return nil, ErrFailedToGetBatch
		}
		r.Src, _ = util.Int32ToIPString(src)
		r.Dst, _ = util.Int32ToIPString(dst)
		r.Date = t.String()
		r.Status = pb.RevtrStatus(pb.RevtrStatus_value[status])
		if r.Status == pb.RevtrStatus_RUNNING {
			r.Runtime = time.Since(t).Nanoseconds()
		}
		ret = append(ret, rtid{rt: r, id: id})
	}

	if err := res.Err(); err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}

	for _, rt := range ret {
		use := rt.rt
		log.Debug(rt)
		if use.Status == pb.RevtrStatus_COMPLETED {
			res2, err := con.Query(revtrGetHopsForRevtr, rt.id)
			if err != nil {
				log.Error(err)
				return nil, ErrFailedToGetBatch
			}
			for res2.Next() {
				h := pb.RevtrHop{}
				var hop, hopType, rtt uint32
				var measurementID, rttMeasurementID int64
				var fromCache bool
				err = res2.Scan(&hop, &hopType, &measurementID, &fromCache, &rtt, &rttMeasurementID)
				h.Hop, _ = util.Int32ToIPString(hop)
				h.Type = pb.RevtrHopType(hopType)
				h.MeasurementId = measurementID
				h.FromCache = fromCache
				h.Rtt = rtt
				h.RttMeasurementId = rttMeasurementID
				use.Path = append(use.Path, &h)
				if err != nil {
					log.Error(err)
					if err := res2.Close(); err != nil {
						log.Error(err)
					}
					return nil, ErrFailedToGetBatch
				}
			}
			if err := res2.Err(); err != nil {
				log.Error(err)
				return nil, ErrFailedToGetBatch
			}
			if err := res2.Close(); err != nil {
				log.Error(err)
			}
		}
		res3, err := con.Query(revtrGetStatsForRevtr, rt.id)
		if err != nil {
			log.Error(err)
			return nil, ErrFailedToGetBatch
		}
		for res3.Next() {
			stat := pb.Stats{}
			var rrdur, tsdur, trdur, assdur, bgtrdur int64
			err = res3.Scan(&stat.RrProbes, &stat.SpoofedRrProbes, &stat.TsProbes,
				&stat.SpoofedTsProbes, &stat.RrRoundCount, &rrdur, &stat.TsRoundCount,
				&tsdur, &stat.TrToSrcRoundCount, &trdur, &stat.AssumeSymmetricRoundCount,
				&assdur, &stat.BackgroundTrsRoundCount, &bgtrdur)
			if err != nil {
				log.Error(err)
				if err := res3.Close(); err != nil {
					log.Error(err)
					return nil, ErrFailedToGetBatch
				}
			}
			if err := res3.Err(); err != nil {
				log.Error(err)
				return nil, ErrFailedToGetBatch
			}
			if err := res3.Close(); err != nil {
				log.Error(err)
			}
			stat.RrDuration = ptypes.DurationProto(time.Duration(rrdur))
			stat.TsDuration = ptypes.DurationProto(time.Duration(tsdur))
			stat.TrToSrcDuration = ptypes.DurationProto(time.Duration(trdur))
			stat.AssumeSymmetricDuration = ptypes.DurationProto(time.Duration(assdur))
			stat.BackgroundTrsDuration = ptypes.DurationProto(time.Duration(bgtrdur))
			use.Stats = &stat
		}
		final = append(final, &use)
	}
	log.Debug(final)
	return final, nil
}

func (r *Repo) UpdateRevtr(uid uint32, revtrID uint32, tracerouteID int64) (error){
	con := r.repo.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return ErrFailedToUpdateRevtr
	}
	
	_, err = tx.Exec(revtrUpdateRevtrTracerouteID, tracerouteID, revtrID)
	if err != nil {
		log.Error(err)
		if err := tx.Rollback(); err != nil {
			log.Error(err)
		}
		return ErrFailedToUpdateRevtr
	}

	err = tx.Commit()
	if err != nil {
		logError(tx.Rollback)
		log.Error(err)
		return ErrFailedToUpdateRevtr
	}
	return nil 
}

type errorf func() error

func logError(e errorf) {
	if err := e(); err != nil {
		log.Error(err)
	}
}

// CreateRevtrBatch creatse a batch of revtrs if the user identified by id
// is allowed to issue more reverse traceroutes
func (r *Repo) CreateRevtrBatch(batch []*pb.RevtrMeasurement, id string) ([]*pb.RevtrMeasurement, uint32, error) {
	con := r.repo.GetWriter()
	// fmt.Printf("Number of open connections: %s")
	// var canDo bool
	// row := tx.QueryRow(revtrCanAddTraces, len(batch), id)
	// err = row.Scan(&canDo)
	// switch {
	// // This requires the assumption that I'm already authorized
	// case err == sql.ErrNoRows:
	// 	canDo = true
	// case err != nil:
	// 	log.Error(err)
	// 	logError(tx.Rollback)
	// 	return nil, 0, ErrCannotAddRevtrBatch
	// }
	// if !canDo {
	// 	logError(tx.Rollback)
	// 	return nil, 0, ErrCannotAddRevtrBatch
	// }
	// canDo := true
	res, err := con.Exec(revtrAddBatch, id)
	if err != nil {
		log.Error(err)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	bID, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	batchID := uint32(bID)
	batchLen := 50000
	var added []*pb.RevtrMeasurement
	for i := 0; i < len(batch); i+=batchLen {
		log.Infof("Inserting batch %d of revtr", i)
		maxIndex := i + batchLen
		if i + batchLen >= len(batch) {
			maxIndex = len(batch)
		}
		query := bytes.Buffer{}
		query.WriteString(revtrInitRevtrBulk)
		for _, rm := range batch[i:maxIndex] {
			src, _ := util.IPStringToInt32(rm.Src)
			dst, _ := util.IPStringToInt32(rm.Dst)
			query.WriteString(fmt.Sprintf("(%d, %d, \"%s\"),", 
					src, dst, rm.Label))
			// res, err := tx.Exec(revtrInitRevtr, src, dst)
			// if err != nil {
			// 	logError(tx.Rollback)
			// 	log.Error(err)
			// 	return nil, 0, ErrCannotAddRevtrBatch
			// }
		}
		//trim the last ,
		queryStr := query.String()
		queryStr = queryStr[0:len(queryStr)-1]
		//format all vals at once
		// log.Infof(queryStr)
		res, err = con.Exec(queryStr)
		if err != nil {
			log.Error(err)
			log.Error(queryStr)
			return nil, 0, err
		}
		nInserted := batchLen
		if i + batchLen >= len(batch) {
			nInserted = len(batch) - i 
		}
		insertedIds, err := dasql.GetIdsFromBulkInsert(int64(nInserted), res)
		if err != nil {
			return nil, 0, err
		} 
		// Now insert bulk ids 
		query = bytes.Buffer{}
		query.WriteString(revtrAddBatchRevtrBulk)
		for i, rm := range batch[i:maxIndex] {
			id := insertedIds[i]
			query.WriteString(fmt.Sprintf("(%d, %d),", 
					batchID, id))
			// id, err := res.LastInsertId()
			// if err != nil {
			// 	logError(tx.Rollback)
			// 	log.Error(err)
			// 	return nil, 0, ErrCannotAddRevtrBatch
			// }
			// _, err = tx.Exec(revtrAddBatchRevtr, batchID, uint32(id))
			// if err != nil {
			// 	logError(tx.Rollback)
			// 	log.Error(err)
			// 	return nil, 0, ErrCannotAddRevtrBatch
			// }
			rm.Id = uint32(id)
			added = append(added, rm)
		}
	
		//trim the last ,
		queryStr = query.String()
		queryStr = queryStr[0:len(queryStr)-1]
		//format all vals at once
		res, err = con.Exec(queryStr)
		if err != nil {
			log.Error(queryStr)
			return nil, 0, err
		}
	}
	

	// err = tx.Commit()
	// if err != nil {
	// 	logError(tx.Rollback)
	// 	log.Error(err)
	// 	return nil, 0, ErrCannotAddRevtrBatch
	// }
	
	return added, batchID, nil
}

// StoreRevtr stores a Revtr
func (r *Repo) StoreRevtr(rt pb.ReverseTraceroute) error {
	con := r.repo.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	src, _ := util.IPStringToInt32(rt.Src)
	dst, _ := util.IPStringToInt32(rt.Dst)
	res, err := tx.Exec(revtrStoreRevtr, src, dst,
		rt.Runtime, rt.StopReason,
		rt.Status.String(), rt.FailReason)
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	for i, h := range rt.Path {
		hop, _ := util.IPStringToInt32(h.Hop)
		_, err := tx.Exec(revtrStoreRevtrHop, id, hop, h.Type, i, h.MeasurementId)
		if err != nil {
			log.Error(err)
			logError(tx.Rollback)
			return err
		}
	}
	rrdur, _ := ptypes.Duration(rt.Stats.RrDuration)
	tsdur, _ := ptypes.Duration(rt.Stats.TsDuration)
	trdur, _ := ptypes.Duration(rt.Stats.TrToSrcDuration)
	asdur, _ := ptypes.Duration(rt.Stats.AssumeSymmetricDuration)
	bgtrdur, _ := ptypes.Duration(rt.Stats.BackgroundTrsDuration)
	_, err = tx.Exec(revtrStoreStats, id, rt.Stats.RrProbes, rt.Stats.SpoofedRrProbes,
		rt.Stats.TsProbes, rt.Stats.SpoofedTsProbes, rt.Stats.RrRoundCount,
		rrdur.Nanoseconds(), rt.Stats.TsRoundCount, tsdur.Nanoseconds(),
		rt.Stats.TrToSrcRoundCount, trdur.Nanoseconds(),
		rt.Stats.AssumeSymmetricRoundCount, asdur.Nanoseconds(),
		rt.Stats.BackgroundTrsRoundCount, bgtrdur.Nanoseconds())
	if err != nil {
		log.Error(err)
		if err := tx.Rollback(); err != nil {
			log.Error(err)
		}
		return ErrFailedToStoreBatch
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	return nil
}

var (
	// ErrNoRevtrUserFound is returned when no user is found with the given key
	ErrNoRevtrUserFound = fmt.Errorf("No user found")
)

// GetUserByKey gets a reverse traceroute user with the given key
func (r *Repo) GetUserByKey(key string) (pb.RevtrUser, error) {
	con := r.repo.GetReader()
	res := con.QueryRow(revtrGetUserByKey, key)
	var ret pb.RevtrUser
	err := res.Scan(&ret.Id, &ret.Name, &ret.Email, &ret.Max, &ret.Delay, &ret.Key, 
		&ret.MaxRevtrPerDay, &ret.RevtrRunToday, &ret.MaxParallelRevtr)
	switch {
	case err == sql.ErrNoRows:
		return ret, ErrNoRevtrUserFound
	case err != nil:
		log.Error(err)
		return ret, err
	default:
		return ret, nil
	}
}

const (
	selectByAddressAndDest24AdjDstQuery = `
	SELECT dest24, address, adjacent, cnt
	FROM adjacencies_to_dest 
	WHERE address = ? AND dest24 = ?
	ORDER BY cnt DESC LIMIT 500`
	selectByIP1AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip1 = ?
							ORDER BY cnt DESC LIMIT 500`
	selectByIP2AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip2 = ?
							ORDER BY cnt DESC LIMIT 500`

	aliasGetByIP          = `SELECT cluster_id FROM traceroute_atlas.ip_aliases WHERE ip_address = ? LIMIT 1`
	aliasGetIPsForCluster = `SELECT ip_address FROM traceroute_atlas.ip_aliases WHERE cluster_id = ? LIMIT 2000`
)

// GetAdjacenciesByIP1 gets ajds by ip1
func (r *Repo) GetAdjacenciesByIP1(ip uint32) ([]types.Adjacency, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByIP1AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	var adjs []types.Adjacency
	for res.Next() {
		var adj types.Adjacency
		err := res.Scan(&adj.IP1, &adj.IP2, &adj.Cnt)
		if err != nil {
			return nil, err
		}
		adjs = append(adjs, adj)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return adjs, nil
}

// GetAdjacenciesByIP2 gets ajds by ip2
func (r *Repo) GetAdjacenciesByIP2(ip uint32) ([]types.Adjacency, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByIP2AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	var adjs []types.Adjacency
	for res.Next() {
		var adj types.Adjacency
		err := res.Scan(&adj.IP1, &adj.IP2, &adj.Cnt)
		if err != nil {
			return nil, err
		}
		adjs = append(adjs, adj)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return adjs, nil
}

// GetAdjacencyToDestByAddrAndDest24 does what it says
func (r *Repo) GetAdjacencyToDestByAddrAndDest24(dest24, addr uint32) ([]types.AdjacencyToDest, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByAddressAndDest24AdjDstQuery, addr, dest24)
	if err != nil {
		return nil, err

	}
	defer logError(res.Close)
	var adjs []types.AdjacencyToDest
	for res.Next() {
		var adj types.AdjacencyToDest
		err = res.Scan(&adj.Dest24, &adj.Address, &adj.Adjacent, &adj.Cnt)
		if err != nil {
			return nil, err

		}
		adjs = append(adjs, adj)

	}
	return adjs, nil

}

var (
	// ErrNoAlias is returned when no alias is found for an ip
	ErrNoAlias = fmt.Errorf("No alias found")
)

// GetClusterIDByIP gets a the cluster ID for a give ip
func (r *Repo) GetClusterIDByIP(ip uint32) (int, error) {
	con := r.repo.GetReader()
	var ret int
	err := con.QueryRow(aliasGetByIP, ip).Scan(&ret)
	switch {
	case err == sql.ErrNoRows:
		return ret, ErrNoAlias
	case err != nil:
		return ret, err
	default:
		return ret, nil
	}
}

// GetIPsForClusterID gets all IPs associated with the given cluster id
func (r *Repo) GetIPsForClusterID(id int) ([]uint32, error) {
	con := r.repo.GetReader()
	var scan uint32
	var ret []uint32
	res, err := con.Query(aliasGetByIP, id)
	defer logError(res.Close)
	if err != nil {
		return nil, err
	}
	for res.Next() {
		err := res.Scan(&scan)
		if err != nil {
			return nil, err
		}
		ret = append(ret, scan)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}