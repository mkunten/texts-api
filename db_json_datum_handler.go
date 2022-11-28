package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type JSONDatum struct {
	Key     string          `db:"key,notnull,primarykey" json:"key"`
	Type    string          `db:"type,notnull" json:"type"`
	Data    json.RawMessage `db:"data,notnull" json:"data,omitempty"`
	Updated time.Time       `db:"updated,notnull" json:"updated"`
}

var lockJSONDatum = sync.Mutex{}

// GET
func (h *DbHandler) GetAllJSONData(c echo.Context) error {
	var jsonData []JSONDatum
	_, err := h.DbMap.Select(&jsonData,
		"SELECT key, type, updated FROM json_data ORDER BY updated")
	if err != nil {
		return badRequest(c, "selectjsondata", err)
	}
	return c.JSON(http.StatusOK, jsonData)
}

func (h *DbHandler) GetJSONDatum(c echo.Context) error {
	key := c.Param("key")
	obj, err := h.DbMap.Get(JSONDatum{}, key)
	if err != nil {
		return badRequest(c, "getjsondata", err)
	}
	if obj == nil {
		return notFound(c, "getjsondata", key)
	}
	return c.JSON(http.StatusOK, obj.(*JSONDatum))
}

// POST
func (h *DbHandler) CreateJSONDatum(c echo.Context) error {
	lockJSONDatum.Lock()
	defer lockJSONDatum.Unlock()

	jd := &JSONDatum{}
	if err := c.Bind(jd); err != nil {
		return badRequest(c, "bind", err)
	}
	jd.Updated = time.Now()

	if jd.Key == "" {
		return badRequest(c, "bind", fmt.Errorf("key is empty"))
	}
	if jd.Type == "" {
		return badRequest(c, "bind", fmt.Errorf("type is empty"))
	}

	query := "INSERT INTO json_data (key, type, data, updated) VALUES ($1, $2, $3, $4)"
	if _, err := h.DbMap.Exec(query, jd.Key, jd.Type, jd.Data, jd.Updated); err != nil {
		return badRequest(c, "insert", err)
	}
	c.Logger().Infof("added: jsonDatum: %s", jd.Key)
	return c.JSON(http.StatusCreated, jd)
}

// PUT
func (h *DbHandler) UpdateJSONDatum(c echo.Context) error {
	lockJSONDatum.Lock()
	defer lockJSONDatum.Unlock()

	jd := &JSONDatum{
		Key: c.Param("key"),
	}
	if err := c.Bind(&jd); err != nil {
		return badRequest(c, "bind", err)
	}
	jd.Updated = time.Now()

	// count, err := h.DbMap.Update(&jd)
	query := "UPDATE json_data SET (data, updated) = ($1, $2) WHERE key = $3"
	res, err := h.DbMap.Exec(query, jd.Data, jd.Updated, jd.Key)
	if err != nil {
		return badRequest(c, "update", err)
	}
	if count, _ := res.RowsAffected(); count != 1 {
		return badRequest(c, "update",
			fmt.Errorf("something wrong: update: %d", count))
	}

	c.Logger().Infof("updated: %s", jd.Key)
	return c.JSON(http.StatusCreated, jd)
}

// DELETE
func (h *DbHandler) DeleteJSONDatum(c echo.Context) error {
	lockJSONDatum.Lock()
	defer lockJSONDatum.Unlock()

	jd := &JSONDatum{
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
