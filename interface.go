package db

import (
	"database/sql"
	"time"
)

type Driver int

const (
	DriverMysql Driver = iota
	DriverMssql
)

type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Driver() string

	debugMode() bool
}

type dbStats struct {
	queryCount int
	timerStart time.Time
}

type DB struct {
	db     *sql.DB
	driver Driver
	stats  *dbStats
	debug  bool
}

type TX struct {
	tx         *sql.Tx
	r          *DB
	queryCount int
}

type TxFn func(q *TX) error

func (q *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	result, err := q.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	q.stats.queryCount++
	return result, nil
}

func (q *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	q.stats.queryCount++
	return rows, nil
}

func (q *DB) Begin(fn TxFn) (err error) {
	tx, err := q.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = fn(&TX{tx: tx, r: q})
	return err
}

// Debug clones the DB information object and sets it's debug mode to true.
// The stats will be shared between the two objects
func (q *DB) Debug() *DB {
	if q.debug {
		return q
	}
	return &DB{
		db:     q.db,
		stats:  q.stats,
		driver: q.driver,
		debug:  true,
	}
}

func (q *DB) Driver() Driver {
	return q.driver
}

func (q *DB) debugMode() bool {
	return q.debug
}

func (q *TX) Exec(query string, args ...interface{}) (sql.Result, error) {
	result, err := q.tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	q.queryCount++
	return result, nil
}

func (q *TX) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := q.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	q.queryCount++
	return rows, nil
}

func (q *TX) Commit() error {
	err := q.tx.Commit()
	if err != nil {
		return err
	}
	q.r.stats.queryCount += q.queryCount
	return nil
}

func (q *TX) Rollback() error {
	return q.tx.Rollback()
}

func (q *TX) Driver() Driver {
	return q.r.driver
}

func (q *TX) debugMode() bool {
	return q.r.debug
}
