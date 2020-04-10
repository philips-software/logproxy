package main

import (
	"context"
	"github.com/labstack/echo"
	"net/http"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)


func TestListenString(t *testing.T) {
	port := os.Getenv("PORT")
	defer func() {
		os.Setenv("PORT", port)
	}()
	os.Setenv("PORT", "")
	s := listenString()
	assert.Equal(t, s, ":8080")
	os.Setenv("PORT", "1028")
	s = listenString()
	assert.Equal(t, s, ":1028")
}

func TestSetup(t *testing.T) {
	logger := log.New()
	sharedKey := os.Getenv("HSDP_LOGINGESTOR_KEY")
	sharedSecret := os.Getenv("HSDP_LOGINGESTOR_SECRET")
	baseURL := os.Getenv("HSDP_LOGINGESTOR_URL")
	productKey := os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")

	defer func() {
		os.Setenv("HSDP_LOGINGESTOR_KEY", sharedKey)
		os.Setenv("HSDP_LOGINGESTOR_SECRET", sharedSecret)
		os.Setenv("HSDP_LOGINGESTOR_URL", baseURL)
		os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", productKey)
	}()
	os.Setenv("HSDP_LOGINGESTOR_KEY", "foo")
	os.Setenv("HSDP_LOGINGESTOR_SECRET", "bar")
	os.Setenv("HSDP_LOGINGESTOR_URL", "http://localhost")
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", "key")

	phLogger, err := setupPHLogger(http.DefaultClient, logger, buildVersion)
	assert.Nilf(t, err, "Expected setupPHLogger() to succeed: %v", err)
	assert.NotNil(t, phLogger)
}

func TestRealMain(t *testing.T) {
	echoChan := make(chan *echo.Echo, 1)
	quitChan := make(chan int, 1)

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	var exitCode int
	select {
		case e := <-echoChan:
			e.Shutdown(context.Background())
		case exitCode = <-quitChan:
	}
	assert.Equal(t, 20, exitCode)

	sharedKey := os.Getenv("HSDP_LOGINGESTOR_KEY")
	sharedSecret := os.Getenv("HSDP_LOGINGESTOR_SECRET")
	baseURL := os.Getenv("HSDP_LOGINGESTOR_URL")
	productKey := os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")
	token := os.Getenv("TOKEN")

	defer func() {
		os.Setenv("HSDP_LOGINGESTOR_KEY", sharedKey)
		os.Setenv("HSDP_LOGINGESTOR_SECRET", sharedSecret)
		os.Setenv("HSDP_LOGINGESTOR_URL", baseURL)
		os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", productKey)
		os.Setenv("TOKEN", token)
	}()
	os.Setenv("HSDP_LOGINGESTOR_KEY", "foo")
	os.Setenv("HSDP_LOGINGESTOR_SECRET", "bar")
	os.Setenv("HSDP_LOGINGESTOR_URL", "http://localhost")
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", "key")
	os.Setenv("TOKEN", "token")

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	exitCode = 255
	select {
	case e := <-echoChan:
		e.Shutdown(context.Background())
		exitCode = 0
	case exitCode = <-quitChan:
	}
	assert.Equal(t, 0, exitCode)

}
