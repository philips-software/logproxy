package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo-contrib/zipkintracing"
	"github.com/openzipkin/zipkin-go"

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

func (h *SyslogHandler) Handler(tracer *zipkin.Tracer) echo.HandlerFunc {
	return func(c echo.Context) error {
		if tracer != nil {
			defer zipkintracing.TraceFunc(c, "syslog_handler", zipkintracing.DefaultSpanTags, tracer)()
		}
		t := c.Param("token")
		if h.token != t {
			return c.String(http.StatusUnauthorized, "")
		}
		b, _ := ioutil.ReadAll(c.Request().Body)
		go func() {
			if tracer != nil {
				span := zipkintracing.StartChildSpan(c, "push", tracer)
				defer span.Finish()
				traceID := span.Context().TraceID.String()
				fmt.Printf("handler=syslog traceID=%s\n", traceID)
			}
			_ = h.pusher.Push(b)
		}()
		return c.String(http.StatusOK, "")
	}
}
