package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"gopkg.in/gorp.v2"
)

type DbHandler struct {
	Db       *sql.DB
	DbMap    *gorp.DbMap
	Datapath string
}

type File struct {
	Name    string    `db:"name" json:"name"`
	Path    string    `db:"path" json:"path"`
	Size    int64     `db:"size" json:"size"`
	Sha256  string    `db:"sha256" json:"sha256"`
	Updated time.Time `db:"updated" json:"updated"`
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

	dbh.DbMap.AddTableWithName(File{}, "files")

	dbh.DbMap.TraceOn("[gorp]", log.New(os.Stdout, "textanalysis-api:", log.Lmicroseconds))

	if err = dbh.DbMap.CreateTablesIfNotExists(); err != nil {
		log.Fatal(err)
	}

	dbh.Datapath = cfg.Db.Datapath
	err = os.MkdirAll(dbh.Datapath, 0755)
	if err != nil {
		return
	}

	return dbh, err
}

// GET
func (h *DbHandler) GetFiles(c echo.Context) error {
	var files []File
	_, err := h.DbMap.Select(&files,
		"SELECT * FROM files ORDER BY updated desc")
	if err != nil {
		return badRequest(c, "select", err)
	}
	return c.JSON(http.StatusOK, files)
}

// POST
func (h *DbHandler) PostFiles(c echo.Context) error {
	var f File
	if err := c.Bind(&f); err != nil {
		return badRequest(c, "bind", err)
	}
	f.Updated = time.Now()

	if f.Path != "" {
	} else {
		fh, err := c.FormFile("file")
		if err != nil {
			return badRequest(c, "formfile", err)
		}
		f.Path = fh.Filename
		f.Size = fh.Size

		src, err := fh.Open()
		if err != nil {
			return badRequest(c, "fileopen", err)
		}
		defer src.Close()

		sha := sha256.New()
		if _, err := io.Copy(sha, src); err != nil {
			return badRequest(c, "shacopy", err)
		}
		f.Sha256 = fmt.Sprintf("%x", sha.Sum(nil))

		src.Seek(0, 0)
		dst, err := os.Create(filepath.Join(h.Datapath, f.Sha256))
		if err != nil {
			return badRequest(c, "dstcreate", err)
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return badRequest(c, "dstcopy", err)
		}
	}

	if f.Name == "" {
		f.Name = filepath.Base(f.Path)
	}

	if err := h.DbMap.Insert(&f); err != nil {
		return badRequest(c, "insert", err)
	}
	c.Logger().Infof("added: %s", f.Name)
	return c.JSON(http.StatusCreated, f)
}
