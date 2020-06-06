package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/philips-software/logproxy/handlers"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/version", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	versionHandler := handlers.VersionHandler("0.0.0")

	// Assertions
	if assert.NoError(t, versionHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "{\"version\":\"0.0.0\"}\n", rec.Body.String())
	}
}
