package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// main
func main() {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	h, err := NewHandler(cfg.Mecab.Dicts)
	if err != nil {
		panic(err)
	}
	defer h.Mecab.Destroy()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.HTTPErrorHandler = ErrorHandler

	api := e.Group("/api")

	mecabApi := api.Group("/mecab")

	mecabApi.POST("/convert", h.PostMecabConvert)

	e.Logger.Fatal(e.Start(":" + cfg.Server.Port))
}

// error handler
type Error struct {
	Code    int    `json:"code"`
	Key     string `json:"error"`
	Message string `json:"message"`
}

func newHTTPError(code int, key string, msg string) *Error {
	return &Error{
		Code:    code,
		Key:     key,
		Message: msg,
	}
}

func (e *Error) Error() string {
	return e.Key + ": " + e.Message
}

func ErrorHandler(err error, c echo.Context) {
	var (
		code = http.StatusInternalServerError
		key  = "ServerError"
		msg  string
	)

	if e, ok := err.(*Error); ok {
		code = e.Code
		key = e.Key
		msg = e.Message
	} else {
		msg = http.StatusText(code)
	}

	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD {
			err := c.NoContent(code)
			if err != nil {
				c.Logger().Error(err)
			}
		} else {
			err := c.JSON(code, newHTTPError(code, key, msg))
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}
}

func badRequest(c echo.Context, key string, err error) *Error {
	return newHTTPError(http.StatusBadRequest, key, err.Error())
}
