package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestIronToRFC5424(t *testing.T) {
	testPayload := "severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message"
	now := time.Unix(1405544146, 0)

	rfc := ironToRFC5424(now, testPayload)

	assert.Equal(t, "<14>1 2014-07-16T22:55:46+02:00 - - - - - severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message", rfc)
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
