package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	Id      int       `db:"id,primarykey,autoincrement" json:"id"`
	Name    string    `db:"name" json:"name"`
	Path    string    `db:"path" json:"path"`
	Size    int64     `db:"size" json:"size"`
	Sha256  string    `db:"sha256,notnull" json:"sha256"`
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

// GET
func (h *DbHandler) GetAllFiles(c echo.Context) error {
	var files []File
	_, err := h.DbMap.Select(&files,
		"SELECT * FROM files ORDER BY updated desc")
	if err != nil {
		return badRequest(c, "select", err)
	}
	return c.JSON(http.StatusOK, files)
}

func (h *DbHandler) GetFile(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return badRequest(c, "atoi", err)
	}
	obj, err := h.DbMap.Get(File{}, id)
	if err != nil {
		return badRequest(c, "get", err)
	}
	if obj == nil {
		return c.JSON(http.StatusNotFound, fmt.Sprintf("not found: %d", id))
	}
	return c.JSON(http.StatusOK, obj.(*File))
}

func (h *DbHandler) GetFileXML(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return badRequest(c, "atoi", err)
	}
	obj, err := h.DbMap.Get(File{}, id)
	if err != nil {
		return badRequest(c, "get", err)
	}
	if obj == nil {
		return c.JSON(http.StatusNotFound, fmt.Sprintf("not found: %d", id))
	}
	return c.Inline(filepath.Join(h.Datapath, obj.(*File).Sha256),
		obj.(*File).Name)
}

func (h *DbHandler) GetFileXMLByName(c echo.Context) error {
	name := c.Param("name")
	var f File
	err := h.DbMap.SelectOne(&f, "SELECT * FROM files WHERE name = $1", name)
	if err != nil {
		return badRequest(c, "selectone", err)
	}
	return c.Inline(filepath.Join(h.Datapath, f.Sha256), f.Name)
}

// POST
func (h *DbHandler) CreateFile(c echo.Context) error {
	var f File
	if err := c.Bind(&f); err != nil {
		return badRequest(c, "bind", err)
	}
	f.Updated = time.Now()
	fmt.Printf("path: %+v\n", f)

	if f.Path != "" {
		if f.Path[0:7] != "http://" && f.Path[0:8] != "https://" {
			return badRequest(c, "wrongpath", fmt.Errorf("missing http(s)://"))
		}
		resp, err := http.Get(f.Path)
		if err != nil {
			return badRequest(c, "filerequest", err)
		}
		defer resp.Body.Close()

		f.Size = resp.ContentLength

		f.Sha256, err = h.SaveFile(resp.Body)
		if err != nil {
			return badRequest(c, "savefile", err)
		}
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

		f.Sha256, err = h.SaveFile(src)
		if err != nil {
			return badRequest(c, "savefile", err)
		}
	}

	if f.Name == "" {
		name, err := url.PathUnescape(filepath.Base(f.Path))
		if err != nil {
			return badRequest(c, "unescape", err)
		}
		f.Name = name
	}

	if err := h.DbMap.Insert(&f); err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"files_sha256_idx\"" {
			return badRequest(c, "insert", fmt.Errorf("already exists: sha256: %s", f.Sha256))
		}
		return badRequest(c, "insert", err)
	}
	c.Logger().Infof("added: %d: %s", f.Id, f.Name)
	return c.JSON(http.StatusCreated, f)
}

func (h *DbHandler) SaveFile(src io.Reader) (hash string, err error) {
	sha := sha256.New()

	tmpfile, err := ioutil.TempFile(os.TempDir(), "temp-")
	if err != nil {
		return
	}
	defer tmpfile.Close()

	w := io.MultiWriter(sha, tmpfile)

	if _, err = io.Copy(w, src); err != nil {
		return
	}
	hash = fmt.Sprintf("%x", sha.Sum(nil))

	fmt.Printf("rename: %s => %s", tmpfile.Name(), filepath.Join(h.Datapath, hash))
	err = os.Rename(tmpfile.Name(), filepath.Join(h.Datapath, hash))
	if err != nil {
		return
	}

	return
}

func (h *DbHandler) UpdateFile(c echo.Context) error {
	var f File
	if err := c.Bind(&f); err != nil {
		return badRequest(c, "bind", err)
	}
	f.Updated = time.Now()

	fmt.Printf("\nok: %+v\n", f)

	count, err := h.DbMap.Update(&f)
	if err != nil {
		return badRequest(c, "update", err)
	}
	if count != 1 {
		return badRequest(c, "update",
			fmt.Errorf("something wrong: updated: %d", count))
	}

	c.Logger().Infof("updated: %d: %s", f.Id, f.Name)
	return c.JSON(http.StatusCreated, f)
}

// DELETE
func (h *DbHandler) DeleteFile(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return badRequest(c, "atoi", err)
	}
	obj, err := h.DbMap.Get(File{}, id)
	if err != nil {
		return badRequest(c, "get", err)
	}
	if obj == nil {
		return badRequest(c, "get", fmt.Errorf("not found: %d", id))
	}

	f := obj.(*File)
	_, err = h.DbMap.Delete(f)
	if err != nil {
		return badRequest(c, "delete", err)
	}
	err = os.Remove(filepath.Join(h.Datapath, f.Sha256))
	if err != nil {
		return badRequest(c, "remove", err)
	}

	c.Logger().Infof("deleted: %d: %s", f.Id, f.Name)
	return c.JSON(http.StatusOK, f)
}
