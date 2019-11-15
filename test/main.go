package main

import (
	"database/sql"
	"fmt"
	"github.com/n1xx1/sorm"
	"log"
	"time"
)

type testBoard struct {
	ID int `db:"board_id,primary,autoincrement"`
}

func (*testBoard) TableName() string {
	return "boards"
}

type testGame struct {
	ID          int    `db:"game_id,primary,autoincrement"`
	Name        string `db:"game_name"`
	LastVersion string `db:"game_last_version"`
}

func (*testGame) TableName() string {
	return "games"
}

type testSaveFile struct {
	ID          int        `db:"save_id,primary,autoincrement"`
	Date        *time.Time `db:"save_date"`
	BoardID     int        `db:"board_id" dbfk:"testBoard"`
	GameID      int        `db:"game_id" dbfk:"testGame"`
	GameVersion string     `db:"game_version"`
}

func (*testSaveFile) TableName() string {
	return "save_files"
}

func main() {
	sorm.AddModel(&testGame{})
	sorm.AddModel(&testBoard{})
	sorm.AddModel(&testSaveFile{})

	sqldb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		"root", "root", "localhost:3306", "teburu"))
	if err != nil {
		log.Fatal(err)
	}

	db := sorm.Open(sqldb, sorm.DriverMysql)
	game := testGame{
		Name: "zce",
	}
	err = sorm.Select(db, &game)
	if err == sorm.ErrEmptyResult {
		log.Fatal("non existing game")
	} else if err != nil {
		log.Fatalf("db error: %s", err)
	}
}
