package queue_test

import (
	"encoding/json"
	"testing"

	"github.com/philips-software/logproxy/queue"

	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

type mockProducer struct {
}

func (m mockProducer) Publish(exchange, routingKey string, msg amqp.Publishing) error {
	return nil
}

func (m mockProducer) Close() {
}

func TestRabbitMQQueue(t *testing.T) {
	q, err := queue.NewRabbitMQQueue(mockProducer{})
	assert.Nil(t, err)
	assert.NotNil(t, q)
	queue := q.Output()
	assert.NotNil(t, queue)
	c, err := q.Start()
	assert.NotNil(t, err)
	assert.Nil(t, c)
	err = q.Push([]byte(rawMessage))
	assert.Nil(t, err)
}

func TestFailedRabbitMQProducer(t *testing.T) {
	q, err := queue.NewRabbitMQQueue()
	assert.NotNil(t, err)
	assert.Nil(t, q)
}

func TestRabbitMQRFC5424Worker(t *testing.T) {
	quit := make(chan bool)
	quitWorker := make(chan bool)
	resourceChannel := make(chan logging.Resource, 1)
	worker := queue.RabbitMQRFC5424Worker(resourceChannel, quitWorker)
	assert.NotNil(t, worker)

	deliveryChan := make(chan amqp.Delivery)
	go worker(deliveryChan, quit)

	deliveryChan <- amqp.Delivery{Body: []byte(rawMessage)}
	delivery := <-resourceChannel
	assert.Equal(t, "2018-09-07T15:39:21.132Z", delivery.LogTime)
	quitWorker <- true
}

func TestRabbitMQResourceWorker(t *testing.T) {
	validResource := logging.Resource{
		ID:                  "deb545e2-ccea-4868-99fe-b9dfbf5ce56e",
		ResourceType:        "LogEvent",
		ServerName:          "foo.bar.com",
		ApplicationName:     "some-space",
		EventID:             "1",
		Category:            "Tracelog",
		Component:           "PHS",
		TransactionID:       "5bc4ce05-37b5-4f08-89e4-ed73790f8058",
		ServiceName:         "mcvs",
		ApplicationInstance: "85e597cb-2648-4187-78ec-2c58",
		ApplicationVersion:  "0.0.0",
		OriginatingUser:     "ActiveUser",
		LogTime:             "2017-10-15T01:53:20Z",
		Severity:            "INFO",
		LogData: logging.LogData{
			Message: "aGVsbG8gd29ybGQK",
		},
	}
	quit := make(chan bool)
	quitWorker := make(chan bool)
	resourceChannel := make(chan logging.Resource, 1)
	worker := queue.RabbitMQResourceWorker(resourceChannel, quitWorker)
	assert.NotNil(t, worker)

	deliveryChan := make(chan amqp.Delivery)
	go worker(deliveryChan, quit)

	js, err := json.Marshal(validResource)
	if !assert.Nil(t, err) {
		return
	}
	deliveryChan <- amqp.Delivery{Body: js}
	delivery := <-resourceChannel
	assert.Equal(t, "2017-10-15T01:53:20Z", delivery.LogTime)
	quitWorker <- true
}
