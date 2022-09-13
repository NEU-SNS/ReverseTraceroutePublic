package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
)

// Repo is a repository for storing and querying for vantage points
type Repo struct {
	repo *repository.DB
}

// Configs is a group of DB Configs
type Configs struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

// Config is a database configuration
type Config repository.Config

type repoOptions struct {
	writeConfigs []Config
	readConfigs  []Config
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
	return &Repo{repo: db}, nil
}

type errorf func() error

func logError(e errorf) {
	if err := e(); err != nil {
		log.Error(err)
	}
}

const (
	getVPS = `select 
    vps.ip, vps.hostname, vps.site, vps.timestamp, vps.record_route, vps.spoof, vps.rec_spoof, vps.ping, vps.trace 
from 
    vantage_points vps
    left outer join quarantined_vps qvps on vps.hostname = qvps.hostname
`
	getAllVPS = `select 
ip, hostname, site, 
timestamp, record_route, spoof, rec_spoof, ping, trace from vantage_points`
	updateVP = `
 REPLACE INTO  
 vantage_points (ip, hostname, site, timestamp, record_route, spoof, rec_spoof, ping, trace, last_check)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now())
`
// set hostname = ?,
//   site = ?,
//   timestamp = ?,
//   record_route = ?,
//   spoof = ?,
//   rec_spoof = ?,
//   ping = ?,
//   trace = ?,
//   last_check = now()
// where ip = ?;
	getVPSToTest = `
SELECT  vps.ip, vps.hostname, vps.site, vps.timestamp, 
vps.record_route, vps.spoof, vps.rec_spoof, vps.ping, vps.trace  
FROM
vantage_points vps INNER JOIN plcontroller.vantage_point plvps ON plvps.ip=vps.ip 
WHERE plvps.controller is not NULL  
order by vps.last_check limit ?;
`
	getVPSForTesting = `
SELECT  vps.ip, vps.hostname, vps.site, vps.timestamp, 
vps.record_route, vps.spoof, vps.rec_spoof, vps.ping, vps.trace  
FROM
vantage_points vps  
WHERE vps.spoof = 1 AND vps.ping=1 AND vps.record_route=1 AND vps.trace=1 
order by vps.last_check limit ?;
`

	getQuarantinedVPs = `select ip, hostname, site, type, added, quarantine from quarantined_vps;`
)

func scanVPs(rows *sql.Rows) ([]*pb.VantagePoint, error) {
	var vps []*pb.VantagePoint
	for rows.Next() {
		cvp := new(pb.VantagePoint)
		err := rows.Scan(&cvp.Ip, &cvp.Hostname, &cvp.Site,
			&cvp.Timestamp, &cvp.RecordRoute, &cvp.Spoof,
			&cvp.RecSpoof, &cvp.Ping, &cvp.Trace)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		vps = append(vps, cvp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vps, nil
}

// GetVPs get all vps
func (r *Repo) GetVPs() ([]*pb.VantagePoint, error) {
	res, err := r.repo.GetReader().Query(getVPS)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	return scanVPs(res)
}

// GetAllVPs get all the vps including quarantines
func (r *Repo) GetAllVPs() ([]*pb.VantagePoint, error) {
	res, err := r.repo.GetReader().Query(getAllVPS)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	return scanVPs(res)
}

// GetVPsForTesting gets up to limit vps used for testing capabilities
func (r *Repo) GetVPsForTesting(limit int) ([]*pb.VantagePoint, error) {
	res, err := r.repo.GetReader().Query(getVPSForTesting, limit)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer logError(res.Close)
	return scanVPs(res)
}

// GetVPsToTest gets up to limit vps to test
func (r *Repo) GetVPsToTest(limit int) ([]*pb.VantagePoint, error) {
	res, err := r.repo.GetReader().Query(getVPSToTest, limit)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer logError(res.Close)
	return scanVPs(res)
}



type vpError struct {
	vp pb.VantagePoint
}

func (vpe vpError) Error() string {
	return fmt.Sprintf("failed to update vp %v", vpe.vp)
}

// UpdateVP updates the given vp matched by IP
func (r *Repo) UpdateVP(vp pb.VantagePoint) error {
	_, err := r.repo.GetWriter().Exec(updateVP,
		vp.Ip,
		vp.Hostname,
		vp.Site,
		vp.Timestamp,
		vp.RecordRoute,
		vp.Spoof,
		vp.RecSpoof,
		vp.Ping,
		vp.Trace,
		)

	if err != nil {
		return err
	}
	return nil
}

func vpNotIn(in, comp map[pb.VantagePoint]bool) []*pb.VantagePoint {
	var notIn []*pb.VantagePoint
	for vp, i := range comp {
		if !i {
			continue
		}
		if !in[vp] {
			add := vp
			notIn = append(notIn, &add)
		}
	}
	return notIn
}

func generateChanges(new, old []*pb.VantagePoint) ([]*pb.VantagePoint, []*pb.VantagePoint) {
	var remove, add []*pb.VantagePoint
	oldm := make(map[pb.VantagePoint]bool)
	newm := make(map[pb.VantagePoint]bool)
	for _, vp := range new {
		newm[*vp] = true
	}
	for _, vp := range old {
		// the already existings vps might have spoof/rr/ts set to true
		// set them back to false so they can be used as a key (new will have all false)
		curr := *vp
		curr.Spoof = false
		curr.Timestamp = false
		curr.RecSpoof = false
		curr.RecordRoute = false
		curr.Ping = false
		curr.Trace = false
		oldm[curr] = true
	}
	add = vpNotIn(oldm, newm)
	remove = vpNotIn(newm, oldm)
	return remove, add
}

// ErrFailedToUpdateVPs is returned when UpdateActiveVPs failed to retreive vps from the db
var ErrFailedToUpdateVPs = fmt.Errorf("failed to update vps, could not read vantage points")

type addVPError struct {
	vp pb.VantagePoint
}

func (ae addVPError) Error() string {
	return fmt.Sprintf("failed to insert vp: %v", ae.vp)
}

// GetQuarantined gets all the quarantined VPs of all quarantined vps
func (r *Repo) GetQuarantined() ([]types.Quarantine, error) {
	rows, err := r.repo.GetReader().Query(getQuarantinedVPs)
	if err != nil {
		return nil, err
	}
	var quar []types.Quarantine
	defer logError(rows.Close)
	for rows.Next() {
		var ip uint32
		var hostname, site, typen string
		var added time.Time
		var quarantine []byte
		err := rows.Scan(&ip, &hostname, &site, &typen, &added, &quarantine)
		if err != nil {
			return nil, err
		}
		q, err := types.GetQuarantine(types.QuarantineType(typen), quarantine)
		if err != nil {
			log.Error(err)
			return nil, fmt.Errorf("Failed to load Quarantines")
		}
		quar = append(quar, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return quar, nil
}

// UpdateActiveVPs updates the active vps in the database
func (r *Repo) UpdateActiveVPs(vps []*pb.VantagePoint) ([]*pb.VantagePoint, []*pb.VantagePoint, error) {
	ovps, err := r.GetAllVPs()
	if err != nil {
		log.Error(err)
		return nil, nil, ErrFailedToUpdateVPs
	}
	rem, add := generateChanges(vps, ovps)
	log.Debug("Added: ", add)
	log.Debug("Removing: ", rem)
	tx, err := r.repo.GetWriter().Begin()
	if err != nil {
		return nil, nil, err
	}
	for _, vp := range add {
		if err := addVP(tx, vp); err != nil {
			logError(tx.Rollback)
			return nil, nil, err
		}
		if err := insertEvent(tx, "ONLINE", vp); err != nil {
			logError(tx.Rollback)
			return nil, nil, err
		}
	}
	for _, vp := range rem {
		if err := delVP(tx, vp); err != nil {
			logError(tx.Rollback)
			return nil, nil, err
		}
		if err := insertEvent(tx, "OFFLINE", vp); err != nil {
			logError(tx.Rollback)
			return nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		log.Error(err)
		return nil, nil, err
	}
	return add, rem, nil
}

const (
	insertVP = `REPLACE into vantage_points(ip, hostname, site) values(?, ?, ?)`
	removeVP = `delete from vantage_points where ip = ?`
	addEvent = `insert into vp_events(type, hostname, site) values(?,?,?)`
)

func insertEvent(tx *sql.Tx, event string, vp *pb.VantagePoint) error {
	_, err := tx.Exec(addEvent, event, vp.Hostname, vp.Site)
	if err != nil {
		return err
	}
	return nil
}

func delVP(tx *sql.Tx, vp *pb.VantagePoint) error {
	_, err := tx.Exec(removeVP, vp.Ip)
	if err != nil {
		return err
	}
	return nil
}

func addVP(tx *sql.Tx, vp *pb.VantagePoint) error {
	res, err := tx.Exec(insertVP, vp.Ip, vp.Hostname, vp.Site)
	if err != nil {
		log.Errorf("%s %s %s %s", insertVP, vp.Ip, vp.Hostname, vp.Site)
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return addVPError{*vp}
	}
	return nil
}

const (
	getRRSpoofers = `
SELECT
    vps.ip,
    vps.hostname,
    vps.site,
    vps.timestamp,
    vps.record_route,
    vps.spoof,
    vps.rec_spoof,
    MAX(IFNULL(dtd.dist, ~0 >>32)) AS dist
FROM
    vantage_points vps
    LEFT OUTER JOIN quarantined_vps qvps ON qvps.hostname = vps.hostname
    LEFT OUTER JOIN (
        SELECT
            dtd.src, dtd.dist
        FROM
            dist_to_dest dtd
        WHERE
            dtd.slash_24 = (? >> 8)
        ) dtd ON dtd.src = vps.ip
WHERE
    vps.record_route AND
    vps.spoof AND
    qvps.hostname is null
GROUP BY
    vps.ip,
    vps.hostname,
    vps.site,
    vps.timestamp,
    vps.record_route,
    vps.spoof,
    vps.rec_spoof
ORDER BY
    dist
`
	getTSSpoofers = `
SELECT
    vps.ip,
    vps.hostname,
    vps.site,
    vps.timestamp,
    vps.record_route,
    vps.spoof,
    vps.rec_spoof
FROM
    vantage_points vps
    LEFT OUTER JOIN quarantined_vps qvps ON qvps.hostname = vps.hostname
WHERE
    vps.timestamp 
    AND vps.spoof
    AND qvps.hostname is null
`
)


// GetRRSpoofers gets vantage points usable for target target
func (r *Repo) GetRRSpoofers(target uint32) ([]types.RRVantagePoint, error) {
	res, err := r.repo.GetReader().Query(getRRSpoofers, target)
	//res, err := r.repo.GetReader().Query(getAllSpoofers)
	if err != nil {
		return nil, err
	}
	var rrvps []types.RRVantagePoint
	defer logError(res.Close)
	for res.Next() {
		rrvp := new(types.RRVantagePoint)
		rrvp.Target = target
		err := res.Scan(&rrvp.Ip,
			&rrvp.Hostname,
			&rrvp.Site,
			&rrvp.Timestamp,
			&rrvp.RecordRoute,
			&rrvp.Spoof,
			&rrvp.RecSpoof,
			&rrvp.Dist)
		if err != nil {
			return nil, err
		}
		rrvps = append(rrvps, *rrvp)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return rrvps, nil
}

// GetTSSpoofers gets vantage points usable for target target up to limit vps
func (r *Repo) GetTSSpoofers() ([]types.TSVantagePoint, error) {
	res, err := r.repo.GetReader().Query(getTSSpoofers)
	if err != nil {
		return nil, err
	}
	var tsvps []types.TSVantagePoint
	defer logError(res.Close)
	for res.Next() {
		tsvp := new(types.TSVantagePoint)
		err := res.Scan(&tsvp.Ip,
			&tsvp.Hostname,
			&tsvp.Site,
			&tsvp.Timestamp,
			&tsvp.RecordRoute,
			&tsvp.Spoof,
			&tsvp.RecSpoof)
		if err != nil {
			return nil, err
		}
		tsvps = append(tsvps, *tsvp)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return tsvps, nil
}

const (
	unquarantinevp = `delete from quarantined_vps where ip = ?;`
	quarantinevp   = `insert ignore into quarantined_vps(ip, hostname, site, type, quarantine) 
values(?, ?, ?, ?, ?);`
	quarantineEvent = `insert into quarantine_events(type, ip, hostname, site, quarantine_type, quarantine) 
 values(?, ?, ?, ?, ?, ?);`
	updateQuarantine = `
update quarantined_vps 
set quarantine = ? 
where ip = ?;`
)

// UnquarantineVPs removes the vps in vps from quarantine
func (r *Repo) UnquarantineVPs(vps []types.Quarantine) error {
	tx, err := r.repo.GetWriter().Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(unquarantinevp)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(err)
		}
	}()
	for _, vp := range vps {
		avp := vp.GetVP()
		res, err := stmt.Exec(vp.GetVP().Ip)
		if err != nil {
			logError(tx.Rollback)
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			log.Error(err)
			continue
		}
		if rows > 0 {
			quar, err := json.Marshal(vp)
			if err != nil {
				logError(tx.Rollback)
				return err
			}
			_, err = tx.Exec(quarantineEvent, "REMOVED", avp.Ip,
				avp.Hostname, avp.Site, string(vp.Type()), quar)
			if err != nil {
				logError(tx.Rollback)
				return err
			}
		}
	}
	return tx.Commit()
}

// QuarantineVPs adds the vps in vps to quarantine
func (r *Repo) QuarantineVPs(vps []types.Quarantine) error {
	tx, err := r.repo.GetWriter().Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(quarantinevp)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(err)
		}
	}()
	for _, vp := range vps {
		avp := vp.GetVP()
		data, err := json.Marshal(vp)
		if err != nil {
			log.Error(err)
			logError(tx.Rollback)
			return err
		}
		res, err := stmt.Exec(avp.Ip, avp.Hostname, avp.Site, string(vp.Type()), data)
		if err != nil {
			logError(tx.Rollback)
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			log.Error(err)
			continue
		}
		if rows > 0 {
			quar, err := json.Marshal(vp)
			if err != nil {
				logError(tx.Rollback)
				return err
			}
			_, err = tx.Exec(quarantineEvent, "ADDED", avp.Ip,
				avp.Hostname, avp.Site, string(vp.Type()), quar)
			if err != nil {
				logError(tx.Rollback)
				return err
			}
		}
	}
	return tx.Commit()
}

var (
	// ErrFailedToUpdateQuarantines is a failure to update quarantines
	ErrFailedToUpdateQuarantines = fmt.Errorf("Could not update quarantines")
)

// UpdateQuarantines update the given quarantines
func (r *Repo) UpdateQuarantines(qs []types.Quarantine) error {
	tx, err := r.repo.GetWriter().Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	stmt, err := tx.Prepare(updateQuarantine)
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return ErrFailedToUpdateQuarantines
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(err)
		}
	}()
	for _, q := range qs {
		data, err := json.Marshal(q)
		if err != nil {
			log.Error(err)
			logError(tx.Rollback)
			return ErrFailedToUpdateQuarantines
		}
		_, err = stmt.Exec(data, q.GetVP().Ip)
		if err != nil {
			log.Errorf("%s %d %s", updateQuarantine, q.GetVP().Ip, string(data))
			log.Error(err)
			logError(tx.Rollback)
			return ErrFailedToUpdateQuarantines
		}
	}
	return tx.Commit()
}

const (
	getLastestQuarantine = `
SELECT qe.type, qe.ip, qe.hostname, qe.site, qe.quarantine_type, qe.quarantine 
FROM quarantine_events qe 
WHERE qe.ip = ? 
ORDER BY time 
LIMIT 1;
`
)

var (
	// ErrNoQuarantine is returened when there is no Quarantine to return from
	// GetLastQuarantine
	ErrNoQuarantine = fmt.Errorf("No Quarantine Found")
)

// GetLastQuarantine gets the most recent quarantine event for the given ip
func (r *Repo) GetLastQuarantine(ip uint32) (types.Quarantine, error) {
	row := r.repo.GetReader().QueryRow(getLastestQuarantine, ip)
	var qet, hostname, site, qt string
	var ipaddr uint32
	var quar []byte
	err := row.Scan(&qet, &ipaddr, &hostname, &site, &qt, &quar)
	if err == sql.ErrNoRows {
		return nil, ErrNoQuarantine
	}
	if err != nil {
		return nil, err
	}
	q, err := types.GetQuarantine(types.QuarantineType(qt), quar)
	if err != nil {
		return nil, err
	}
	return q, nil
}
