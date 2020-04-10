package main

import (
	"context"
	"github.com/labstack/echo"
	"os"
	"testing"

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

func TestRealMain(t *testing.T) {
	echoChan := make(chan *echo.Echo, 1)
	quitChan := make(chan int, 1)

	sharedKey := os.Getenv("HSDP_LOGINGESTOR_KEY")
	sharedSecret := os.Getenv("HSDP_LOGINGESTOR_SECRET")
	baseURL := os.Getenv("HSDP_LOGINGESTOR_URL")
	productKey := os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")
	token := os.Getenv("TOKEN")
	port := os.Getenv("PORT")

	defer func() {
		os.Setenv("HSDP_LOGINGESTOR_KEY", sharedKey)
		os.Setenv("HSDP_LOGINGESTOR_SECRET", sharedSecret)
		os.Setenv("HSDP_LOGINGESTOR_URL", baseURL)
		os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", productKey)
		os.Setenv("TOKEN", token)
		os.Setenv("PORT", port)
	}()
	os.Setenv("HSDP_LOGINGESTOR_KEY", "foo")
	os.Setenv("HSDP_LOGINGESTOR_SECRET", "bar")
	os.Setenv("HSDP_LOGINGESTOR_URL", "http://localhost")
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", "key")
	os.Setenv("TOKEN", "token")
	os.Setenv("PORT", "0")

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	exitCode := 255
	select {
	case e := <-echoChan:
		err := e.Shutdown(context.Background())
		assert.Nil(t, err)
		exitCode = 0
	case exitCode = <-quitChan:
	}
	assert.Equal(t, 0, exitCode)

}

func TestMissingToken(t *testing.T) {
	echoChan := make(chan *echo.Echo, 1)
	quitChan := make(chan int, 1)

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	exitCode := <-quitChan
	assert.Equal(t, 3, exitCode)

	os.Setenv("LOGPROXY_SYSLOG", "false") // Disable Syslog
	os.Setenv("LOGPROXY_IRONIO", "true") // Enable IronIO

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	exitCode = <-quitChan
	assert.Equal(t, 4, exitCode)
}

func TestNoEndpoints(t *testing.T) {
	echoChan := make(chan *echo.Echo, 1)
	quitChan := make(chan int, 1)

	os.Setenv("LOGPROXY_SYSLOG", "false") // Disable Syslog
	os.Setenv("LOGPROXY_IRONIO", "false") // Enable IronIO

	go func(e chan *echo.Echo, q chan int) {
		realMain(e, q)
	}(echoChan, quitChan)

	exitCode := <-quitChan
	assert.Equal(t, 1, exitCode)
}
