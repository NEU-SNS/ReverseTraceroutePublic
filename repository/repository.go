package repository

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	// import the mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// DB represents a database collection
type DB struct {
	wdb []*sql.DB
	rdb []*sql.DB
	rr  *rand.Rand
	wr  *rand.Rand
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

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local"

func makeDb(conf Config) (*sql.DB, error) {
	conString := fmt.Sprintf(conFmt, conf.User, conf.Password, conf.Host, conf.Port, conf.Db)
	db, err := sql.Open("mysql", conString)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if err = db.Ping(); err != nil {
		log.Error(err)
		return nil, err
	}
	db.SetMaxOpenConns(0)
	db.SetMaxIdleConns(512)
	
	db.SetConnMaxLifetime(0)
	return db, nil
}

// NewDB creates a new DB with the given config
func NewDB(con DbConfig) (*DB, error) {
	ret := &DB{}
	ret.rr = rand.New(rand.NewSource(time.Now().UnixNano()))
	ret.wr = rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, conf := range con.WriteConfigs {
		db, err := makeDb(conf)
		if err != nil {
			return nil, err
		}
		ret.wdb = append(ret.wdb, db)
	}
	for _, conf := range con.ReadConfigs {
		db, err := makeDb(conf)
		if err != nil {
			return nil, err
		}
		ret.rdb = append(ret.rdb, db)
	}
	return ret, nil
}

// GetReader gets a sql.DB that is configured for reading
func (db *DB) GetReader() *sql.DB {
	l := len(db.rdb)
	if l == 1 {
		return db.rdb[0]
	}
	return db.rdb[db.rr.Intn(len(db.rdb))]
}

// GetWriter gets a sql.DB that is configured for writing
func (db *DB) GetWriter() *sql.DB {
	l := len(db.wdb)
	if l == 1 {
		return db.wdb[0]
	}
	return db.wdb[db.wr.Intn(len(db.wdb))]
}

// Close closes the DB connections
func (db *DB) Close() error {
	for _, d := range db.wdb {
		d.Close()
	}
	for _, d := range db.rdb {
		d.Close()
	}
	return nil
}
