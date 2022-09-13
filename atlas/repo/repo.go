package repo

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/types"
	"github.com/NEU-SNS/ReverseTraceroute/environment"

	dasql "github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/jmoiron/sqlx" // For IN placeholder
)

// Repo is a respository for storing and querying traceroutes
type Repo struct {
	Repo *repository.DB
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
	return &Repo{Repo: db}, nil
}

type errorf func() error

func logError(e errorf) {
	if err := e(); err != nil {
		log.Error(err)
	}
}

func (r *Repo) MonitorDB() {
	t := time.NewTicker(5 * time.Second)
	
	for {
		select{
		case <-t.C:
			dbReaderStats := r.Repo.GetReader().Stats()
			dbWriterStats  := r.Repo.GetWriter().Stats()
			if dbReaderStats.InUse > 0 {
				log.Infof("Number of in use / open /max  connections reader: %d %d %d", 
				dbReaderStats.InUse, dbReaderStats.OpenConnections, dbReaderStats.MaxOpenConnections)
			} 
			if dbWriterStats.InUse > 0 {
				log.Infof("Number of in use / open /max  connections writer: %d %d %d", 
				dbWriterStats.InUse, dbWriterStats.OpenConnections, dbWriterStats.MaxOpenConnections)
			} 
		}
	}
}


const (
	layout = "2006-01-02T15:04:05-04:00"
)

const (
	selectRRIntersectionByIDIntersection = `
	SELECT tr_ttl_start, tr_ttl_end
	FROM atlas_rr_intersection ath
	WHERE traceroute_id = ? and rr_hop = ? 
	LIMIT 1
`

	selectTracerouteByID = `
	SELECT at.src, at.dest, hop, ttl, at.platform, at.date
	FROM atlas_traceroute_hops ath
	INNER JOIN atlas_traceroutes at ON at.id = ath.trace_id
	WHERE trace_id = ?
	ORDER BY ttl
	`

	findIntersectingRR = `
	SELECT traceroute_id, rr_hop, tr_ttl_start, tr_ttl_end, at.src, at.platform
	FROM atlas_rr_intersection arri
	INNER JOIN  atlas_traceroutes at on at.id=arri.traceroute_id
	WHERE at.stale != TRUE AND rr_hop = ? AND dest = ? AND date >= DATE_SUB(?, interval ?  minute)
	AND platform in (?)
	ORDER BY  at.date desc, at.platform, at.id desc
	LIMIT 1
`	
	findIntersectingRRIgnoreSource = `
	SELECT traceroute_id, rr_hop, tr_ttl_start, tr_ttl_end, at.src, at.platform
	FROM atlas_rr_intersection arri
	INNER JOIN  atlas_traceroutes at on at.id=arri.traceroute_id
	WHERE at.stale != TRUE AND rr_hop = ? AND src != ? AND dest = ? AND date >= DATE_SUB(?, interval ?  minute)
	AND platform in (?)
	ORDER BY  at.date desc, at.platform, at.id desc
	LIMIT 1
`	
	findIntersectingRRIgnoreSourceAS = `
	SELECT traceroute_id, rr_hop, tr_ttl_start, tr_ttl_end, at.src, at.platform
	FROM atlas_rr_intersection arri
	INNER JOIN  atlas_traceroutes at on at.id=arri.traceroute_id
	WHERE at.stale != TRUE AND rr_hop = ? AND source_asn != ? AND dest = ? AND date >= DATE_SUB(?, interval ?  minute)
	AND platform in (?)
	ORDER BY  at.date desc, at.platform, at.id desc
	LIMIT 1
`	
	findIntersecting = `
SELECT 
	? as src, A.Id, A.dest, A.IP, hops.hop, hops.ttl, A.src, A.platform, A.date
FROM 
(
SELECT
	*
FROM
(
(SELECT atr.Id, atr.date, atr.src, atr.dest, atr.platform FROM
atlas_traceroutes atr 
WHERE atr.stale != TRUE AND atr.dest = ? AND atr.platform in (?)  AND atr.date >= DATE_SUB(?, interval ?  minute)

ORDER BY atr.date desc, atr.platform, atr.id desc)
) X 
INNER JOIN atlas_traceroute_hops ath on ath.trace_id = X.Id
INNER JOIN
(
SELECT ? IP
UNION
SELECT
		b.ip_address 
	FROM
		ip_aliases a INNER JOIN ip_aliases b on a.cluster_id = b.cluster_id
	WHERE
		a.ip_address = ?	
) Z ON ath.hop = Z.IP
ORDER BY date desc
limit 1
) A
INNER JOIN atlas_traceroute_hops hops on hops.trace_id = A.Id
ORDER BY hops.ttl
`
	findIntersectingIgnoreSource = `
SELECT 
	? as src, A.Id, A.dest, A.IP, hops.hop, hops.ttl, A.src, A.platform, A.date
FROM 
(
SELECT
	*
FROM
(
(SELECT atr.Id, atr.date, atr.src, atr.dest, atr.platform FROM
atlas_traceroutes atr 
WHERE atr.stale != TRUE AND atr.src != ? AND atr.dest = ? AND atr.platform in (?) AND atr.date >= DATE_SUB(?, interval ?  minute) 
ORDER BY atr.date desc, atr.platform, atr.id desc)
) X 
INNER JOIN atlas_traceroute_hops ath on ath.trace_id = X.Id
INNER JOIN
(
SELECT ? IP
UNION
SELECT
		b.ip_address
	FROM
		ip_aliases a INNER JOIN ip_aliases b on a.cluster_id = b.cluster_id
	WHERE
		a.ip_address = ?	
) Z ON ath.hop = Z.IP
ORDER BY date desc
limit 1
) A
INNER JOIN atlas_traceroute_hops hops on hops.trace_id = A.Id
ORDER BY hops.ttl
`
findIntersectingIgnoreSourceAS = `
SELECT 
	? as src, A.Id, A.dest, A.IP, hops.hop, hops.ttl, A.src, A.platform, A.date
FROM 
(
SELECT
	*
FROM
(
(SELECT atr.Id, atr.date, atr.src, atr.dest, atr.platform FROM
atlas_traceroutes atr 
WHERE  atr.stale != TRUE AND atr.src != ? AND atr.dest = ? AND atr.platform in (?) AND atr.date >= DATE_SUB(?, interval ?  minute) AND atr.source_asn != ?
ORDER BY atr.date desc, atr.platform, atr.id desc)
) X 
INNER JOIN atlas_traceroute_hops ath on ath.trace_id = X.Id
INNER JOIN
(
SELECT ? IP
UNION
SELECT
		b.ip_address
	FROM
		ip_aliases a INNER JOIN ip_aliases b on a.cluster_id = b.cluster_id
	WHERE
		a.ip_address = ?	
) Z ON ath.hop = Z.IP
ORDER BY date desc
limit 1
) A
INNER JOIN atlas_traceroute_hops hops on hops.trace_id = A.Id
ORDER BY hops.ttl
`

	getSources = `
SELECT
    src
FROM
    atlas_traceroutes
WHERE
    dest = ? AND date >= DATE_SUB(NOW(), interval ? minute) 
GROUP BY
    src;
`
)

type hopRow struct {
	src  uint32
	dest uint32
	hop  uint32
	ttl  uint32
}

var (
	// ErrNoIntFound is returned when no intersection is found
	ErrNoIntFound = fmt.Errorf("No Intersection Found")
)

// GetAtlasSources gets all sources that were used for existing atlas traceroutes
// the set of vps - this would be the sources to use to run traces
func (r *Repo) GetAtlasSources(dst uint32, stale time.Duration) ([]uint32, error) {
	rows, err := r.Repo.GetReader().Query(getSources, dst, int64(stale.Minutes()))
	var srcs []uint32
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer logError(rows.Close)
	for rows.Next() {
		var curr uint32
		err := rows.Scan(&curr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		srcs = append(srcs, curr)
	}
	if err = rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	return srcs, nil
}

// CheckIntersectingPath retrieves the traceroute that has been chosen
func (r *Repo) CheckIntersectingPath(tracerouteID int64, hopIntersection uint32) (*pb.Path, error) {
	// First check if the hop intersection is in the traceroute
	// If it is, it was an exact match if we do not need to look further
	// Else it was a guessed RR match so we need to look in the RR table
	var err error

	aliases := map[uint32]struct{}{}
	aliases[hopIntersection] = struct{}{}
	// First get the aliases of the intersection
	rowsAlias, err := r.Repo.GetReader().Query(getAlias, hopIntersection)
	defer logError(rowsAlias.Close)
	for rowsAlias.Next() {
		var alias uint32
		err = rowsAlias.Scan(&alias)
		aliases[alias] = struct{}{}
	}
	// Then get the corresponding traceroute
	rows, err := r.Repo.GetReader().Query(selectTracerouteByID, tracerouteID)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer logError(rows.Close)
	ret := pb.Path{}
	candidateHops := [] *pb.Hop {}
	isAfterIntersection := false
	var platform string 
	var date string
	for rows.Next() {
		row := hopRow{}
		err = rows.Scan(&row.src, &row.dest, &row.hop, &row.ttl, &platform, &date)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		hop := &pb.Hop{
			Ip:  row.hop,
			Ttl: row.ttl,
			IntersectHopType: pb.IntersectHopType_EXACT,
		}
		candidateHops = append(candidateHops, hop)
		if _, ok := aliases[row.hop] ; ok {
			isAfterIntersection = true
		}
		if isAfterIntersection {
			ret.Hops = append(ret.Hops, hop)
		}
	}
	if len(ret.Hops) > 0 {
		return &ret, nil
	}
	// Otherwise it was a RR intersection, so find the corresponding hops
	rowsRR, err := r.Repo.GetReader().Query(selectRRIntersectionByIDIntersection, tracerouteID, hopIntersection)
	defer logError(rowsRR.Close)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for rowsRR.Next() {
		trTTLStart := uint32(0)
		trTTLEnd := uint32(0)
		err = rowsRR.Scan(&trTTLStart, &trTTLEnd)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		for _, hop := range(candidateHops) {
			if hop.Ttl < trTTLStart {
				continue
			} else if trTTLStart <= hop.Ttl && hop.Ttl <= trTTLEnd {
				ret.Hops = append(ret.Hops, &pb.Hop{
					Ip:  hop.Ip,
					Ttl: hop.Ttl,
					IntersectHopType: pb.IntersectHopType_BETWEEN,
				})
			} else {
				ret.Hops = append(ret.Hops, &pb.Hop{
					Ip:  hop.Ip,
					Ttl: hop.Ttl,
					IntersectHopType: pb.IntersectHopType_EXACT,
				})
			}
		}
		return &ret, nil
	}
	// Should never get here 
	return nil, err
	
}

// FindIntersectingTraceroute finds a traceroute that intersects hop towards the dst
func (r *Repo) FindIntersectingTraceroute(iq types.IntersectionQuery) (types.IntersectionResponse, error) {
	log.Debug("Finding intersecting traceroute ", iq)
	// var err error
	foundIntersectingRR := false
	if iq.UseAtlasRR {
		// Allow to use the RR pings from the traceroute atlas.
		var  rowsRR *sql.Rows
		if iq.IgnoreSourceAS {
			q,args,err := sqlx.In(findIntersectingRRIgnoreSourceAS, iq.Addr, iq.SourceAS, time.Now().Format(environment.DateFormat), iq.Dst, iq.Stale, iq.Platforms)
			if err != nil  {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rowsRR, err = r.Repo.GetReader().Query(q, args...)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		} else if iq.IgnoreSource {
			q, args, err := sqlx.In(findIntersectingRRIgnoreSource, iq.Addr, iq.Src, iq.Dst, time.Now().Format(environment.DateFormat), iq.Stale, iq.Platforms)
			// log.Infof(q)
			if err != nil  {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rowsRR, err = r.Repo.GetReader().Query(q, args...)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		} else {
			q, args, err := sqlx.In(findIntersectingRR, iq.Addr, iq.Dst, time.Now().Format(environment.DateFormat),iq.Stale, iq.Platforms)
			if err != nil  {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rowsRR, err = r.Repo.GetReader().Query(q, args...)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		}
		tracerouteID := int64(0)
		rrHop := 0
		trHopStart := uint32(0)
		trHopEnd := uint32(0)
		src := uint32(0)
		platform := ""
		intersectionTTL := uint32(0)
		distanceToSource := uint32(0)
		defer rowsRR.Close()
		for rowsRR.Next() {
			err := rowsRR.Scan(&tracerouteID, &rrHop, &trHopStart, &trHopEnd, &src, &platform)
			if err != nil {
				return types.IntersectionResponse{}, err
			}
			foundIntersectingRR = true
			intersectionTTL = (trHopStart + trHopEnd) / 2 // Take the mean of the intersection
			break // There should be only one line.
		}
		if foundIntersectingRR {
			// Get the corresponding traceroute 
			rowsTr, err := r.Repo.GetReader().Query(selectTracerouteByID, tracerouteID)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			defer logError(rowsTr.Close)
			ret := pb.Path{}
			src := uint32(0)
			platform := ""
			var date string
			for rowsTr.Next() {
				row := hopRow{}
				err := rowsTr.Scan(&row.src, &row.dest, &row.hop, &row.ttl, &platform, &date)
				if err != nil {
					return types.IntersectionResponse{}, err
				}
				src = row.src
				var ht pb.IntersectHopType
				if row.ttl < trHopStart {
					// It's a hop before the intersection, so do not return it
					continue
				} 
				
				if trHopStart != trHopEnd && trHopStart <= row.ttl && row.ttl <= trHopEnd {
					ht = pb.IntersectHopType_BETWEEN
				} else {
					ht = pb.IntersectHopType_EXACT
				}

				if len(ret.Hops) == 0 {
					// Add the RR IP addresses (that is not in the traceroute but need it for further intersection, will
					// mark the between intersection later on.)
					ret.Hops = append(ret.Hops, &pb.Hop{
						Ip:  iq.Addr,
						Ttl: row.ttl,
						IntersectHopType: ht,
					})	
				}
				ret.Hops = append(ret.Hops, &pb.Hop{
					Ip:  row.hop,
					Ttl: row.ttl,
					IntersectHopType: ht,
				})
				ret.Address = iq.Addr
				if row.hop == row.dest {
					distanceToSource = row.ttl - intersectionTTL	
				}
			}
			if err := rowsTr.Err(); err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			if len(ret.Hops) == 0 {
				return types.IntersectionResponse{}, ErrNoIntFound
			}
			trTime, err := time.Parse(time.RFC3339, date)
			trTimeUnix := trTime.Unix()
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			return types.IntersectionResponse{
				TracerouteID: tracerouteID,
				Path : &ret, 
				Src: src,
				Platform: platform,
				Timestamp: trTimeUnix,
				DistanceToSource: distanceToSource,
			}, nil
		}
	} 
	var rows *sql.Rows
	if !foundIntersectingRR{
		// Still try to find an intersecting traceroute as the intersecting hop might not be an RR hop
		if iq.IgnoreSourceAS {
			// This is the most constrained 
			q, args, err := sqlx.In(findIntersectingIgnoreSourceAS, iq.Addr, iq.Src, 
				iq.Dst, iq.Platforms, time.Now().Format(environment.DateFormat), iq.Stale,
				iq.SourceAS, iq.Addr, iq.Addr)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rows, err = r.Repo.GetReader().Query(q, args)

			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		} else if iq.IgnoreSource {
			q, args, err := sqlx.In(findIntersectingIgnoreSource, iq.Addr, iq.Src, iq.Dst, iq.Platforms, time.Now().Format(environment.DateFormat), iq.Stale, iq.Addr, iq.Addr)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rows, err = r.Repo.GetReader().Query(q, args...)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		} else {
			q, args, err := sqlx.In(findIntersecting, iq.Addr, iq.Dst, iq.Platforms, time.Now().Format(environment.DateFormat), iq.Stale, iq.Addr, iq.Addr)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
			rows, err = r.Repo.GetReader().Query(q, args...)
			if err != nil {
				log.Error(err)
				return types.IntersectionResponse{}, err
			}
		}
	}
	defer logError(rows.Close)
	ret := pb.Path{}
	isAfterIntersection := false
	intersection := uint32(0)
	tracerouteID := int64(0)
	src := uint32(0)
	platform := ""
	intersectionTTL := uint32(0)
	distanceToSource := uint32(0)
	var date string
	for rows.Next() {
		row := hopRow{}
		// row.src is the intersection IP in the request, intersection is either the intersection or an alias
		err := rows.Scan(&row.src, &tracerouteID, &row.dest, &intersection, &row.hop, &row.ttl, &src,
		&platform, &date)
		if err != nil {
			return types.IntersectionResponse{}, err
		}
		if row.hop == intersection && intersection != 0 {
			isAfterIntersection = true
			intersectionTTL = row.ttl
		}
		if isAfterIntersection {
			ret.Hops = append(ret.Hops, &pb.Hop{
				Ip:  row.hop,
				Ttl: row.ttl,
				IntersectHopType: pb.IntersectHopType_EXACT,
			})
			ret.Address = intersection
			if row.hop == row.dest {
				distanceToSource = row.ttl - intersectionTTL	
			}
			
		}
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return types.IntersectionResponse{}, err
	}
	if len(ret.Hops) == 0 {
		return types.IntersectionResponse{}, ErrNoIntFound
	}
	trTime, err := time.Parse(time.RFC3339, date)
	trTimeUnix := trTime.Unix()
	// testTime := time.Unix(trTimeUnix, 0)
	// fmt.Println(testTime)
	if err != nil {
		log.Error(err)
		return types.IntersectionResponse{}, err
	}
	return types.IntersectionResponse{
		TracerouteID: tracerouteID,
		Path: &ret,
		Src: src,
		Platform: platform,
		Timestamp: trTimeUnix,
		DistanceToSource: distanceToSource,
	}, nil
}

const (
	insertAtlasTrace = `INSERT INTO atlas_traceroutes(dest, src, source_asn, platform) VALUES(?, ?, ?, ?)`
	insertAtlasTraceBulk = `INSERT INTO atlas_traceroutes(dest, src, source_asn, platform, ` + "`date`" + `) VALUES `
	insertAtlasHop   = `
	INSERT INTO atlas_traceroute_hops(trace_id, hop, ttl, mpls) 
	VALUES (?, ?, ?, ?)`
	insertAtlasHopBulk   = `
	INSERT INTO atlas_traceroute_hops(trace_id, hop, ttl, mpls) 
	VALUES `
)

// StoreAtlasTraceroute stores a traceroute in a form that the Atlas requires
func (r *Repo) StoreAtlasTraceroute(trace *datamodel.Traceroute) (int64, error) {
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	res, err := tx.Exec(insertAtlasTrace, trace.Dst, trace.Src, trace.SourceAsn, trace.Platform)
	if err != nil {
		logError(tx.Rollback)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		logError(tx.Rollback)
		return 0, err
	}


	query := bytes.Buffer{}
	query.WriteString(insertAtlasHopBulk)
	for _, hop := range trace.GetHops() {
		isMpls := "FALSE"
		if hop.IcmpExt != nil && len(hop.IcmpExt.IcmpExtensionList) > 0  {
			for _, icmpExt := range(hop.IcmpExt.IcmpExtensionList) {
				if icmpExt.ClassNumber == 1 && icmpExt.TypeNumber == 1 {
					// https://tools.ietf.org/html/rfc4950 MPLS extension
					isMpls = "TRUE" 
				}
			} 
		} 
		query.WriteString(fmt.Sprintf(`
		(%d, %d, %d, %s),`, 
			id, hop.Addr, hop.ProbeTtl, isMpls, 
		))

	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err = tx.Exec(queryStr)
	if err != nil {
		return 0, err
	}
	// stmt, err := tx.Prepare(insertAtlasHop)
	// if err != nil {
	// 	logError(tx.Rollback)
	// 	return err
	// }
	// _, err = stmt.Exec(int32(id), trace.Src, 0)
	// if err != nil {
	// 	logError(tx.Rollback)
	// 	return err
	// }
	// for _, hop := range trace.GetHops() {
	// 	_, err := stmt.Exec(int32(id), hop.Addr, hop.ProbeTtl)
	// 	if err != nil {
	// 		logError(tx.Rollback)
	// 		return err
	// 	}
	// }
	// err = stmt.Close()
	// if err != nil {
	// 	logError(tx.Rollback)
	// 	return err
	// }
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return 0, err
	}
	return id, nil
}

func storeAtlasHopBulk(tx *sql.Tx, ids [] int64, trs[] * datamodel.Traceroute) error {
	query := bytes.Buffer {}
	query.WriteString(insertAtlasHopBulk)
	
	if len(trs) == 0{
		return nil
	}

	traceHopsCount := 0
	for i, trace := range trs {
		traceHops := trace.Hops
		if len(traceHops) == 0{
			continue
		}
		for _, hop := range traceHops{
			traceHopsCount++
			// if i == 2 {
			// 	break
			// }
			isMpls := "FALSE"
			if hop.IcmpExt != nil &&len(hop.IcmpExt.IcmpExtensionList) > 0 {
				for _, icmpExt := range(hop.IcmpExt.IcmpExtensionList) {
					if icmpExt.ClassNumber == 1 && icmpExt.TypeNumber == 1 {
						// https://tools.ietf.org/html/rfc4950 MPLS extension
						isMpls = "TRUE" 
					}
				} 
			}
			query.WriteString(fmt.Sprintf("(%d, %d, %d, %s),", 
			ids[i], hop.Addr, hop.ProbeTtl, isMpls,
			))
		}
	}
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	if traceHopsCount == 0 {
		return nil
	}
	_, err := tx.Exec(queryStr)
	if err != nil {
		return err
	}
	return nil
}

func storeAtlasTracerouteBulk(tx *sql.Tx, trs []*datamodel.Traceroute) ([] int64, error) {
	query := bytes.Buffer{}
	query.WriteString(insertAtlasTraceBulk)
	for _, in := range trs {
		// if i == 2 {
		// 	break
		// }
		date := time.Now().Format(environment.DateFormat) // This is akward but have to keep time synchronized... 

		query.WriteString(fmt.Sprintf("(%d, %d, %d, \"%s\", \"%s\"),", 
		in.Dst, in.Src, in.SourceAsn, in.Platform, date))
	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	res, err := tx.Exec(queryStr)

	if err != nil {
		return nil, err
	}
	insertedIds, err := dasql.GetIdsFromBulkInsert(int64(len(trs)), res)
	if err != nil {
		return nil, err
	} 
	return insertedIds, nil
}

// StoreAtlasTracerouteBulk stores  traceroutes in bulk 
func (r *Repo) StoreAtlasTracerouteBulk(trs []*datamodel.Traceroute) ([] int64, error) {
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return []int64{}, err
	}
	
	ids, err := storeAtlasTracerouteBulk(tx, trs)
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return []int64{}, err
	}

	err = storeAtlasHopBulk(tx, ids, trs)
	if err != nil {
		log.Error(err)
		return []int64{}, err
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return []int64{}, err
	}
	return ids, nil
}

const(
	insertTrRRHopAliasBulk = `
	INSERT INTO atlas_rr_pings(traceroute_id, ping_id, tr_hop, rr_hop, rr_hop_index, ` + "`date`" + `) VALUES 
	`
	getAlias = `
	SELECT 
	ip_address
	FROM ip_aliases 
	WHERE cluster_id in (SELECT cluster_id FROM ip_aliases WHERE ip_address = ? )	
`
	getAliasSets = `
SELECT 
	cluster_id, ip_address
FROM ip_aliases 
WHERE cluster_id in (SELECT cluster_id FROM ip_aliases WHERE ip_address in
`
)
// GetIPAliases returns the alias set of a group of IP address
func (r *Repo) GetIPAliases(ips []uint32) (map[uint32]int64, map[int64][]uint32, error) {
	conn := r.Repo.GetReader()
	query := getAliasSets 
	query += "("
	for i, ip := range(ips) {
		if i > 0 {
			query += ","	
		}
		query += fmt.Sprintf("%d", ip)
	}
	query += "))"
	// fmt.Println(query)
	rows, err:= conn.Query(query)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	aliasSetByIP := map[uint32]int64{}
	ipsByAliasSet := map[int64][]uint32 {}
	defer logError(rows.Close)
	for rows.Next() {
		var clusterID int64
		var ipAddress uint32
		err := rows.Scan(&clusterID, &ipAddress)
		if err != nil {
			log.Error(err)
			return nil, nil, err
		}
		aliasSetByIP[ipAddress] = clusterID
		if _, ok := ipsByAliasSet[clusterID]; !ok {
			ipsByAliasSet[clusterID] = []uint32  {}
		}
		ipsByAliasSet[clusterID] = append(ipsByAliasSet[clusterID], ipAddress)
	}
	if err = rows.Err(); err != nil {
		log.Error(err)
		return nil, nil, err
	}
	return aliasSetByIP, ipsByAliasSet, nil
}

// StoreTrRRHop stores a list of match trIDRRHop 
func (r *Repo) StoreTrRRHop(trIDRRHopAliass []types.TrIDRRHop) error {
	
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	query := bytes.Buffer{}
	query.WriteString(insertTrRRHopAliasBulk)
	for _, in := range trIDRRHopAliass {
		// if i == 2 {
		// 	break
		// }
		date := in.Date.Format(environment.DateFormat)
		query.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, '%s'),", 
		in.ID, in.PingID, in.TrHop, in.RRHop, in.RRHopIndex, date))
	}
	
	//trim the last ,
	queryStr := query.String()
	queryStr = queryStr[0:len(queryStr)-1]
	//format all vals at once
	_, err = tx.Exec(queryStr)
	if err != nil {
		log.Error(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

const (
	selectOldNewTraceroute = "SELECT trace_id, hop, ttl FROM atlas_traceroute_hops WHERE trace_id = ? OR trace_id = ? ORDER BY ttl, trace_id "
	selectOldTracerouteRR = "SELECT traceroute_id, tr_ttl_end WHERE rr_hop= ? WHERE traceroute_id=? or traceroute_id=?"
	updateStaleTraceroute  = 
	`UPDATE atlas_traceroutes SET stale=TRUE WHERE id = ?`
	updateStaleTraceroutes  = 
	`UPDATE atlas_traceroutes SET stale=TRUE WHERE id in (?) `
	updateStaleTraceroutesSource  = 
	`UPDATE atlas_traceroutes SET stale=TRUE WHERE dest = ? `
	updateStaleTracerouteIntersection  = 
	`UPDATE atlas_traceroutes SET stale=TRUE WHERE id in
	(SELECT distinct(trace_id) from atlas_traceroute_hops WHERE hop in  (?) )
	`

	updateRRIntersectionNonStale  = 
	`UPDATE atlas_rr_intersection SET traceroute_id= ? WHERE traceroute_id = ?`
	
	selectAvailableHopsPerDst = `
	
		SELECT A.dest, A.hop, cluster_id FROM (
			SELECT distinct dest, hop
			FROM atlas_traceroutes t 
			INNER JOIN atlas_traceroute_hops th on t.id=th.trace_id 
			WHERE t.date >= DATE_SUB(?, interval 24 hour)
		) AS A 
		LEFT JOIN 
		ip_aliases a ON a.ip_address = A.hop
		

		UNION 
		SELECT A.dest, A.rr_hop, cluster_id FROM (
			SELECT distinct dest, rr_hop
			FROM atlas_traceroutes t 
			INNER JOIN atlas_rr_intersection th on t.id=th.traceroute_id 
			WHERE t.date >= DATE_SUB(?, interval 24  hour)
		) AS A 
		LEFT JOIN 
		ip_aliases a ON a.ip_address = A.rr_hop
	
	`

	selectAvailableAliasesPerDst = `
	SELECT ip_address, cluster_id FROM ip_aliases where cluster_id IN (?)
	`




)

func (r* Repo) MarkTracerouteStaleSource(source uint32) error {
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		log.Error(err)
		return err
	}

	q,args,err := sqlx.In(updateStaleTraceroutesSource, source)
	if err != nil  {
		tx.Rollback()
		log.Error(err)
		return err
	}
	_, err = tx.Exec(q, args...)
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	return nil
}

func (r* Repo) MarkTraceroutesStale(tracerouteIDs [] int64) error {
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		log.Error(err)
		return err
	}

	q,args,err := sqlx.In(updateStaleTraceroutes, tracerouteIDs)
	if err != nil  {
		tx.Rollback()
		log.Error(err)
		return err
	}
	_, err = tx.Exec(q, args...)
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	return nil
}

func (r *Repo) MarkTracerouteStale(intersectIP uint32, oldTracerouteID int64, newTracerouteID int64) error {
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		log.Error(err)
		return err
	}

	// First select the traceroutes that changed.
	connRead := r.Repo.GetReader()
	trRows, err := connRead.Query(selectOldNewTraceroute, oldTracerouteID, newTracerouteID)
	if err != nil {
		log.Error(err)
		return err
	}

	
	type TrHop struct {
		hop uint32
		ttl uint8
	}
	oldTr := [] TrHop {}
	oldTrHopByIndex := map[uint32] int {}
	oldTrHopByTTL := map[uint8]uint32 {}
	newTrHopByIndex := map[uint32] int {}
	newTrHop := []TrHop {}
	oldTrIndex := 0
	newTrIndex := 0
	stillIntersect := false
	for trRows.Next() {
		var trId int64
		var trHop TrHop
		err := trRows.Scan(&trId, &trHop.hop, &trHop.ttl)
		if err != nil {
			log.Error(err)
			return err
		}
		if trId == oldTracerouteID {
			oldTrHopByIndex[trHop.hop] = oldTrIndex
			oldTrIndex ++
			oldTrHopByTTL[trHop.ttl] = trHop.hop
			oldTr = append(oldTr, trHop)

		} else {
			newTrHopByIndex[trHop.hop] = newTrIndex
			newTrIndex ++
			newTrHop = append(newTrHop, trHop)
		}
	}
	_, stillIntersect = newTrHopByIndex[intersectIP]
	if stillIntersect {
		return nil 
	}

	var oldMaxTTL uint8
	isIntersectionFromRR := false
	if _, ok := oldTrHopByIndex[intersectIP] ; !ok{
		isIntersectionFromRR = true
		rrRows, err := connRead.Query(selectOldTracerouteRR, intersectIP, oldTracerouteID)
		if err != nil {
			log.Error(err)
			return err
		}
		for rrRows.Next() {
			maxTTL := uint8(0)
			tracerouteID := int64(0)
			err := rrRows.Scan(&tracerouteID, &maxTTL)
			if err != nil {
				log.Error(err)
				return err
			}
			if tracerouteID == oldTracerouteID {
				oldMaxTTL = uint8(maxTTL)
			} else if tracerouteID == newTracerouteID {
				stillIntersect = true
			}
		}
	}

	if stillIntersect {
		return nil 
	}

	// Find the closest IP from the intersection that changed in common.
	// Go from the destination (M-Lab source)

	
	var changingHopIndex int
	if isIntersectionFromRR {
		changingHopIndex = oldTrHopByIndex[oldTrHopByTTL[oldMaxTTL]] 
	} else {
		isAfterIntersection := false
		for i := len(oldTr) - 1; i >=0; i-- {
			oldTrHop := oldTr[i]
			if isAfterIntersection {
				// Check that the IP is in the new path.
				if index, ok := newTrHopByIndex[oldTrHop.hop] ; ok {	
					changingHopIndex = index
					break
					// Normally it is sure that we find at least one common that is the source.
				}
			}
			if oldTrHop.hop == intersectIP {
				isAfterIntersection = true	
			}
			
		}
	}
	
	// All the traceroutes sharing a segment in TR + all RR IPs before the tr hop
	changingSegment := newTrHop[:changingHopIndex]
	changingHops := [] uint32 {}
	for _, trHop := range(changingSegment) {
		changingHops = append(changingHops, trHop.hop)
	}


	// Update the traceroutes that have an IP in common with that segment. 	
	q,args,err := sqlx.In(updateStaleTracerouteIntersection, changingHops)
	if err != nil  {
		log.Error(err)
		return err
	}

	// Update the traceroute
	_, err = tx.Exec(q, args...)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (r *Repo) UpdateRRIntersectionTracerouteNonStale(oldID int64, newID int64) error {
	
	conn := r.Repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		log.Error(err)
		return err
	}

	_, err = tx.Exec(updateRRIntersectionNonStale, oldID, newID)
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil 
	
} 

func (r *Repo) CompareOldAndNewTraceroute(intersectionIP uint32, oldTracerouteID int64, newTracerouteID int64) (bool, bool, error) {
	// Check if the traceroute has changed or not.

	conn := r.Repo.GetReader()
	
	
	trRows, err := conn.Query(selectOldNewTraceroute, oldTracerouteID, newTracerouteID)
	if err != nil {
		log.Error(err)
		return false, false, err
	}
	
	type TrHop struct {
		hop uint32
		ttl uint8
	}

	hasChanged := false
	isStillIntersect := false

	oldTrCurrentHop := TrHop{}
	defer trRows.Close()
	for trRows.Next() {
		var trId int64
		var trHop TrHop
		err := trRows.Scan(&trId, &trHop.hop, &trHop.ttl)
		if err != nil {
			log.Error(err)
			return false, false, err
		}
		if trId == oldTracerouteID {
			oldTrCurrentHop.hop = trHop.hop
			oldTrCurrentHop.ttl = trHop.ttl

		} else {
			// We just need to check if the hop is the same as the current one
			if trHop.ttl == oldTrCurrentHop.ttl && trHop.hop != oldTrCurrentHop.hop {
				// This hop has changed, so flag it as not the same. 
				hasChanged = true
			}
			if trHop.hop == intersectionIP {
				isStillIntersect = true
			}
		}
	}
	return hasChanged, isStillIntersect, nil
}



func (r *Repo) GetAvailableHopAtlasPerSource() (map[uint32]map[uint32]struct{}, error) {

	conn := r.Repo.GetReader()
	
	hopsInAtlasPerDst := map[uint32]map[uint32]struct{}{}
	clusterIDsPerDst := map[uint32]map[int64] struct {}{}
	clusterIdsSet := map[int64] struct{}{}
	clusterIds := []int64{}
	now := time.Now().Format(environment.DateFormat)
	rows, err := conn.Query(selectAvailableHopsPerDst, now, now)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var clusterID sql.NullInt64
		var dst uint32
		var hop uint32

		err := rows.Scan(&dst, &hop, &clusterID)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		if _, ok := hopsInAtlasPerDst[dst]; !ok{
			hopsInAtlasPerDst[dst] = map[uint32]struct{}{}
			clusterIDsPerDst[dst] =  map[int64]struct{}{}
		}
		hopsInAtlasPerDst[dst][hop] = struct{}{}
		if clusterID.Valid {
			clusterIDsPerDst[dst][clusterID.Int64] = struct{}{}
			clusterIdsSet[clusterID.Int64] = struct{}{}
		}

	}

	for clusterId, _ := range(clusterIdsSet) {
		clusterIds = append(clusterIds, clusterId)
	} 

	if len(clusterIds) == 0 {
		return hopsInAtlasPerDst, nil
	}
	
	// Now get the cluster IDs corresponding to the ones in aliases
	q,args,err := sqlx.In(selectAvailableAliasesPerDst, clusterIds)
	if err != nil  {
		log.Error(err)
		return nil, err
	}
	aliasRows, err := r.Repo.GetReader().Query(q, args...)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var clusterID int64
		var alias uint32
		err := aliasRows.Scan(&alias, &clusterID)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		for dst, hops := range(hopsInAtlasPerDst){
			if _, ok := clusterIDsPerDst[dst][clusterID]; ok {
				hops[alias] = struct{}{}
			} 
		}
	}

	return hopsInAtlasPerDst, nil 
}