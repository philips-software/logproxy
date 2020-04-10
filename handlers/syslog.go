package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
)

var (
	Exchange   = "logproxy"
	RoutingKey = "new.rfc5424"
)

type SyslogHandler struct {
	queue Queue
	debug bool
	token string
}

func NewSyslogHandler(token string, pusher Queue) (*SyslogHandler, error) {
	if token == "" {
		return nil, fmt.Errorf("Missing TOKEN value")
	}
	handler := &SyslogHandler{}
	handler.token = token
	handler.queue = pusher

	if os.Getenv("DEBUG") == "true" {
		handler.debug = true
	}
	return handler, nil
}

func (h *SyslogHandler) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		t := c.Param("token")
		if h.token != t {
			return c.String(http.StatusUnauthorized, "")
		}
		b, _ := ioutil.ReadAll(c.Request().Body)
		go h.queue.Push(b)
		return c.String(http.StatusOK, "")
	}
}
