package handlers

import (
	"github.com/labstack/echo"
)

type HealthHandler struct {
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h HealthHandler) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		response := &healthResponse{
			Status: "UP",
		}
		return c.JSON(200, response)
	}
}
