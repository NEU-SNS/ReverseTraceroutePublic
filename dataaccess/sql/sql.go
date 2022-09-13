/*
Copyright (c) 2015, Northeastern University

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package sql provides a sql data provider for reverse traceroute
package sql

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/go-sql-driver/mysql"
)

/*
This is just duplicate code while for the period of transition
*/

// DB is the data access object
type DB struct {
	*repository.DB
}

// DbConfig is the database config
type DbConfig struct {
	WriteConfigs []Config
	ReadConfigs  []Config
	Environment string 
}

// Config is the configuration for an indivual database
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

var pingTable = "pings"
var pingResponseTable = "ping_responses"
var pingStatsTable = "ping_stats"
var pingRRTable = "record_routes"
var pingTSTable = "timestamps"
var pingTSAddrTable = "timestamp_addrs"
var pingICMPTSTable = "icmp_timestamps"
var tracerouteTable = "traceroutes"
var tracerouteHopsTable = "traceroute_hops" 

var tables = []*string{&pingTable, &pingResponseTable, &pingStatsTable, &pingRRTable, 
	&pingTSTable, &pingTSAddrTable, &pingICMPTSTable,
	 &tracerouteTable, &tracerouteHopsTable}

// NewDB creates a db object with the given config
func NewDB(con DbConfig) (*DB, error) {
	var conf repository.DbConfig
	conf.Environment = con.Environment
	for _, wc := range con.WriteConfigs {
		conf.WriteConfigs = append(conf.WriteConfigs, repository.Config(wc))
	}
	for _, rc := range con.ReadConfigs {
		conf.ReadConfigs = append(conf.ReadConfigs, repository.Config(rc))
	}
	db, err := repository.NewDB(conf)
	if err != nil {
		return nil, err
	}
	
	if conf.Environment == "debug" {
		for _, table :=range(tables) {
			*table +=  "_debug"
		}
	}

	initTraceReadQueries()
	initTraceWriteQueries()
	initPingReadQueries()
	initPingWriteQueries()
	initPingBatchQueries()
	

	return &DB{db}, nil
}

// VantagePoint represents a vantage point
type VantagePoint struct {
	IP           uint32
	Controller   sql.NullInt64
	HostName     string
	Site         string
	TimeStamp    bool
	RecordRoute  bool
	CanSpoof     bool
	Active       bool
	ReceiveSpoof bool
	LastUpdated  time.Time
	SpoofChecked mysql.NullTime
	Port         uint32
}

// ToDataModel converts a sql.VantagePoint to a dm.VantagePoint
func (vp *VantagePoint) ToDataModel() *dm.VantagePoint {
	nvp := &dm.VantagePoint{}
	nvp.Ip = vp.IP
	if vp.Controller.Valid {
		nvp.Controller = uint32(vp.Controller.Int64)
	}
	nvp.Hostname = vp.HostName
	nvp.Timestamp = vp.TimeStamp
	nvp.RecordRoute = vp.TimeStamp
	nvp.CanSpoof = vp.CanSpoof
	nvp.ReceiveSpoof = vp.ReceiveSpoof
	nvp.LastUpdated = vp.LastUpdated.Unix()
	nvp.Site = vp.Site
	if vp.SpoofChecked.Valid {
		nvp.SpoofChecked = vp.LastUpdated.Unix()
	}
	nvp.Port = vp.Port
	return nvp
}

const (
	getVpsQuery string = `
SELECT
	ip, controller, hostname, timestamp,
	record_route, can_spoof,
    receive_spoof, last_updated, port, site
FROM
	vantage_point;
`
	getActiveVpsQuery string = `
SELECT
	ip, controller, hostname, timestamp,
	record_route, can_spoof,
    receive_spoof, last_updated, port, site
FROM
	vantage_point
WHERE
	controller is not null;
`
)

// GetVPs gets all the VPs in the database
func (db *DB) GetVPs() ([]*dm.VantagePoint, error) {
	rows, err := db.GetReader().Query(getVpsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vps []*dm.VantagePoint
	for rows.Next() {
		vp := &VantagePoint{}
		err := rows.Scan(
			&vp.IP,
			&vp.Controller,
			&vp.HostName,
			&vp.TimeStamp,
			&vp.RecordRoute,
			&vp.CanSpoof,
			&vp.ReceiveSpoof,
			&vp.LastUpdated,
			&vp.Port,
			&vp.Site,
		)
		if err != nil {
			return vps, err
		}
		vps = append(vps, vp.ToDataModel())
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return vps, nil
}

// GetActiveVPs gets all the VPs in the database that are connected
func (db *DB) GetActiveVPs() ([]*dm.VantagePoint, error) {
	rows, err := db.GetReader().Query(getActiveVpsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vps []*dm.VantagePoint
	for rows.Next() {
		vp := &VantagePoint{}
		err := rows.Scan(
			&vp.IP,
			&vp.Controller,
			&vp.HostName,
			&vp.TimeStamp,
			&vp.RecordRoute,
			&vp.CanSpoof,
			&vp.ReceiveSpoof,
			&vp.LastUpdated,
			&vp.Port,
			&vp.Site,
		)
		if err != nil {
			return vps, err
		}
		vps = append(vps, vp.ToDataModel())
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return vps, nil
}

const (

	insertVantagePointQuery string = `
	INSERT IGNORE INTO 
	vantage_point (ip, controller)  VALUES (?, ?)
`
	removeVantagePointQuery string = `
	DELETE FROM vantage_points WHERE ip = ? 
`

	getOwnerQuery string = `
	SELECT owner, email, hostname, site FROM vantage_point WHERE ip = ?
`

	updateControllerQuery string = `
UPDATE
	vantage_point
SET
	controller = ?
WHERE
	ip = ?
`
	updateControllerQueryNull string = `
UPDATE
	vantage_point
SET
	controller = IF(controller = ?, NULL, controller)
WHERE
	ip = ?
`
	clearAllVps string = `
UPDATE
        vantage_point
SET
        controller = NULL
WHERE
        controller is not null
`
)

// ClearAllVPs sets all vps in the plcontroller db to offline
func (db *DB) ClearAllVPs() error {
	_, err := db.GetWriter().Exec(clearAllVps)
	if err != nil {
		return err
	}
	return nil
}

// 
func (db *DB) InsertVP(ip uint32 , con uint32) error {
	query := insertVantagePointQuery
	args := [] interface{} {ip, con}
	_, err := db.GetWriter().Exec(query, args...)
	return err
}

func (db *DB) RemoveVantagePoint(ip uint32) error {
	query := removeVantagePointQuery
	args := [] interface{} {ip}
	_, err := db.GetWriter().Exec(query, args...)
	return err
}

func (db *DB) GetVPOwner(ip uint32) (dm.VPOwner, error) {
	con := db.GetReader()
	args := [] interface {} {ip}
	row := con.QueryRow(getOwnerQuery, args...)
	var owner dm.VPOwner
	err := row.Scan(&owner.Name, &owner.Email, &owner.Hostname, &owner.Site)
	if err != nil {
		log.Error(err)
		return dm.VPOwner{}, err
	}
	return owner, nil
}

// UpdateController updates a vantage point's controller
func (db *DB) UpdateController(ip, newc, con uint32) error {
	var args []interface{}
	query := updateControllerQuery
	if newc == 0 {
		query = updateControllerQueryNull
		args = append(args, con)
	} else {
		args = append(args, newc)
	}
	args = append(args, ip)
	_, err := db.GetWriter().Exec(query, args...)
	return err
}

const (
	updateActiveQuery string = `
UPDATE
	vantage_point
SET
	active = ?
WHERE
	ip = ?
`
)

// UpdateActive updates the acive flag of a vantage point
func (db *DB) UpdateActive(ip uint32, active bool) error {
	_, err := db.GetWriter().Exec(updateActiveQuery, active, ip)
	return err
}

const (
	updateCanSpoofQuery string = `
UPDATE
	vantage_point
SET
	can_spoof = ?,
	spoof_checked = NOW()
WHERE
	ip = ?
`
)

// UpdateCanSpoof updates the can spoof flag for a vantage point
func (db *DB) UpdateCanSpoof(ip uint32, canSpoof bool) error {
	_, err := db.GetWriter().Exec(updateCanSpoofQuery, canSpoof, ip)
	return err
}

const (
	updateCheckStatus string = `
UPDATE
	vantage_point
SET
	last_health_check = ?
WHERE
	ip = ?
`
)

// UpdateCheckStatus updates the result of the health check for a vantage point
func (db *DB) UpdateCheckStatus(ip uint32, result string) error {
	_, err := db.GetReader().Exec(updateCheckStatus, result, ip)
	return err
}

var (
	insertTrace string 
	insertTraceBulk string 
	insertTraceHop string 
	insertTraceHopBulk string 
)

func initTraceWriteQueries() {
	insertTrace = fmt.Sprintf(`
INSERT INTO
%s(src, dst, type, user_id, method, sport, 
			dport, stop_reason, stop_data, ` + "`start`"+ `, 
			version, hop_count, attempts, hop_limit, 
			first_hop, wait, wait_probe, tos, probe_size, label)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, tracerouteTable)

insertTraceBulk = fmt.Sprintf(`INSERT INTO 
%s(src, dst, type, user_id, method, sport, 
	dport, stop_reason, stop_data, `+ "`start`"+ `, 
	version, hop_count, attempts, hop_limit, 
	first_hop, wait, wait_probe, tos, probe_size, revtr_measurement_id, label)
VALUES `, tracerouteTable)
insertTraceHop = fmt.Sprintf(`
INSERT INTO
%s(traceroute_id, hop, addr, probe_ttl, probe_id, 
				probe_size, rtt, reply_ttl, reply_tos, reply_size, 
				reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, tracerouteHopsTable)
insertTraceHopBulk = fmt.Sprintf(`
INSERT INTO
%s(traceroute_id, hop, addr, probe_ttl, probe_id, 
				probe_size, rtt, reply_ttl, reply_tos, reply_size, 
				reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos)
VALUES `, tracerouteHopsTable)
}

func storeTraceroute(tx *sql.Tx, in *dm.Traceroute) (int64, error) {
	start := time.Unix(in.Start.Sec, in.Start.Usec*1000)
	res, err := tx.Exec(insertTrace, in.Src, in.Dst, in.Type,
		in.UserId, in.Method, in.Sport, in.Dport,
		in.StopReason, in.StopData, start,
		in.Version, in.HopCount, in.Attempts,
		in.Hoplimit, in.Firsthop, in.Wait, in.WaitProbe,
		in.Tos, in.ProbeSize)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func storeTracerouteBulk(tx* sql.Tx, ins []*dm.Traceroute) ([]int64, error) {
	query := bytes.Buffer{}
	query.WriteString(insertTraceBulk)

	for _, in := range(ins) {
		
		start := time.Unix(in.Start.Sec, in.Start.Usec*1000).Format(environment.DateFormat)
		// start := time.Unix(in.Start.Sec, in.Start.Usec*1000).Unix()

		// start := in.Start.Sec
		query.WriteString(fmt.Sprintf(`
		   (%d, %d, "%s", %d, "%s", %d, %d,
			"%s", %d, "%s", "%s", %d, %d,
			%d, %d, %d, %d, %d, %d, %d, "%s"),`, 
			in.Src, in.Dst, in.Type, in.UserId, in.Method, in.Sport, in.Dport,
			in.StopReason, in.StopData, start, in.Version, in.HopCount, in.Attempts,
			in.Hoplimit, in.Firsthop, in.Wait, in.WaitProbe, in.Tos, in.ProbeSize, in.RevtrMeasurementId, in.Label))
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	res, err := tx.Exec(queryStr)

	if err != nil {
		log.Error(err)
		log.Error(queryStr)
		return nil, err
	}
	insertedIds, err := GetIdsFromBulkInsert(int64(len(ins)), res)
	if err != nil {
		return nil, err
	} 
	return insertedIds, nil

}

func storeTraceHop(tx *sql.Tx, id int64, hop uint32, in *dm.TracerouteHop) error {
	rtt := in.GetRtt()
	var inRtt uint32
	if rtt != nil {
		/*
			I would generally say this cast is not safe, but we know that it will fit in
			a uint32 because that's what we got it from
		*/
		inRtt = uint32(rtt.Sec)*1000000 + uint32(rtt.Usec)
	}
	_, err := tx.Exec(insertTraceHop, id, hop, in.Addr, in.ProbeTtl,
		in.ProbeId, in.ProbeSize, inRtt, in.ReplyTtl,
		in.ReplyTos, in.ReplySize, in.ReplyIpid,
		in.IcmpType, in.IcmpCode, in.IcmpQTtl,
		in.IcmpQIpl, in.IcmpQTos)
	return err
}

func storeTraceroutesHopsBulk(tx* sql.Tx, ids [] int64, ins []*dm.Traceroute) (error) {

	query := bytes.Buffer{}
	query.WriteString(insertTraceHopBulk)
	noHopsInserted := true
	for t, in := range(ins) {
		if len(in.GetHops()) == 0 {
			continue
		}
		noHopsInserted = false
		for i, hop := range in.GetHops() {
			
			rtt := hop.GetRtt()
			var inRtt uint32
			if rtt != nil {
				/*
					I would generally say this cast is not safe, but we know that it will fit in
					a uint32 because that's what we got it from
				*/
				inRtt = uint32(rtt.Sec)*1000000 + uint32(rtt.Usec)
			}
			query.WriteString(fmt.Sprintf(`(%d, %d, %d, %d,
				%d, %d, %d, %d,
				%d, %d, %d,
				%d, %d, %d,
				%d, %d),`, 
			ids[t], i, hop.Addr, hop.ProbeTtl,
			hop.ProbeId, hop.ProbeSize, inRtt, hop.ReplyTtl,
			hop.ReplyTos, hop.ReplySize, hop.ReplyIpid,
			hop.IcmpType, hop.IcmpCode, hop.IcmpQTtl,
			hop.IcmpQIpl, hop.IcmpQTos))
	
		}
	}

	if noHopsInserted {
		return nil 
	}

	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	if err != nil {
		log.Error(err)
		log.Error(queryStr)
		return err
	} 
	return nil

}

func (db *DB) StoreTracerouteBulk(ins []*dm.Traceroute) ([]int64, error) {
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return nil, err
	}

	ids, err := storeTracerouteBulk(tx, ins)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = storeTraceroutesHopsBulk(tx, ids, ins)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return ids, nil 

}

// StoreTraceroute saves a traceroute to the DB
func (db *DB) StoreTraceroute(in *dm.Traceroute) (int64, error) {
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	

	id, err := storeTraceroute(tx, in)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if len(in.GetHops()) == 0 {
		tx.Commit()
		return id, nil
	}
	query := bytes.Buffer{}
	query.WriteString(insertTraceHopBulk)
	for i, hop := range in.GetHops() {
		
		rtt := hop.GetRtt()
		var inRtt uint32
		if rtt != nil {
			/*
				I would generally say this cast is not safe, but we know that it will fit in
				a uint32 because that's what we got it from
			*/
			inRtt = uint32(rtt.Sec)*1000000 + uint32(rtt.Usec)
		}
		query.WriteString(fmt.Sprintf(`(%d, %d, %d, %d,
			%d, %d, %d, %d,
			%d,%d, %d,
			%d, %d, %d,
			%d, %d),`, 
		id, i, hop.Addr, hop.ProbeTtl,
		hop.ProbeId, hop.ProbeSize, inRtt, hop.ReplyTtl,
		hop.ReplyTos, hop.ReplySize, hop.ReplyIpid,
		hop.IcmpType, hop.IcmpCode, hop.IcmpQTtl,
		hop.IcmpQIpl, hop.IcmpQTos))

	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err = tx.Exec(queryStr)
	if err != nil {
		tx.Rollback()
		return 0, err
	} 
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return id, nil
}

var (
	getTraceBySrcDst string
	getTraceBySrcDstStale string
	addTraceBatch string
	addTraceBatchTrace string
	getTraceBatch0 string
	getTraceBatch string
	getTracerouteByLabel string 
)


func initTraceReadQueries() {
	getTraceBySrcDst = fmt.Sprintf(`
	SELECT 
	t.id, src, dst, type, user_id, method, sport, dport, stop_reason, stop_data, ` + "`start`"+ ` , version,
	hop_count, attempts, hop_limit, first_hop, wait, wait_probe, tos, t.probe_size, traceroute_id, 
	hop, addr, probe_ttl, probe_id, th.probe_size, rtt, reply_ttl, reply_tos, reply_size, 
	reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos
	FROM
	(
		SELECT 
			*
		FROM
			%s tt
		WHERE
			tt.src = ? and tt.dst = ?
		ORDER BY
			tt.start DESC
		LIMIT 1
	) t left outer join
	%s th on th.traceroute_id = t.id
	ORDER BY
	t.start DESC
`, tracerouteTable, tracerouteHopsTable)
	getTraceBySrcDstStale = `
SELECT 
	t.id, src, dst, type, user_id, method, sport, dport, stop_reason, stop_data, start, version,
    hop_count, attempts, hop_limit, first_hop, wait, wait_probe, tos, t.probe_size, traceroute_id, 
    hop, addr, probe_ttl, probe_id, th.probe_size, rtt, reply_ttl, reply_tos, reply_size, 
	reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos
FROM
	(
		SELECT 
			*
		FROM
			%s tt
		WHERE
			tt.src = %d and tt.dst = %d
		ORDER BY
			tt.start DESC
		LIMIT 1
	) t left outer join
	%s th on th.traceroute_id = t.id
WHERE t.start >= DATE_SUB(NOW(), interval %d minute)
ORDER BY
	t.start DESC
`

	addTraceBatch      = `insert into trace_batch(user_id) VALUES(?)`
	addTraceBatchTrace = `insert into trace_batch_trace(batch_id, trace_id) VALUES(?, ?)`
	getTraceBatch0     = "SELECT t.id, t.src, t.dst, t.type, t.user_id, t.method, t.sport, " +
		"t.dport, t.stop_reason, t.stop_data, t.start, t.version, t.hop_count, t.attempts, " +
		"t.hop_limit, t.first_hop, t.wait, t.wait_probe, t.tos, t.probe_size  " +
		"FROM users u inner join trace_batch tb on tb.user_id = u.id inner join " +
		"inner join trace_batch_trace tbt on tb.id = tbt.batch_id inner join " +
		"traceroutes t on t.id = tbt.trace_id WHERE u.`key` = ? and tb.id = ?;"
	getTraceBatch = "SELECT t.id, t.src, t.dst, t.type, t.user_id, t.method, " +
		"t.sport, t.dport, t.stop_reason, t.stop_data, t.start, t.version, " +
		"t.hop_count, t.attempts, t.hop_limit, t.first_hop, t.wait, t.wait_probe, " +
		"t.tos, t.probe_size, th.traceroute_id, " +
		"th.hop, th.addr, th.probe_ttl, th.probe_id, th.probe_size, " +
		"th.rtt, th.reply_ttl, th.reply_tos, th.reply_size, " +
		"th.reply_ipid, th.icmp_type, th.icmp_code, th.icmp_q_ttl, th.icmp_q_ipl, th.icmp_q_tos " +
		"FROM " +
		"users u " +
		"inner join trace_batch tb on u.id = tb.user_id " +
		"inner join trace_batch_trace tbt on tb.id = tbt.batch_id " +
		"inner join traceroutes t on tbt.trace_id = t.id " +
		"inner join traceroute_hops th on th.traceroute_id = t.id " +
		"WHERE " +
		"u.`key` = ? and tb.id = ?; "

	getTracerouteByLabel = fmt.Sprintf("SELECT t.id, t.src, t.dst, t.type, t.user_id, t.method, " +
	"t.sport, t.dport, t.stop_reason, t.stop_data, t.start, t.version, " +
	"t.hop_count, t.attempts, t.hop_limit, t.first_hop, t.wait, t.wait_probe, " +
	"t.tos, t.probe_size, th.traceroute_id, " +
	"th.hop, th.addr, th.probe_ttl, th.probe_id, th.probe_size, " +
	"th.rtt, th.reply_ttl, th.reply_tos, th.reply_size, " +
	"th.reply_ipid, th.icmp_type, th.icmp_code, th.icmp_q_ttl, th.icmp_q_ipl, th.icmp_q_tos " +
	"FROM %s t " +
	"INNER join %s th on th.traceroute_id = t.id " +
	"WHERE " +
	"label = ? ", tracerouteTable, tracerouteHopsTable)
}

// AddTraceBatch adds a traceroute batch
func (db *DB) AddTraceBatch(u dm.User) (int64, error) {
	res, err := db.GetWriter().Exec(addTraceBatch, u.ID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddTraceToBatch adds pings pids to batch bid
func (db *DB) AddTraceToBatch(bid int64, tids []int64) error {
	con := db.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	for _, tid := range tids {
		_, err := tx.Exec(addTraceBatchTrace, bid, tid)
		if err != nil {
			log.Error(err)
			return tx.Rollback()
		}
	}
	return tx.Commit()
}

// GetTraceBatch gets a batch of traceroute for user u with id bid
func (db *DB) GetTraceBatch(u dm.User, bid int64) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(getTraceBatch, u.Key, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

func (db *DB) GetTracerouteByLabel(label string) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(getTracerouteByLabel, label)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

func splitTraces(rows *sql.Rows) ([]*dm.Traceroute, error) {
	currTraces := make(map[int64]*dm.Traceroute)
	currHops := make(map[int64][]*dm.TracerouteHop)
	for rows.Next() {
		curr := &dm.Traceroute{}
		hop := &dm.TracerouteHop{}
		var start time.Time
		var id int64
		var tID sql.NullInt64
		var hopNum, rtt sql.NullInt64
		err := rows.Scan(&id, &curr.Src, &curr.Dst, &curr.Type, &curr.UserId, &curr.Method, &curr.Sport,
			&curr.Dport, &curr.StopReason, &curr.StopData, &start, &curr.Version, &curr.HopCount,
			&curr.Attempts, &curr.Hoplimit, &curr.Firsthop, &curr.Wait, &curr.WaitProbe, &curr.Tos,
			&curr.ProbeSize, &tID, &hopNum, &hop.Addr, &hop.ProbeTtl, &hop.ProbeId, &hop.ProbeSize,
			&rtt, &hop.ReplyTtl, &hop.ReplyTos, &hop.ReplySize, &hop.ReplyIpid, &hop.IcmpType,
			&hop.IcmpCode, &hop.IcmpQTtl, &hop.IcmpQIpl, &hop.IcmpQTos)
		if err != nil {
			return nil, err
		}
		if _, ok := currTraces[id]; !ok {
			curr.Start = &dm.TracerouteTime{}
			nano := start.UnixNano()
			curr.Start.Sec = nano / 1000000000
			curr.Start.Usec = (nano % 1000000000) / 1000
			currTraces[id] = curr
		}
		if tID.Valid {
			currHops[tID.Int64] = append(currHops[tID.Int64], hop)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var ret []*dm.Traceroute
	for key, val := range currTraces {
		if hops, ok := currHops[key]; ok {
			val.Hops = hops
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// GetTRBySrcDst gets traceroutes with the given src, dst
func (db *DB) GetTRBySrcDst(src, dst uint32) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(getTraceBySrcDst, src, dst)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

// GetTRBySrcDstWithStaleness gets a traceroute with the src/dst this is newer than s
func (db *DB) GetTRBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(fmt.Sprintf(getTraceBySrcDstStale, tracerouteTable, src, dst, tracerouteHopsTable, int(s.Minutes())))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

// GetTraceMulti gets traceroutes that match the given TracerouteMeasurements
func (db *DB) GetTraceMulti(in []*dm.TracerouteMeasurement) ([]*dm.Traceroute, error) {
	var ret []*dm.Traceroute
	for _, tm := range in {
		ts, err := db.GetTRBySrcDstWithStaleness(tm.Src, tm.Dst, time.Duration(tm.Staleness)*time.Minute)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ts...)
	}
	return ret, nil
}

var (
	getPing string 
	getPingStaleness string
	getPingStalenessRR string
	getPingResponses string 
	getPingStats string 
	getRecordRoutes string 
	getTimeStamps string 
	getTimeStampsAndAddr string 
	getMaxRevtrMeasurementID string 
)

func initPingReadQueries() {

	getPing = fmt.Sprintf("SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, " +
		"p.tsandaddr, p.icmptimestamp, p.icmpsum, dl, p.`8` " +
		"FROM %s p " +
		"WHERE p.src = ? and p.dst = ?;", pingTable)
	getPingStaleness = fmt.Sprintf("SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, p.tsandaddr, p.icmptimestamp, " +
		"p.icmpsum, dl, p.`8` " +
		"FROM %s p " +
		"WHERE p.src = ? and p.dst = ? and p.spoofed_from = ? and p.start >= DATE_SUB(NOW(), interval ? minute) order by p.start desc limit 1;", pingTable)
	getPingStalenessRR = fmt.Sprintf("SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, p.tsandaddr, p.icmptimestamp, " +
		"p.icmpsum, dl, p.`8` " +
		"FROM %s p " +
		"WHERE p.src = ? and p.dst = ? and p.spoofed_from = ? and p.record_route and p.start >= DATE_SUB(NOW(), interval ? minute) order by p.start desc limit 1;", pingTable)
	getPingResponses = fmt.Sprintf("SELECT pr.id, pr.ping_id, pr.`from`, pr.seq, " +
		"pr.reply_size, pr.reply_ttl, pr.rtt, pr.probe_ipid, pr.reply_ipid, " +
		"pr.icmp_type, pr.icmp_code, pr.tx, pr.rx " +
		"FROM %s pr " +
		"WHERE pr.ping_id = ?;", pingResponseTable)
	getPingStats = fmt.Sprintf("SELECT ps.loss, ps.min, " +
		"ps.max, ps.avg, ps.std_dev " +
		"FROM %s ps " +
		"WHERE ps.ping_id = ?;", pingStatsTable)
	getRecordRoutes = "SELECT rr.response_id, rr.hop, rr.ip " +
		"FROM %s rr " +
		"WHERE rr.response_id = %d ORDER BY rr.hop;"
	getTimeStamps = "SELECT ts.ts " +
		"FROM %s ts " +
		"WHERE ts.response_id = %d ORDER BY ts.`order`;"
	getTimeStampsAndAddr = "SELECT tsa.ip, tsa.ts " +
		"FROM %s tsa " +
		"WHERE tsa.response_id = %d ORDER BY tsa.`order`;"
	
	getMaxRevtrMeasurementID = "SELECT max(revtr_measurement_id) from %s;"
}

type rrHop struct {
	ResponseID sql.NullInt64
	Hop        uint8
	IP         uint32
}

type ts struct {
	ResponseID sql.NullInt64
	Order      uint8
	Ts         uint32
}

type tsAndAddr struct {
	ResponseID sql.NullInt64
	Order      uint8
	Ts         uint32
	IP         uint32
}

type icmpTimestamp struct {
	ResponseID sql.NullInt64
	OTs uint32
	RTs uint32
	TTs uint32
}

func makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight bool) []string {
	/*
		These are the posible keys for the map

		"v4rr"
		"spoof"
		"payload"
		"tsonly"
		"tsandaddr"
		"icmpsum"
		"dl"
		"8"
	*/
	var ret []string
	if spoofed {
		ret = append(ret, "spoof")
	}
	if recordRoute {
		ret = append(ret, "v4rr")
	}
	if payload {
		ret = append(ret, "payload")
	}
	if tsonly {
		ret = append(ret, "tsonly")
	}
	if tsandaddr {
		ret = append(ret, "tsandaddr")
	}
	if icmptimestamp {
		ret = append(ret, "icmptimestamp")
	}
	if icmpsum {
		ret = append(ret, "icmpsum")
	}
	if dl {
		ret = append(ret, "dl")
	}
	if eight {
		ret = append(ret, "8")
	}
	return ret
}

// GetPingsMulti gets pings that match the given PingMeasurements
func (db *DB) GetPingsMulti(in []*dm.PingMeasurement) ([]*dm.Ping, error) {
	var ret []*dm.Ping
	for _, pm := range in {
		if pm.TimeStamp != "" {
			continue
		}
		var stale int64
		if pm.Staleness == 0 {
			stale = 60
		}
		var ps []*dm.Ping
		var err error
		if pm.RR {
			if pm.Spoof {
				spi, _ := util.IPStringToInt32(pm.SAddr)
				ps, err = db.getPingSrcDstStaleRR(spi, pm.Dst, pm.Src, time.Duration(stale)*time.Minute)
				if err != nil {
					return nil, err
				}
			} else {
				ps, err = db.getPingSrcDstStaleRR(pm.Src, pm.Dst, 0, time.Duration(stale)*time.Minute)
				if err != nil {
					return nil, err
				}
			}
		} else {
			if pm.Spoof {
				spi, _ := util.IPStringToInt32(pm.SAddr)
				ps, err = db.GetPingBySrcDstWithStaleness(spi, pm.Dst, pm.Src, time.Duration(stale)*time.Minute)
				if err != nil {
					return nil, err
				}
			} else {
				ps, err = db.GetPingBySrcDstWithStaleness(pm.Src, pm.Dst, 0, time.Duration(stale)*time.Minute)
				if err != nil {
					return nil, err
				}
			}
		}
		ret = append(ret, ps...)
	}
	return ret, nil
}

func getRR(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getRecordRoutes, pingRRTable, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var hops []uint32
	for rows.Next() {
		rrhop := new(rrHop)
		err := rows.Scan(&rrhop.ResponseID, &rrhop.Hop, &rrhop.IP)
		if err != nil {
			return err
		}

		hops = append(hops, rrhop.IP)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.RR = hops
	return nil
}
func getTS(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getTimeStamps, pingTSTable, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var tss []uint32
	for rows.Next() {
		timestamp := new(uint32)
		err := rows.Scan(timestamp)
		if err != nil {
			return err
		}
		tss = append(tss, *timestamp)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.Tsonly = tss
	return nil
}

func getStats(p *dm.Ping, stmt *sql.Stmt) error {
	row := stmt.QueryRow(p.Id)
	stats := &dm.PingStats{}
	err := row.Scan(&stats.Loss, &stats.Min, &stats.Max, &stats.Avg, &stats.Stddev)
	if err != nil {
		return err
	}
	p.Statistics = stats
	return nil
}

func getTSAndAddr(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getTimeStampsAndAddr, pingTSAddrTable, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var tss []*dm.TsAndAddr
	for rows.Next() {
		tsandaddr := new(dm.TsAndAddr)
		err := rows.Scan(&tsandaddr.Ip, &tsandaddr.Ts)
		if err != nil {
			return err
		}
		tss = append(tss, tsandaddr)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.Tsandaddr = tss
	return nil
}

type errorf func() error

func logError(f errorf) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

func getResponses(ping *dm.Ping, tx *sql.Tx, rr, ts, tsaddr bool) error {
	rspstmt, err := tx.Prepare(getPingResponses)
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		if err := rspstmt.Close(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := rspstmt.Query(ping.Id)
	if err != nil {
		return err
	}

	defer rows.Close()

	var responses []*dm.PingResponse
	var respIds []int64
	for rows.Next() {
		resp := new(dm.PingResponse)
		var rID, pID sql.NullInt64
		var tx, rx int64
		err := rows.Scan(&rID, &pID, &resp.From, &resp.Seq, &resp.ReplySize,
			&resp.ReplyTtl, &resp.Rtt, &resp.ProbeIpid, &resp.ReplyIpid,
			&resp.IcmpType, &resp.IcmpCode, &tx, &rx)
		if err != nil {
			return err
		}
		resp.Tx = &dm.Time{}
		resp.Tx.Sec = tx / 1000000000
		resp.Tx.Usec = (tx % 1000000000) / 1000
		resp.Rx = &dm.Time{}
		resp.Rx.Sec = rx / 1000000000
		resp.Rx.Usec = (rx % 1000000000) / 1000
		ping.Responses = append(ping.Responses, resp)
		respIds = append(respIds, rID.Int64)
		responses = append(responses, resp)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i, resp := range responses {
		switch {
		case rr:
			err = getRR(respIds[i], resp, tx)
		case ts:
			err = getTS(respIds[i], resp, tx)
		case tsaddr:
			err = getTSAndAddr(respIds[i], resp, tx)
		}
		if err != nil {
			return err
		}
	}
	ping.Responses = responses
	return nil
}

// GetPingBySrcDst gets pings with the given src/dst
func (db *DB) GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error) {
	// We only keep 24 hours worth of data in the db
	return db.GetPingBySrcDstWithStaleness(src, dst, 0, time.Hour*24)
}

func (db *DB) getPingSrcDstStaleRR(src, dst, sp uint32, s time.Duration) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingStalenessRR, src, dst, sp, int(s.Minutes()))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer rows.Close()
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmptimestamp, &icmpsum,
			&dl, &eight)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}
func hasFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if flag == f {
			return true
		}
	}
	return false
}

// GetPingBySrcDstWithStaleness gets a ping with the src/dst that is newer than s
func (db *DB) GetPingBySrcDstWithStaleness(src, dst, sp uint32, s time.Duration) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingStaleness, src, dst, sp, int(s.Minutes()))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer rows.Close()
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmptimestamp, &icmpsum,
			&dl, &eight)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}

func (db *DB) GetMaxRevtrMeasurementID(measurementType string) (uint64, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		log.Error(err)
		return 0, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	var measurementTable = ""
	if measurementType == "pings" {
		measurementTable = pingTable
	} else if measurementType == "traceroutes" {
		measurementTable = tracerouteTable
	}
	query := fmt.Sprintf(getMaxRevtrMeasurementID, measurementTable)
	rows, err := tx.Query(query)
	if err != nil {
		log.Error(err)
		log.Error(query)
		return 0, err
	}
	defer rows.Close()
	var max sql.NullInt64
	for rows.Next() {
		err := rows.Scan(&max)
		if err != nil {
			log.Error(err)
			return 0, err
		} 
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return 0, err
	}
	if !max.Valid {
		return 0, nil 
	}
	return uint64(max.Int64), nil
}

var (
	insertPingBulk string 
	insertPing string 
	insertPingResponseBulk string 
	insertPingResponse string 
	insertRRBulk string 
	insertRR string 
	insertTsBulk string 
	insertTS string 
	insertTSADDRBulk string 
	insertTSADDR string 
	insertICMPTSBulk string 
	insertPingStatsBulk string 
	insertPingStats string 
)

func initPingWriteQueries () {

	insertPingBulk = fmt.Sprintf("INSERT INTO" +
	" %s(src, dst, start, ping_sent, probe_size," +
	" user_id, ttl, wait, spoofed_from, version," +
	" spoofed, record_route, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, `8`, revtr_measurement_id, label)" +
	" VALUES ", pingTable)
	insertPing = fmt.Sprintf("INSERT INTO" +
		" %s(src, dst, start, ping_sent, probe_size," +
		" user_id, ttl, wait, spoofed_from, version," +
		" spoofed, record_route, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, `8`, revtr_measurement_id, label)" +
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", pingTable)

	insertPingResponseBulk = fmt.Sprintf("INSERT INTO " +
	"%s(ping_id, `from`, seq, reply_size, reply_ttl, " +
	"			   reply_proto, rtt, probe_ipid, reply_ipid, " +
	"icmp_type, icmp_code, tx, rx) " +
	"VALUES ", pingResponseTable) 

	insertPingResponse = fmt.Sprintf("INSERT INTO " +
		"%s(ping_id, `from`, seq, reply_size, reply_ttl, " +
		"			   reply_proto, rtt, probe_ipid, reply_ipid, " +
		"icmp_type, icmp_code, tx, rx) " +
		"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ", pingResponseTable)

	insertRRBulk = fmt.Sprintf(`
	INSERT INTO
	%s(response_id, hop, ip)
	VALUES 
	`, pingRRTable)

	insertRR = fmt.Sprintf(`
	INSERT INTO
	%s(response_id, hop, ip)
	VALUES(?, ?, ?)`, pingRRTable)	

	insertTsBulk = fmt.Sprintf("INSERT INTO %s(response_id, `order`, ts) VALUES ", pingTSTable)
	insertTS = fmt.Sprintf("INSERT INTO %s(response_id, `order`, ts) VALUES(?, ?, ?)", pingTSTable)

	insertTSADDRBulk = fmt.Sprintf("INSERT INTO %s(response_id, `order`, ip, ts) VALUES ", pingTSAddrTable)
	insertTSADDR = fmt.Sprintf("INSERT INTO %s(response_id, `order`, ip, ts) VALUES(?, ?, ?, ?)", pingTSAddrTable)

	insertICMPTSBulk = fmt.Sprintf("INSERT INTO %s(response_id, origin_ts, receive_ts, transmit_ts) VALUES ", pingICMPTSTable)


	insertPingStatsBulk = fmt.Sprintf(`
	INSERT INTO
	%s(ping_id, loss, min, max, avg, std_dev)
	VALUES 
	`, pingStatsTable)
	insertPingStats = fmt.Sprintf(`
	INSERT INTO
	%s(ping_id, loss, min, max, avg, std_dev)
	VALUES(?, ?, ?, ?, ?, ?)
	`, pingStatsTable)
}

func GetIdsFromBulkInsert(bulkSize int64, res sql.Result) ([] int64, error){
	insertedIds := []int64{}
	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	for i := lastInsertId; i < lastInsertId + bulkSize; i++ {
		insertedIds = append(insertedIds, i)	
	} 
	return insertedIds, nil
}

func storePingBulk(tx *sql.Tx, ins []*dm.Ping) ([]int64, error) {

	query := bytes.Buffer{}
	query.WriteString(insertPingBulk)
	for _, in := range ins {
		// if i == 2 {
		// 	break
		// }
		var start time.Time
		if in.Start == nil {
			start = time.Now()
		} else {
			start = time.Unix(in.Start.Sec, in.Start.Usec*1000)
		}
		flags := make(map[string]byte)
		for _, flag := range in.Flags {
			flags[flag] = 1
		}
		
		query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, %d, %d, %d, %d, \"%s\", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, \"%s\"),", 
		in.Src, in.Dst, start.UnixNano(), in.PingSent, in.ProbeSize,
		in.UserId, in.Ttl, in.Wait, in.SpoofedFrom, in.Version,
		flags["spoof"], flags["v4rr"], flags["payload"], flags["tsonly"],
		flags["tsandaddr"], flags["icmptimestamp"],flags["icmpsum"], flags["dl"], flags["8"], in.RevtrMeasurementId, in.Label))
	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	res, err := tx.Exec(queryStr)

	if err != nil {
		return nil, err
	}
	insertedIds, err := GetIdsFromBulkInsert(int64(len(ins)), res)
	if err != nil {
		return nil, err
	} 
	return insertedIds, nil
}

func storePing(tx *sql.Tx, in *dm.Ping) (int64, error) {
	/*
		These are the posible keys for the map

		"v4rr"
		"spoof"
		"payload"
		"tsonly"
		"tsandaddr"
		"icmpsum"
		"dl"
		"8"
	*/
	var start time.Time
	if in.Start == nil {
		start = time.Now()
	} else {
		start = time.Unix(in.Start.Sec, in.Start.Usec*1000)
	}
	flags := make(map[string]byte)
	for _, flag := range in.Flags {
		flags[flag] = 1
	}
	res, err := tx.Exec(insertPing, in.Src, in.Dst, start.UnixNano(), in.PingSent, in.ProbeSize,
		in.UserId, in.Ttl, in.Wait, in.SpoofedFrom, in.Version,
		flags["spoof"], flags["v4rr"], flags["payload"], flags["tsonly"],
		flags["tsandaddr"], flags["icmptimestamp"], flags["icmpsum"], flags["dl"], flags["8"], in.RevtrMeasurementId, in.Label)

	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func storePingStatsBulk(tx *sql.Tx, ids [] int64, pings[] *dm.Ping) error {
	query := bytes.Buffer{}
	query.WriteString(insertPingStatsBulk)
	statsCount := 0
	for i, p := range pings {
		stat := p.GetStatistics()
		if stat == nil{
			continue
		} else {
			statsCount += 1
		}
		query.WriteString(fmt.Sprintf("(%d, %f, %f, %f, %f, %f),", 
		ids[i], stat.Loss, stat.Min, stat.Max, stat.Avg, stat.Stddev,
		))

	} 
	if statsCount == 0 {
		return nil
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	return err
}

func storePingStats(tx *sql.Tx, id int64, stat *dm.PingStats) error {
	if stat == nil {
		return nil
	}
	_, err := tx.Exec(insertPingStats, id, stat.Loss, stat.Min, stat.Max, stat.Avg, stat.Stddev)
	return err
}

func storePingRRBulk(tx *sql.Tx, rrHops [] *rrHop ) error {
	query := bytes.Buffer{}
	query.WriteString(insertRRBulk)

	if len(rrHops) == 0{
		return nil
	}
	for _, rr := range rrHops {
		query.WriteString(fmt.Sprintf("(%d, %d, %d),", 
		rr.ResponseID.Int64, rr.Hop, rr.IP)	)
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	if err != nil {
		return err
	}
	return err
}

func storePingRR(tx *sql.Tx, id int64, rr uint32, hop int8) error {
	_, err := tx.Exec(insertRR, id, hop, rr)
	return err
}

func storeTsBulk(tx *sql.Tx, tsResponses [] * ts) error {
	query := bytes.Buffer{}
	query.WriteString(insertTsBulk)


	if len(tsResponses) == 0{
		return nil
	}
	for _, t := range tsResponses {
		query.WriteString(fmt.Sprintf("(%d, %d, %d),", 
		t.ResponseID.Int64, t.Order, t.Ts))
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	if err != nil {
		return err
	}
	return err
}

func storeTS(tx *sql.Tx, id int64, order int8, ts uint32) error {
	_, err := tx.Exec(insertTS, id, order, ts)
	return err
}

func storeTSAndAddrBulk(tx *sql.Tx, tsAndAddrResponses [] * tsAndAddr) error {
	query := bytes.Buffer{}
	query.WriteString(insertTSADDRBulk)

	if len(tsAndAddrResponses) == 0 {
		return nil
	}

	for _, t := range tsAndAddrResponses {
		query.WriteString(fmt.Sprintf("(%d, %d, %d, %d),", 
		t.ResponseID.Int64, t.Order, t.IP, t.Ts))	
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	if err != nil {
		return err
	}
	return err
}
func storeTSAndAddr(tx *sql.Tx, id int64, order int8, ts *dm.TsAndAddr) error {
	_, err := tx.Exec(insertTSADDR, id, order, ts.Ip, ts.Ts)
	return err
}

func storeICMPTSBulk(tx *sql.Tx, icmpTimestampResponses [] * icmpTimestamp) error {
	query := bytes.Buffer{}
	query.WriteString(insertICMPTSBulk)

	if len(icmpTimestampResponses) == 0 {
		return nil
	}

	for _, t := range icmpTimestampResponses {
		query.WriteString(fmt.Sprintf("(%d, %d, %d, %d),", 
		t.ResponseID.Int64, t.OTs, t.RTs, t.TTs))	
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err := tx.Exec(queryStr)
	if err != nil {
		return err
	}
	return err
}


func storePingResponseBulk(trx *sql.Tx, ids []int64, rs []*dm.Ping) error {

	query := bytes.Buffer {}
	query.WriteString(insertPingResponseBulk)
	rrResponses := [] *rrHop {}
	tsResponses := [] * ts {}
	tsAndAddrResponses := [] *tsAndAddr {}
	icmpTSResponses := [] *icmpTimestamp{}
	responseCount := 0
	for i, ping := range rs {
		ping_responses := ping.GetResponses()
		if len(ping_responses) == 0{
			continue
		}

		flags := make(map[string]byte)
		for _, flag := range ping.Flags {
			flags[flag] = 1
		}

		for _, r := range ping_responses{
			// if i == 2 {
			// 	break
			// }
			tx := r.Tx.Sec*1000000000 + r.Tx.Usec*1000
			rx := r.Rx.Sec*1000000000 + r.Rx.Usec*1000
			
			query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, \"%s\", %d, %d, %d, %d, %d, %d, %d),", 
			ids[i], r.From, r.Seq,
			r.ReplySize, r.ReplyTtl, r.ReplyProto,
			r.Rtt, r.ProbeIpid, r.ReplyIpid,
			r.IcmpType, r.IcmpCode, tx, rx))
			
			if _, ok := flags["icmptimestamp"] ; ok{
				icmpTSResponses = append(icmpTSResponses, &icmpTimestamp{
					ResponseID : sql.NullInt64{
						Int64: int64(responseCount),
					},
					OTs: r.IcmpTs.OriginateTimestamp,
					RTs: r.IcmpTs.ReceiveTimestamp,
					TTs: r.IcmpTs.TransmitTimestamp,
				})
			}
			
			for hop, ip := range r.RR{
				rrResponses = append(rrResponses, &rrHop{
					ResponseID : sql.NullInt64{
						Int64: int64(responseCount),
					}, 
					Hop: uint8(hop),
					IP: ip,
				})
			}
			for order, timestamp := range r.Tsonly{
				tsResponses = append(tsResponses, &ts{
					ResponseID : sql.NullInt64{
						Int64: int64(responseCount),
					}, 
					Order: uint8(order),
					Ts: timestamp,
				})
			}
			for order, tsAA := range r.Tsandaddr{
				tsAndAddrResponses = append(tsAndAddrResponses, &tsAndAddr{
					ResponseID : sql.NullInt64{
						Int64: int64(responseCount),
					}, 
					Order: uint8(order),
					Ts: tsAA.Ts,
					IP: tsAA.Ip,
				})
			}
			responseCount += 1
		}	
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	if responseCount == 0{
		// No response at all for all the pings in the batch, so just return.
		return nil
	}
	res, err := trx.Exec(queryStr)
	if err != nil {
		return err
	}
	insertedIds, err := GetIdsFromBulkInsert(int64(responseCount), res)
	if err != nil {
		return err
	}

	// Change the IDs to make them correspond to response IDs
	for _, rr := range rrResponses {
		rr.ResponseID.Int64 = insertedIds[rr.ResponseID.Int64]
	}
	err = storePingRRBulk(trx, rrResponses)
	if err != nil {
		return err
	}

	// Change the IDs to make them correspond to response IDs
	for _, t := range tsResponses {
		t.ResponseID.Int64 = insertedIds[t.ResponseID.Int64]
	}
	err = storeTsBulk(trx, tsResponses)
	if err != nil {
		return err
	}


	// Change the IDs to make them correspond to response IDs
	for _, t := range tsAndAddrResponses {
		t.ResponseID.Int64 = insertedIds[t.ResponseID.Int64]
	}
	err = storeTSAndAddrBulk(trx, tsAndAddrResponses)
	if err != nil {
		return err
	}

	for _, t := range icmpTSResponses {
		t.ResponseID.Int64 = insertedIds[t.ResponseID.Int64]
	}

	err = storeICMPTSBulk(trx, icmpTSResponses)
	if err != nil {
		return err
	}


	return nil
}

func storePingResponse(trx *sql.Tx, id int64, r *dm.PingResponse) error {
	if id == 0 || r == nil {
		return fmt.Errorf("Invalid parameter: storePingResponse")
	}
	tx := r.Tx.Sec*1000000000 + r.Tx.Usec*1000
	rx := r.Rx.Sec*1000000000 + r.Rx.Usec*1000
	res, err := trx.Exec(insertPingResponse, id, r.From, r.Seq,
		r.ReplySize, r.ReplyTtl, r.ReplyProto,
		r.Rtt, r.ProbeIpid, r.ReplyIpid,
		r.IcmpType, r.IcmpCode, tx, rx)
	if err != nil {
		return err
	}
	nid, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for i, rr := range r.RR {
		err := storePingRR(trx, nid, rr, int8(i))
		if err != nil {
			return err
		}
	}
	for i, ts := range r.Tsonly {
		err := storeTS(trx, nid, int8(i), ts)
		if err != nil {
			return err
		}
	}
	for i, ts := range r.Tsandaddr {
		err := storeTSAndAddr(trx, nid, int8(i), ts)
		if err != nil {
			return err
		}
	}
	return nil
}


func (db *DB) StorePingBulk(ins []*dm.Ping) ([]int64, error)  {
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return nil, err
	}
	ids, err := storePingBulk(tx, ins)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = storePingResponseBulk(tx, ids, ins)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = storePingStatsBulk(tx, ids, ins)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return ids, tx.Commit() 
} 

// StorePing saves a ping to the DB
func (db *DB) StorePing(in *dm.Ping) (int64, error) {
	log.Debug("Storing PING", in.Label, ",", in.Src, ",", in.Dst)
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	id, err := storePing(tx, in)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	for _, pr := range in.GetResponses() {
		err = storePingResponse(tx, id, pr)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	err = storePingStats(tx, id, in.GetStatistics())
	return id, tx.Commit()
}

const (
	insertAdjQuery = `INSERT INTO adjacencies(ip1, ip2) VALUES (?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt+1`
)

// StoreAdjacency stores an adjacency
func (db *DB) StoreAdjacency(l, r net.IP) error {
	con := db.GetWriter()
	ip1, err := util.IPtoInt32(l)
	if err != nil {
		return err
	}
	ip2, err := util.IPtoInt32(r)
	if err != nil {
		return err
	}
	_, err = con.Exec(insertAdjQuery, ip1, ip2)
	if err != nil {
		return err
	}
	return nil
}

const (
	insertAdjDstQuery = `
	INSERT INTO adjacencies_to_dest(dest24, address, adjacent) VALUES(?, ?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt + 1
	`
)

// StoreAdjacencyToDest stores an adjacencies to dest
func (db *DB) StoreAdjacencyToDest(dest24, addr, adj net.IP) error {
	con := db.GetWriter()
	destip, _ := util.IPtoInt32(dest24)
	destip = destip >> 8
	addrip, _ := util.IPtoInt32(addr)
	adjip, _ := util.IPtoInt32(adj)
	_, err := con.Exec(insertAdjDstQuery, destip, addrip, adjip)
	if err != nil {
		return err
	}
	return nil
}

const (
	removeAliasCluster = `DELETE FROM ip_aliases WHERE cluster_id = ?`
	storeAlias         = `INSERT INTO ip_aliases(cluster_id, ip_address) VALUES(?, ?)`
	storeAliasBulk     = `INSERT INTO ip_aliases(cluster_id, ip_address) VALUES `
)

// StoreAlias stores an IP alias
func (db *DB) StoreAlias(id int, ips []net.IP) error {
	con := db.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	// _, err = tx.Exec(removeAliasCluster, id)
	// if err != nil {
	// 	tx.Rollback()
	// 	return err
	// }

	query := bytes.Buffer{}
	query.WriteString(storeAliasBulk)

	for _, ip := range ips {
		ipint, _ := util.IPtoInt32(ip)
		query.WriteString(fmt.Sprintf("(%d, %d),", 
		id, ipint))
		// _, err = tx.Exec(storeAlias, id, ipint)
		// if err != nil {
		// 	tx.Rollback()
		// 	return err
		// }
	}

	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err = tx.Exec(queryStr)
	if err != nil {
		log.Error(err)
		return tx.Rollback()
	}

	
	return tx.Commit()
}

var (
	getUser string 
	addPingBatch string 
	addPingBatchPingBulk string 
	addPingBatchPing string 
	getPingBatch string 
	getPingBatchNResponses string 
	getPingByLabel string 
)

func initPingBatchQueries() {

	getUser          = "select * from users where `key` = ?;"
	addPingBatch     = `insert into ping_batch(user_id) VALUES(?)`
	addPingBatchPingBulk = `insert into ping_batch_ping(batch_id, ping_id) VALUES `
	addPingBatchPing = `insert into ping_batch_ping(batch_id, ping_id) VALUES(?, ?)`

	getPingBatch     = "SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, " +
		"p.tsandaddr, p.icmptimestamp, p.icmpsum, dl, p.`8` FROM " +
		"users u " +
		"inner join ping_batch pb on pb.user_id = u.id " +
		"inner join ping_batch_ping pbp on pb.id = pbp.batch_id " +
		"inner join pings p on p.id = pbp.ping_id " +
		"Where u.`key` = ? and pb.id = ?;"
	getPingBatchNResponses = "SELECT count(*) FROM " +
	"users u " +
	"inner join ping_batch pb on pb.user_id = u.id " +
	"inner join ping_batch_ping pbp on pb.id = pbp.batch_id " +
	"inner join ping_stats pstats on pstats.ping_id = pbp.ping_id " +
	"Where u.`key` = ? and pb.id = ?;"

	getPingByLabel = fmt.Sprintf(" select  p.id, p.src, p.dst, p.start, p.ping_sent, " +
	"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
	"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, " +
	"p.tsandaddr, p.icmptimestamp, p.icmpsum, dl, p.`8` " +
	// // Response fields
	// "pr.id, pr.ping_id, pr.`from`, pr.seq, " +
	// "pr.reply_size, pr.reply_ttl, pr.rtt, pr.probe_ipid, pr.reply_ipid, " +
	// "pr.icmp_type, pr.icmp_code, pr.tx, pr.rx, " +
	// // Stats fields 
	// "ps.loss, ps.min, " +
	// "ps.max, ps.avg, ps.std_dev " +
	// // Record route fields 
	// "rr.response_id, rr.hop, rr.ip, " +
	// // Timestamp fields
	// "ts.ts," +
	// // Timestamp address
	// "tsa.ip, tsa.ts " +

	"from %s as p " + 
	// "INNER JOIN ping_stats ps on ps.ping_id=p.id " +
	// "INNER JOIN ping_responses as pr ON pr.ping_id=p.id " +
	// "INNER JOIN record_routes rr ON rr.response_id = pr.id  " +
	// "INNER JOIN timestamps ts on ts.response_id=pr.id " +
	// "INNER JOIN timestamp_addrs tsa ON tsa.response_id=pr.id  " +
	"WHERE label= ? ;", pingTable) 
}

// AddPingBatch adds a batch of pings
func (db *DB) AddPingBatch(u dm.User) (int64, error) {
	con := db.GetWriter()
	res, err := con.Exec(addPingBatch, u.ID)
	if err != nil {
		return 0, err
	}
	bid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return bid, err
}

// AddPingsToBatch adds pings pids to batch bid
func (db *DB) AddPingsToBatch(bid int64, pids []int64) error {
	con := db.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}

	if len(pids) == 0{
		return nil
	}
		

	query := bytes.Buffer{}
	query.WriteString(addPingBatchPingBulk)

	for _, pid := range pids {
		query.WriteString(fmt.Sprintf("(%d, %d),", 
		bid, pid))
	}

	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err = tx.Exec(queryStr)
	if err != nil {
		log.Error(err)
		return tx.Rollback()
	}
	return tx.Commit()
}

// GetUser get a user with the given key
func (db *DB) GetUser(key string) (dm.User, error) {
	con := db.GetReader()
	row := con.QueryRow(getUser, key)
	var user dm.User
	err := row.Scan(&user.ID, &user.Name, &user.EMail, &user.Max, &user.Delay, &user.Key)
	if err != nil {
		return dm.User{}, err
	}
	return user, nil
}

// GetPingBatch gets a batch of pings for user u with id bid
func (db *DB) GetPingBatch(u dm.User, bid int64) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingBatch, u.Key, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmptimestamp, &icmpsum,
			&dl, &eight)
		if err != nil {
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}


func (db *DB) GetPingBatchNResponses(u dm.User, bid int64) (uint32, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingBatchNResponses, u.Key, bid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := uint32(0)
	rows.Next()
	err = rows.Scan(&count)
	
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}


// GetPingBatch gets a batch of pings for user u with id bid
func (db *DB) GetPingByLabel(label string) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingByLabel, label)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmptimestamp, &icmpsum,
			&dl, &eight,
			
		)
		if err != nil {
			
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmptimestamp, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}