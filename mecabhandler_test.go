package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func createHandler() *MecabHandler {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	h, err := NewMecabHandler(cfg.Mecab.Dicts)
	if err != nil {
		panic(err)
	}

	return h
}

// handler
func TestParseToNode(t *testing.T) {
	h := createHandler()

	err := h.ParseToNode("こおりつけ！")
	if err != nil {
		t.Fatalf("couldnotparsed: %s", err)
	}

	expected := []string{"こおりつけ", "動詞"}
	assert.Equal(t, expected, []string{h.Nodes[0].Surface, h.Nodes[0].Features.Pos1})

}

func TestMecabConvert(t *testing.T) {
	h := createHandler()
	url := "/api/mecab/convert"
	testfile := "test/data/test.xml"

	e := echo.New()
	e.POST(url, h.PostMecabConvert)
	testServer := httptest.NewServer(e)
	defer testServer.Close()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	fileWriter, err := writer.CreateFormFile("file", "test.xml")
	if err != nil {
		t.Fatalf("Failed to create file writer. %s", err)
	}

	readFile, err := os.Open(testfile)
	if err != nil {
		t.Fatalf("Failed to open file. %s", err)
	}
	defer readFile.Close()
	io.Copy(fileWriter, readFile)
	writer.Close()

	res, err := http.Post(testServer.URL+url, writer.FormDataContentType(), &buffer)
	if err != nil {
		t.Fatalf("Failed to POST request. %s", err)
	}

	mes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read HTTP response body. %s", err)
	}
	n := gjson.GetBytes(mes, "data").Array()[0]
	actual := []string{n.Get("surface").String(), n.Get("features.pos1").String()}

	expected := []string{"めかぶ", "名詞"}
	assert.Equal(t, expected, actual)
}
