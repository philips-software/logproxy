package queue_test

import (
	"net/http"
	"testing"

	"github.com/philips-software/logproxy/queue"

	"github.com/dip-software/go-dip-api/logging"
	"github.com/stretchr/testify/assert"
)

const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging appName [APP/PROC/WEB/0] - - {"app":"appName","val":{"message":"bericht"},"ver":"1.0.0","evt":"eventID","sev":"info","cmp":"component","trns":"transactionID","usr":null,"srv":"serverName","service":"serviceName","usr":"foo","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"xxx","time":"2018-09-07T15:39:21Z"}`

type nilMetrics struct {
}

func (n *nilMetrics) IncPluginDropped() {
}

func (n *nilMetrics) IncPluginModified() {
}

var _ queue.Metrics = (*nilMetrics)(nil)

func (n *nilMetrics) IncEnhancedEncodedMessage() {
}

func (n *nilMetrics) IncProcessed() {
}

func (n *nilMetrics) IncEnhancedTransactionID() {
}

type nilLogger struct {
}

func (n *nilLogger) Debugf(_ string, _ ...interface{}) {
	// Don't log anything
}

type nilStorer struct {
}

var _ logging.Storer = &nilStorer{}

func (n *nilStorer) StoreResources(_ []logging.Resource, count int) (*logging.StoreResponse, error) {
	if count == 23 {
		return &logging.StoreResponse{
			Response: &http.Response{
				StatusCode: http.StatusBadRequest,
			},
			Failed: []logging.Resource{
				{},
				{},
			},
		}, logging.ErrBatchErrors
	}
	return &logging.StoreResponse{
		Response: &http.Response{
			StatusCode: http.StatusCreated,
		},
	}, nil
}

func TestChannelQueue(t *testing.T) {
	q, err := queue.NewChannelQueue(queue.WithMetrics(&nilMetrics{}))
	assert.Nil(t, err)
	assert.NotNil(t, q)

	resourceChannel := q.Output()
	quit, err := q.Start()
	assert.Nil(t, err)
	assert.NotNil(t, quit)

	phLogger, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, "testBuild", &nilMetrics{})
	assert.Nil(t, err)

	go phLogger.ResourceWorker(q, quit, nil)

	go func() {
		_ = q.Push([]byte(rawMessage))
	}()

	r := <-resourceChannel
	assert.NotNil(t, r)
	assert.Equal(t, "2018-09-07T15:39:21.132Z", r.LogTime)
}
