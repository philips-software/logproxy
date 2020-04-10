package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockProducer struct {

}

func (m mockProducer) Publish(exchange, routingKey string, msg amqp.Publishing) error {
	return nil
}

func (m mockProducer) Close() {
}

func TestRabbitMQQueue(t *testing.T) {
	q, err := NewRabbitMQQueue(mockProducer{})
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
	q, err := NewRabbitMQQueue()
	assert.NotNil(t, err)
	assert.Nil(t, q)
}

func TestRabbitMQRFC5424Worker(t *testing.T) {
	quit := make(chan bool)
	quitWorker := make(chan bool)
	resourceChannel := make(chan logging.Resource, 1)
	worker := RabbitMQRFC5424Worker(resourceChannel,quitWorker)
	assert.NotNil(t, worker)

	deliveryChan := make(chan amqp.Delivery)
	go worker(deliveryChan, quit)

	deliveryChan <- amqp.Delivery{ Body: []byte(rawMessage)}
	delivery := <- resourceChannel
	assert.Equal(t, "2018-09-07T15:39:21.132Z", delivery.LogTime)
	quitWorker <- true
}
