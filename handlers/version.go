package handlers

import (
	"github.com/labstack/echo/v4"
)

type versionResponse struct {
	Version string `json:"version"`
}

func VersionHandler(version string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, &versionResponse{version})
	}
}
