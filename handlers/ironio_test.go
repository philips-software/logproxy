package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/philips-software/logproxy/handlers"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestIronToRFC5424(t *testing.T) {
	testPayload := "severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message"
	now := time.Unix(0, 1580244137839197000).UTC()

	rfc := handlers.IronToRFC5424(now, testPayload)
	assert.Equal(t, "<14>1 2020-01-28T20:42:17.839Z 5e20da41d748ad000ace7654 loafoe/iron-test 5e299d0af210cc00097e9883 - - This is a message", rfc)

	rfc = handlers.IronToRFC5424(now, "malformed")
	assert.Equal(t, "<14>1 2020-01-28T20:42:17.839Z - - - - - nomatch: malformed", rfc)

}

func TestIronIOHandler(t *testing.T) {
	e, teardown := setup(t)
	defer teardown()

	body := bytes.NewBufferString("severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message")

	req := httptest.NewRequest(echo.POST, "/ironio/drain/t0ken", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIronInvalidToken(t *testing.T) {
	os.Setenv("DEBUG", "true")

	e, teardown := setup(t)
	defer teardown()

	req := httptest.NewRequest(echo.POST, "/ironio/drain/t00ken", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
