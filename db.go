package sorm

import (
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

var ErrEmptyResult = fmt.Errorf("empty result")

func Open(db *sql.DB, driver Driver) *DB {
	go periodicPing(db, pingOffset)
	return &DB{
		db:     db,
		stats:  &dbStats{},
		driver: driver,
	}
}

var pingOffset time.Duration = 0

func periodicPing(db *sql.DB, offset time.Duration) {
	for {
		time.Sleep(time.Second*30 + offset)
		err := db.Ping()
		if err != nil {
			fmt.Println(err)
		}
	}
}
