package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRabbitMQQueue(t *testing.T) {
	q, err := NewRabbitMQQueue()
	// TODO figure out proper mocking
	assert.NotNil(t, err)
	assert.Equal(t, "dial error: Connector with id 'amqp' doesn't give a service with the type '*amqp.Connection'. (perhaps no services match the connector)", err.Error())
	assert.NotNil(t, q)
	queue := q.Output()
	assert.NotNil(t, queue)
	c, err := q.Start()
	assert.NotNil(t, err)
	assert.Nil(t, c)
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
