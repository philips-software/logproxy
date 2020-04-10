package handlers

import (
	"bytes"
	"github.com/philips-software/go-hsdp-api/logging"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"

	"github.com/stretchr/testify/assert"
)

type mockProducer struct {
	t *testing.T
	q chan logging.Resource
}

func (m *mockProducer) Push(body []byte)  {
}

func (m *mockProducer) Start() (chan bool, error) {
	d := make(chan bool)
	return d, nil
}

func (m *mockProducer) Output() <-chan logging.Resource {
	if m.q == nil {
		m.q = make(chan logging.Resource)
	}
	return m.q
}

func setup(t *testing.T) (*echo.Echo, func()) {
	e := echo.New()
	syslogHandler, err := NewSyslogHandler("t0ken", &mockProducer{t: t})
	assert.Nilf(t, err, "Expected NewSyslogHandler() to succeed")
	ironHandler, err := NewIronIOHandler("t0ken", &mockProducer{t: t})
	assert.Nilf(t, err, "Expected NewIronIOHandler() to succeed")

	e.POST("/syslog/drain/:token", syslogHandler.Handler())
	e.POST("/ironio/drain/:token", ironHandler.Handler())

	return e, func() {
		e.Close()
	}
}

func TestInvalidToken(t *testing.T) {
	e, teardown := setup(t)
	defer teardown()

	req := httptest.NewRequest(echo.POST, "/syslog/drain/t00ken", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSyslogHandler(t *testing.T) {
	e, teardown := setup(t)
	defer teardown()

	var payload = `Starting Application on 50676a99-dce0-418a-6b25-1e3d with PID 8 (/home/vcap/app/BOOT-INF/classes started by vcap in /home/vcap/app)`
	var appVersion = `1.0-f53a57a`
	var transactionID = `eea9f72c-09b6-4d56-905b-b518fc4dc5b7`

	var rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging 7215cbaa-464d-4856-967c-fd839b0ff7b2 [APP/PROC/WEB/0] - - {"app":"msa-eustaging","val":{"message":"` + payload + `"},"ver":"` + appVersion + `","evt":null,"sev":"INFO","cmp":"CPH","trns":"` + transactionID + `","usr":null,"srv":"msa-eustaging.eu-west.philips-healthsuite.com","service":"msa","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"Tracelog","time":"2018-09-07T15:39:21Z"}`
	body := bytes.NewBufferString(rawMessage)

	req := httptest.NewRequest(echo.POST, "/syslog/drain/t0ken", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
