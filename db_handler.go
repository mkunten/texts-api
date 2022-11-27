package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/go-gorp/gorp/v3"
	_ "github.com/lib/pq"
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
		Dialect: CustomPostgresDialect{},
	}

	t := dbh.DbMap.AddTableWithName(File{}, "files")
	t.AddIndex("files_sha256_idx", "Btree", []string{"sha256"}).SetUnique(true)

	t = dbh.DbMap.AddTableWithName(JSONData{}, "json_data")

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

// CustomPostgresDialect for gorp to manipulate json
/*
	original:
	https://github.com/go-gorp/gorp/issues/254#issuecomment-248733253
*/

type CustomPostgresDialect struct {
	gorp.PostgresDialect
}

func (d CustomPostgresDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	if val == reflect.TypeOf((json.RawMessage)(nil)) {
		return "jsonb"
	}
	return d.PostgresDialect.ToSqlType(val, maxsize, isAutoIncr)
}
