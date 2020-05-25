package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/philips-software/logproxy/handlers"
	"github.com/stretchr/testify/assert"
	"testing"
)

const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging appName [APP/PROC/WEB/0] - - {"app":"appName","val":{"message":"bericht"},"ver":"1.0.0","evt":"eventID","sev":"info","cmp":"component","trns":"transactionID","usr":null,"srv":"serverName","service":"serviceName","usr":"foo","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"xxx","time":"2018-09-07T15:39:21Z"}`

type nilLogger struct {
}

func (n *nilLogger) Debugf(format string, args ...interface{}) {
	// Don't log anything
}

type nilStorer struct {
}

var _ logging.Storer = &nilStorer{}

func (n *nilStorer) StoreResources(msgs []logging.Resource, count int) (*logging.StoreResponse, error) {
	return &logging.StoreResponse{}, nil
}

func TestChannelQueue(t *testing.T) {
	q, err := NewChannelQueue()
	assert.Nil(t, err)
	assert.NotNil(t, q)

	resourceChannel := q.Output()
	quit, err := q.Start()
	assert.Nil(t, err)
	assert.NotNil(t, quit)

	phLogger, err := handlers.NewDeliverer(&nilStorer{}, &nilLogger{}, "testBuild")
	assert.Nil(t, err)

	doneChan := make(chan bool)
	go phLogger.ResourceWorker(resourceChannel, doneChan)
	go func() {
		_ = q.Push([]byte(rawMessage))
	}()

	r := <-resourceChannel
	assert.NotNil(t, r)
	assert.Equal(t, "2018-09-07T15:39:21.132Z", r.LogTime)
}
