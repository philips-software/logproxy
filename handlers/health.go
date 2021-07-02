package handlers

import (
	"fmt"

	"github.com/labstack/echo-contrib/zipkintracing"
	"github.com/labstack/echo/v4"
	"github.com/openzipkin/zipkin-go"
)

type HealthHandler struct {
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h HealthHandler) Handler(tracer *zipkin.Tracer) echo.HandlerFunc {
	return func(c echo.Context) error {
		if tracer != nil {
			span := zipkintracing.StartChildSpan(c, "dump", tracer)
			defer span.Finish()
			traceID := span.Context().TraceID.String()
			fmt.Printf("handler=health traceID=%s\n", traceID)
		}
		response := &healthResponse{
			Status: "UP",
		}
		return c.JSON(200, response)
	}
}
