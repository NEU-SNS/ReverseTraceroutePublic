/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

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

// Package dataaccess provides database access
package dataaccess

import (
	sqllib "database/sql"
	"net"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

// DbConfig is a database configuration
type DbConfig struct {
	WriteConfigs []Config
	ReadConfigs  []Config
	Environment string 
}

// Config is a DB server config
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

// DataAccess represents data access
type DataAccess struct {
	conf DbConfig
	db   *sql.DB
}

func uToNSec(u int64) int64 {
	//1000 nsec to a usec
	return u * 1000
}

// Monitor get the stats about the db
func (d *DataAccess) Monitor() (sqllib.DBStats, sqllib.DBStats) {
	return d.db.GetReader().Stats(), d.db.GetWriter().Stats()
}

// StoreTraceroute stores a traceroute
func (d *DataAccess) StoreTraceroute(t *dm.Traceroute) (int64, error) {
	return d.db.StoreTraceroute(t)
}

// StoreTracerouteBulk stores traceroutes in bulk
func (d *DataAccess) StoreTracerouteBulk(ts []*dm.Traceroute) ([]int64, error) {
	return d.db.StoreTracerouteBulk(ts)
}

// GetTRBySrcDst gets a trace by src and dst
func (d *DataAccess) GetTRBySrcDst(src, dst uint32) ([]*dm.Traceroute, error) {
	return d.db.GetTRBySrcDst(src, dst)
}

// GetTRBySrcDstWithStaleness gets a trace with the src and dst no older than the give time
func (d *DataAccess) GetTRBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Traceroute, error) {
	return d.db.GetTRBySrcDstWithStaleness(src, dst, s)
}

// GetTraceMulti gets traceroutes that match the given TracerouteMeasurements
func (d *DataAccess) GetTraceMulti(in []*dm.TracerouteMeasurement) ([]*dm.Traceroute, error) {
	return d.db.GetTraceMulti(in)
}

// GetPingBySrcDst gets a ping
func (d *DataAccess) GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error) {
	return d.db.GetPingBySrcDst(src, dst)
}

// GetPingBySrcDstWithStaleness gets a ping
func (d *DataAccess) GetPingBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Ping, error) {
	return d.db.GetPingBySrcDstWithStaleness(src, dst, 0, s)
}


// StorePings stores pings in bulk
func (d *DataAccess) StorePingBulk(ps [] *dm.Ping) ([]int64, error) {
	return d.db.StorePingBulk(ps)
}

// StorePing stores a ping
func (d *DataAccess) StorePing(p *dm.Ping) (int64, error) {
	return d.db.StorePing(p)
}

// Close closes
func (d *DataAccess) Close() error {
	return d.db.Close()
}

// GetPingsMulti gets pings that match the given PingMeasurements
func (d *DataAccess) GetPingsMulti(in []*dm.PingMeasurement) ([]*dm.Ping, error) {
	return d.db.GetPingsMulti(in)
}

// UpdateCheckStatus updates the health check status for a vp
func (d *DataAccess) UpdateCheckStatus(ip uint32, stat string) error {
	return d.db.UpdateCheckStatus(ip, stat)
}

// GetVPs get the VPs
func (d *DataAccess) GetVPs() ([]*dm.VantagePoint, error) {
	return d.db.GetVPs()
}

// ClearAllVPs nulls the controller of all VPS
func (d *DataAccess) ClearAllVPs() error {
	return d.db.ClearAllVPs()
}

// GetActiveVPs get the vps which are currently connected to the controller
func (d *DataAccess) GetActiveVPs() ([]*dm.VantagePoint, error) {
	return d.db.GetActiveVPs()
}

// GetVPOwner returns the owner of a VP
func (d *DataAccess) InsertVP(ip uint32, contr uint32) error {
	return d.db.InsertVP(ip, contr)
}

// UpdateController updates the controller for a VP
func (d *DataAccess) UpdateController(ip, old, nc uint32) error {
	return d.db.UpdateController(ip, old, nc)
}

// GetVPOwner returns the owner of a VP
func (d *DataAccess) GetVPOwner(ip uint32) (dm.VPOwner, error) {
	return d.db.GetVPOwner(ip)
}

// StoreAdjacency stores an ajacienty
func (d *DataAccess) StoreAdjacency(l, r net.IP) error {
	return d.db.StoreAdjacency(l, r)
}

// StoreAdjacencyToDest stores and adjacency to dest
func (d *DataAccess) StoreAdjacencyToDest(dest24, addr, adj net.IP) error {
	return d.db.StoreAdjacencyToDest(dest24, addr, adj)
}

// StoreAlias stores an alias with id id
func (d *DataAccess) StoreAlias(id int, ips []net.IP) error {
	return d.db.StoreAlias(id, ips)
}

// GetUser gets a user for the given key
func (d *DataAccess) GetUser(key string) (dm.User, error) {
	return d.db.GetUser(key)
}

// AddPingBatch adds a ping batch for user u
func (d *DataAccess) AddPingBatch(u dm.User) (int64, error) {
	return d.db.AddPingBatch(u)
}

// AddPingsToBatch adds pings pids to batch bid
func (d *DataAccess) AddPingsToBatch(bid int64, pids []int64) error {
	return d.db.AddPingsToBatch(bid, pids)
}

// GetPingBatch get the pings in the associated batch
func (d *DataAccess) GetPingBatch(u dm.User, bid int64) ([]*dm.Ping, error) {
	return d.db.GetPingBatch(u, bid)
}

// GetPingBatch get the pings in the associated batch
func (d *DataAccess) GetPingByLabel(label string) ([]*dm.Ping, error) {
	return d.db.GetPingByLabel(label)
}

// GetPingBatch get the number of responses in the associated batch
func (d *DataAccess) GetPingBatchNResponses(u dm.User, bid int64) (uint32, error) {
	return d.db.GetPingBatchNResponses(u, bid)
}

// AddTraceBatch adds a traceroute batch for user u
func (d *DataAccess) AddTraceBatch(u dm.User) (int64, error) {
	return d.db.AddTraceBatch(u)
}

// AddTraceToBatch adds traceroutes tids to batch bid
func (d *DataAccess) AddTraceToBatch(bid int64, tids []int64) error {
	return d.db.AddTraceToBatch(bid, tids)
}

// GetTraceBatch get the traceroute in the associated batch
func (d *DataAccess) GetTraceBatch(u dm.User, bid int64) ([]*dm.Traceroute, error) {
	return d.db.GetTraceBatch(u, bid)
}

// GetPingBatch get the pings in the associated batch
func (d *DataAccess) GetTracerouteByLabel(label string) ([]*dm.Traceroute, error) {
	return d.db.GetTracerouteByLabel(label)
}

func (d *DataAccess) GetMaxRevtrMeasurementID(measurementType string) (uint64, error) {
	return d.db.GetMaxRevtrMeasurementID(measurementType)
}

// New create a new dataAccess with the given config
func New(c DbConfig) (*DataAccess, error) {
	var conf sql.DbConfig
	for _, cc := range c.ReadConfigs {
		conf.ReadConfigs = append(conf.ReadConfigs, sql.Config{
			User:     cc.User,
			Password: cc.Password,
			Host:     cc.Host,
			Port:     cc.Port,
			Db:       cc.Db,
		})
	}
	for _, cc := range c.WriteConfigs {
		conf.WriteConfigs = append(conf.WriteConfigs, sql.Config{
			User:     cc.User,
			Password: cc.Password,
			Host:     cc.Host,
			Port:     cc.Port,
			Db:       cc.Db,
		})
	}
	conf.Environment = c.Environment
	db, err := sql.NewDB(conf)
	if err != nil {
		return nil, err
	}
	return &DataAccess{conf: c, db: db}, nil
}
