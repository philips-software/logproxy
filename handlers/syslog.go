package handlers

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/philips-software/logproxy/queue"

	"github.com/labstack/echo/v4"
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
	tracer := opentracing.GlobalTracer()

	return func(c echo.Context) error {
		span := tracer.StartSpan("syslog_handler")
		defer span.Finish()
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
