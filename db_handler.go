package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"gopkg.in/gorp.v2"
)

type DbHandler struct {
	Db       *sql.DB
	DbMap    *gorp.DbMap
	Datapath string
}

// constructor
func NewDbHandler(cfg *config) (dbh *DbHandler, err error) {
	dbh = &DbHandler{}
	dbh.Db, err = sql.Open("postgres", cfg.Db.Dsn)
	if err != nil {
		return
	}

	dbh.DbMap = &gorp.DbMap{
		Db:      dbh.Db,
		Dialect: gorp.PostgresDialect{},
	}

	t := dbh.DbMap.AddTableWithName(File{}, "files")
	t.AddIndex("files_sha256_idx", "Btree", []string{"sha256"}).SetUnique(true)

	dbh.DbMap.TraceOn("[gorp]",
		log.New(os.Stdout, "texts-api:", log.Lmicroseconds))

	if err = dbh.DbMap.CreateTablesIfNotExists(); err != nil {
		log.Fatal(err)
	}
	if err = dbh.DbMap.CreateIndex(); err != nil &&
		!strings.HasSuffix(err.Error(), "already exists") &&
		!strings.HasSuffix(err.Error(), "すでに存在します") {
		log.Fatal(err)
	}

	dbh.Datapath = cfg.Db.Datapath
	err = os.MkdirAll(dbh.Datapath, 0755)
	if err != nil {
		return
	}

	return dbh, err
}
