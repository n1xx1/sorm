package main

import (
	"database/sql"
	"fmt"
	"github.com/n1xx1/sorm"
	"log"
	"time"
)

type TestTable struct {
	ID   int    `db:"game_id,primary,autoincrement"`
	Name string `db:"game_name"`
	Date *time.Time
}

func (*TestTable) TableName() string {
	return "test"
}

func main() {
	sorm.AddModel(&TestTable{})

	sqldb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		"root", "root", "localhost:3306", "example"))
	if err != nil {
		log.Fatal(err)
	}

	db := sorm.Open(sqldb, sorm.DriverMysql)
	_ = db
}
