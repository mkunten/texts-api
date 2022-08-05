package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/bluele/mecab-golang"
	"github.com/jszwec/csvutil"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	Mecab      *mecab.MeCab
	Nodes      []MecabNode
	NodeHeader []string
	UnkHeader  []string
}

type MecabNode struct {
	ID       int          `json:"id"`
	Length   int          `json:"length"`
	Stat     int          `json:"stat"`
	StartPos int          `json:"startPos"`
	Surface  string       `json:"surface"`
	Features NodeFeatures `json:"features"`
}

type MecabStat int

const (
	MECAB_NOR_NODE MecabStat = iota
	MECAB_UNK_NODE
	MECAB_BOS_NODE
	MECAB_EOS_NODE
)

type NodeFeatures struct {
	Pos1     string `json:"pos1"`     // 品詞大分類
	Pos2     string `json:"pos2"`     // 品詞中分類
	Pos3     string `json:"pos3"`     // 品詞小分類
	Pos4     string `json:"pos4"`     // 品詞細分類
	CType    string `json:"cType"`    // 活用型
	CForm    string `json:"cForm"`    // 活用形
	LForm    string `json:"lForm"`    // 語彙素読み
	Lemma    string `json:"lemma"`    // 語彙素（＋語彙素細分類）
	Orth     string `json:"orth"`     // 書字形出現形
	Pron     string `json:"pron"`     // 発音形出現形
	Kana     string `json:"kana"`     // 仮名形出現形
	Goshu    string `json:"goshu"`    // 語種
	OrthBase string `json:"orthBase"` // 書字形基本形
	PronBase string `json:"pronBase"` // 発音形基本形
	KanaBase string `json:"kanaBase"` // 仮名形基本形
	FormBase string `json:"formBase"` // 語形基本形
	IType    string `json:"iType"`    // 語頭変化化型
	IForm    string `json:"iForm"`    // 語頭変化形
	IConType string `json:"iConType"` // 語頭変化結合型
	FType    string `json:"fType"`    // 語末変化化型
	FForm    string `json:"fForm"`    // 語末変化形
	FConType string `json:"fConType"` // 語末変化結合型
	AType    string `json:"aType"`    // アクセント型
	AConType string `json:"aConType"` // アクセント結合型
	AModType string `json:"aModType"` // アクセント修飾型
	Lid      string `json:"lid"`      // 語彙表ID
	Lemma_id string `json:"lemma_id"` // 語彙素ID
}

func (mn MecabNode) String() string {
	return fmt.Sprintf("%d,%d,%d:%s:%s",
		mn.ID, mn.Length, mn.StartPos, mn.Surface, mn.Features)
}

// constructor
func NewHandler(dicts []string) (h *Handler, err error) {
	h = &Handler{}
	h.Mecab, err = mecab.New(
		fmt.Sprintf("-d %s", strings.Join(dicts, " ")))
	if err != nil {
		return h, err
	}
	h.NodeHeader, err = csvutil.Header(NodeFeatures{}, "csv")
	return h, err
}

// POST
func (h *Handler) PostMecabConvert(c echo.Context) error {
	// read an uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return badRequest(c, "nofile", err)
	}
	src, err := file.Open()
	if err != nil {
		return badRequest(c, "nofile", err)
	}
	defer src.Close()

	// apply xpath
	var s string
	if filepath.Ext(file.Filename) == ".xml" {
		doc, err := xmlquery.Parse(src)
		if err != nil {
			return badRequest(c, "notxml", err)
		}
		t := xmlquery.FindOne(doc, "//TEI/text/body")
		s = t.OutputXML(false)
	} else {
		data, err := ioutil.ReadAll(src)
		if err != nil {
			return badRequest(c, "nofile", err)
		}
		s = string(data)
	}

	err = h.ParseToNode(s)
	if err != nil {
		return badRequest(c, "couldnotparsed", err)
	}

	c.Response().Header().Set(
		echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(map[string][]MecabNode{
		"data": h.Nodes,
	})
}

// utililies
func (h *Handler) ParseToNode(s string) error {
	tg, err := h.Mecab.NewTagger()
	if err != nil {
		return err
	}
	defer tg.Destroy()
	lt, err := h.Mecab.NewLattice(s)
	if err != nil {
		return err
	}
	defer lt.Destroy()

	var nodes []MecabNode
	node := tg.ParseToNode(lt)
	for {
		stat := node.Stat()
		if stat == int(MECAB_BOS_NODE) || stat == int(MECAB_EOS_NODE) {
			if node.Next() != nil {
				break
			}
		}

		features := node.Feature()
		if stat == int(MECAB_UNK_NODE) {
			features += strings.Repeat(",", 21)
		}

		csvReader := csv.NewReader(strings.NewReader(features))
		dec, err := csvutil.NewDecoder(csvReader, h.NodeHeader...)
		if err != nil {
			return err
		}
		var nf NodeFeatures
		if err := dec.Decode(&nf); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		nodes = append(nodes, MecabNode{
			ID:       node.Id(),
			Length:   node.Length(),
			Stat:     node.Stat(),
			StartPos: node.StartPos(),
			Surface:  node.Surface(),
			Features: nf,
		})
		if node.Next() != nil {
			break
		}
	}
	h.Nodes = nodes

	return nil
}

func (h *Handler) PrintNode() {
	for _, node := range h.Nodes {
		fmt.Println(node)
	}
}
