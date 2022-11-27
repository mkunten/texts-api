package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type JSONData struct {
	Key     string          `db:"key,primarykey" json:"key"`
	Data    json.RawMessage `db:"data,notnull" json:"data"`
	Updated time.Time       `db:"updated" json:"updated"`
}

var lockJSONData = sync.Mutex{}

// GET
func (h *DbHandler) GetJSONData(c echo.Context) error {
	key := c.Param("key")
	obj, err := h.DbMap.Get(JSONData{}, key)
	if err != nil {
		return badRequest(c, "getjsondata", err)
	}
	if obj == nil {
		return notFound(c, "getjsondata", key)
	}
	return c.JSON(http.StatusOK, obj.(*JSONData))
}

// POST
func (h *DbHandler) CreateJSONData(c echo.Context) error {
	lockJSONData.Lock()
	defer lockJSONData.Unlock()

	jd := &JSONData{
		Key:     c.Param("key"),
		Updated: time.Now(),
	}
	if err := c.Bind(jd); err != nil {
		return badRequest(c, "bind", err)
	}

	query := "INSERT INTO json_data (key, data, updated) VALUES ($1, $2, $3)"
	if _, err := h.DbMap.Exec(query, jd.Key, jd.Data, jd.Updated); err != nil {
		return badRequest(c, "insert", err)
	}
	c.Logger().Infof("added: jsonData: %s", jd.Key)
	return c.JSON(http.StatusCreated, jd)
}

// PUT

// DELETE
func (h *DbHandler) DeleteJSONData(c echo.Context) error {
	lockJSONData.Lock()
	defer lockJSONData.Unlock()

	jd := &JSONData{
		Key: c.Param("key"),
	}
	count, err := h.DbMap.Delete(jd)
	if err != nil {
		return badRequest(c, "deletejsondata", err)
	}
	if count != 1 {
		return notFound(c, "deletejsondata", jd.Key)
	}

	c.Logger().Infof("deleted: %s", jd.Key)
	return c.JSON(http.StatusOK, jd.Key)
}
