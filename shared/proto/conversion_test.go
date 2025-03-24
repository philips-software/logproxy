package proto_test

import (
	"testing"

	"github.com/philips-software/logproxy/shared/proto"

	"github.com/dip-software/go-dip-api/logging"
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
	msg, err := proto.FromResource(src)
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
	to, err := msg.ToResource()
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, to) {
		return
	}
	assert.Equal(t, to.ID, "foo")
	assert.Equal(t, to.ApplicationName, "app")
	assert.Equal(t, to.LogData.Message, "bar")
	assert.Equal(t, to.TransactionID, "1234")
}
