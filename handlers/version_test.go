package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/version", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	versionHandler := VersionHandler("0.0.0")

	// Assertions
	if assert.NoError(t, versionHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, `{"version":"0.0.0"}`, rec.Body.String())
	}
}
