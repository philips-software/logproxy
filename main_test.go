package main

import (
	"net/http"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	assert.Equal(t, buildVersion, "v1.1.0-deadbeaf")
}

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

func TestSetupPHLogger(t *testing.T) {
	logger := log.New()
	sharedKey := os.Getenv("HSDP_LOGINGESTOR_KEY")
	sharedSecret := os.Getenv("HSDP_LOGINGESTOR_SECRET")
	baseURL := os.Getenv("HSDP_LOGINGESTOR_URL")
	productKey := os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")

	os.Setenv("HSDP_LOGINGESTOR_KEY", sharedKey)
	os.Setenv("HSDP_LOGINGESTOR_SECRET", sharedSecret)
	os.Setenv("HSDP_LOGINGESTOR_URL", baseURL)
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", productKey)
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
