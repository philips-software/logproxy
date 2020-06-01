package proto

import (
	"testing"

	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/stretchr/testify/assert"
)

func TestFromResource(t *testing.T) {
	src := logging.Resource{
		ResourceType:    "LoggingResource",
		ID:              "foo",
		TransactionID:   "1234",
		ApplicationName: "app",
		LogData: logging.LogData{
			Message: "bar",
		},
		Custom: []byte(`{"key":"value"}`),
	}
	msg, err := FromResource(src)
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, msg) {
		return
	}
	assert.Equal(t, msg.Id, "foo")
	assert.Equal(t, msg.ApplicationName, "app")
	assert.Equal(t, msg.LogData.Message, "bar")
	assert.Equal(t, msg.TransactionId, "1234")
}
