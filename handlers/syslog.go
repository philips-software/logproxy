package handlers

import (
	"fmt"
	"github.com/philips-software/logproxy/queue"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
)

type SyslogHandler struct {
	pusher queue.Queue
	debug  bool
	token  string
}

func NewSyslogHandler(token string, pusher queue.Queue) (*SyslogHandler, error) {
	if token == "" {
		return nil, fmt.Errorf("Missing TOKEN value")
	}
	handler := &SyslogHandler{}
	handler.token = token
	handler.pusher = pusher

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
		go func() {
			_ = h.pusher.Push(b)
		}()
		return c.String(http.StatusOK, "")
	}
}
