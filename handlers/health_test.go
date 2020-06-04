package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/philips-software/logproxy/handlers"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var statusJSON = `{"status":"UP"}`

func TestHealth(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	healthHandler := &handlers.HealthHandler{}
	handler := healthHandler.Handler()

	// Assertions
	if assert.NoError(t, handler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, statusJSON, rec.Body.String())
	}
}
