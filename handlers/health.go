package handlers

import (
	"github.com/labstack/echo/v4"
	"go.elastic.co/apm"
)

type HealthHandler struct {
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h HealthHandler) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, _ := apm.StartSpan(c.Request().Context(), "health", "handler")
		defer span.End()
		response := &healthResponse{
			Status: "UP",
		}
		return c.JSON(200, response)
	}
}
