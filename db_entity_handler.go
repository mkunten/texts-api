package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type Entity struct {
	ID           string    `db:"id,notnull,primarykey" json:"id"`
	Type         string    `db:"type,notnull" json:"type"`
	Updated      time.Time `db:"updated,notnull" json:"updated"`
	AltLabels    []string  `db:"-" json:"altLabels"`
	ExactMatches []string  `db:"-" json:"exactMatches"`
}

type EntityAltLabel struct {
	EntityID string `db:"entity_id,notnull" json"entityID"`
	AltLabel string `db:"alt_label,notnull" json:"altLabel"`
}

type EntityExactMatch struct {
	EntityID   string `db:"entity_id,notnull" json"entityID"`
	ExactMatch string `db:"exact_match,notnull" json:"exactMatch"`
}

var lockEntity = sync.Mutex{}

// GET
func (h *DbHandler) GetAllEntities(c echo.Context) error {
	var entities []Entity
	_, err := h.DbMap.Select(&entities,
		"SELECT id, type, updated FROM entities ORDER BY updated")
	if err != nil {
		return badRequest(c, "selectentities", err)
	}
	var entityAltLabels []EntityAltLabel
	_, err = h.DbMap.Select(&entityAltLabels,
		"SELECT entity_id, alt_label FROM entities_alt_labels")
	if err != nil {
		return badRequest(c, "selectentities", err)
	}
	var entityExactMatches []EntityExactMatch
	_, err = h.DbMap.Select(&entityExactMatches,
		"SELECT entity_id, exact_match FROM entities_exact_matches")
	if err != nil {
		return badRequest(c, "selectentities", err)
	}
	al := map[string][]string{}
	for _, el := range entityAltLabels {
		al[el.EntityID] = append(al[el.EntityID], el.AltLabel)
	}
	em := map[string][]string{}
	for _, el := range entityExactMatches {
		em[el.EntityID] = append(em[el.EntityID], el.ExactMatch)
	}
	for idx, e := range entities {
		if m, ok := al[e.ID]; ok {
			entities[idx].AltLabels = m
		} else {
			entities[idx].AltLabels = []string{}
		}
		if m, ok := em[e.ID]; ok {
			entities[idx].ExactMatches = m
		} else {
			entities[idx].ExactMatches = []string{}
		}
	}
	return c.JSON(http.StatusOK, entities)
}

func (h *DbHandler) GetEntity(c echo.Context) error {
	id := c.Param("id")
	obj, err := h.DbMap.Get(Entity{}, id)
	if err != nil {
		return badRequest(c, "getentity", err)
	}
	if obj == nil {
		return notFound(c, "getentity", id)
	}
	var altLabels []string
	if _, err = h.DbMap.Select(&altLabels,
		"SELECT alt_label FROM entities_alt_labels WHERE entity_id = $1", id); err != nil {
		return badRequest(c, "getentity", err)
	}
	var exactMatches []string
	if _, err = h.DbMap.Select(&exactMatches,
		"SELECT exact_match FROM entities_exact_matches WHERE entity_id = $1", id); err != nil {
		return badRequest(c, "getentity", err)
	}
	entity := obj.(*Entity)
	entity.AltLabels = altLabels
	entity.ExactMatches = exactMatches
	return c.JSON(http.StatusOK, entity)
}

// POST
func (h *DbHandler) CreateEntity(c echo.Context) error {
	lockEntity.Lock()
	defer lockEntity.Unlock()

	e := &Entity{}
	if err := c.Bind(e); err != nil {
		return badRequest(c, "bind", err)
	}
	e.Updated = time.Now()

	if e.ID == "" {
		return badRequest(c, "bind", fmt.Errorf("id is empty"))
	}

	trans, err := h.DbMap.Begin()
	if err != nil {
		return badRequest(c, "trans", err)
	}
	if err = trans.Insert(e); err != nil {
		return badRequest(c, "insert", err)
	}
	for _, s := range e.AltLabels {
		al := &EntityAltLabel{
			EntityID: e.ID,
			AltLabel: s,
		}
		if err = trans.Insert(al); err != nil {
			return badRequest(c, "insert", err)
		}
	}
	for _, s := range e.ExactMatches {
		em := &EntityExactMatch{
			EntityID:   e.ID,
			ExactMatch: s,
		}
		if err = trans.Insert(em); err != nil {
			return badRequest(c, "insert", err)
		}
	}
	if err = trans.Commit(); err != nil {
		return badRequest(c, "trans", err)
	}

	c.Logger().Infof("added: entities: %s", e.ID)
	return c.JSON(http.StatusCreated, e)
}

// PUT
func (h *DbHandler) UpdateEntity(c echo.Context) error {
	lockEntity.Lock()
	defer lockEntity.Unlock()

	e := &Entity{
		ID: c.Param("id"),
	}
	if err := c.Bind(&e); err != nil {
		return badRequest(c, "bind", err)
	}
	e.Updated = time.Now()

	trans, err := h.DbMap.Begin()
	if err != nil {
		return badRequest(c, "trans", err)
	}
	count, err := trans.Update(&e)
	if err != nil {
		return badRequest(c, "update", err)
	}
	if count != 1 {
		return badRequest(c, "update",
			fmt.Errorf("something wrong: update: %d", count))
	}
	if _, err = trans.Exec("DELETE FROM entities_alt_labels WHERE entity_id = $1", e.ID); err != nil {
		return badRequest(c, "deleteentity", err)
	}
	for _, s := range e.AltLabels {
		al := &EntityAltLabel{
			EntityID: e.ID,
			AltLabel: s,
		}
		if err = trans.Insert(al); err != nil {
			return badRequest(c, "insert", err)
		}
	}
	if _, err = trans.Exec("DELETE FROM entities_exact_matches WHERE entity_id = $1", e.ID); err != nil {
		return badRequest(c, "deleteentity", err)
	}
	for _, s := range e.ExactMatches {
		em := &EntityExactMatch{
			EntityID:   e.ID,
			ExactMatch: s,
		}
		if err = trans.Insert(em); err != nil {
			return badRequest(c, "insert", err)
		}
	}
	if err = trans.Commit(); err != nil {
		return badRequest(c, "trans", err)
	}

	c.Logger().Infof("updated: %s", e.ID)
	return c.JSON(http.StatusCreated, e)
}

// DELETE
func (h *DbHandler) DeleteEntity(c echo.Context) error {
	lockEntity.Lock()
	defer lockEntity.Unlock()

	e := &Entity{
		ID: c.Param("id"),
	}

	trans, err := h.DbMap.Begin()
	if err != nil {
		return badRequest(c, "trans", err)
	}
	count, err := trans.Delete(e)
	if err != nil {
		return badRequest(c, "deleteentity", err)
	}
	if count != 1 {
		return notFound(c, "deleteentity", e.ID)
	}
	if _, err = trans.Exec("DELETE FROM entities_alt_labels WHERE entity_id = $1", e.ID); err != nil {
		return badRequest(c, "deleteentity", err)
	}
	if _, err = trans.Exec("DELETE FROM entities_exact_matches WHERE entity_id = $1", e.ID); err != nil {
		return badRequest(c, "deleteentity", err)
	}
	if err = trans.Commit(); err != nil {
		return badRequest(c, "trans", err)
	}

	c.Logger().Infof("deleted: %s", e.ID)
	return c.JSON(http.StatusOK, map[string]string{"id": e.ID})
}
