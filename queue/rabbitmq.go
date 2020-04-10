package queue

import (
	"fmt"
	"github.com/loafoe/go-rabbitmq"
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/philips-software/logproxy/handlers"
	"github.com/streadway/amqp"
)

var (
	Exchange   = "logproxy"
	RoutingKey = "new.rfc5424"
)

type RabbitMQ struct {
	producer rabbitmq.Producer
	resourceChannel chan logging.Resource
}

func consumerTag() string {
	return "logproxy"
}

// RFC5424QueueName returns the queue name to use
func RFC5424QueueName() string {
	return "logproxy_rfc5424"
}

func setupProducer() (rabbitmq.Producer, error) {
	producer, err := rabbitmq.NewProducer(rabbitmq.Config{
		Exchange:     handlers.Exchange,
		ExchangeType: "topic",
		Durable:      false,
	})
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func NewRabbitMQQueue() (*RabbitMQ, error) {
	resourceChannel := make(chan logging.Resource)
	producer, err := setupProducer()

	return &RabbitMQ{
		producer: producer,
		resourceChannel: resourceChannel,
	}, err
}


func (r RabbitMQ)Output() <-chan logging.Resource {
	return r.resourceChannel
}

func (r RabbitMQ) Push(raw []byte) {
	err := r.producer.Publish(Exchange, RoutingKey, amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "application/octet-stream",
		ContentEncoding: "",
		Body:            raw,
		DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
		Priority:        0,              // 0-9
		// a bunch of application/implementation-specific fields
	})
	if err != nil {
		fmt.Printf("Error publishing: %v\n", err)
	}
}

func (r RabbitMQ) Start() (chan bool, error) {
	doneChannel := make(chan bool)
	// Consumer
	consumer, err := rabbitmq.NewConsumer(rabbitmq.Config{
		RoutingKey:   handlers.RoutingKey,
		Exchange:     handlers.Exchange,
		ExchangeType: "topic",
		Durable:      false,
		AutoDelete:   true,
		QueueName:    RFC5424QueueName(),
		CTag:         consumerTag(),
		HandlerFunc:  RabbitMQRFC5424Worker(r.resourceChannel, doneChannel),
	})
	if err != nil {
		return nil, err
	}
	if err := consumer.Start(); err != nil {
		return nil, err
	}
	return doneChannel, nil
}

func ackDelivery(d amqp.Delivery) {
	err := d.Ack(true)
	if err != nil {
		fmt.Printf("Error Acking delivery: %v\n", err)
	}
}

func RabbitMQRFC5424Worker(resourceChannel chan<- logging.Resource, done <-chan bool) rabbitmq.ConsumerHandlerFunc {
	return func(deliveries <-chan amqp.Delivery, doneChannel <-chan bool) {
		for {
			select {
			case d := <-deliveries:
				resource, err := handlers.BodyToResource(d.Body)
				ackDelivery(d)
				if err != nil {
					fmt.Printf("Error processing syslog message: %v\n", err)
					continue
				}
				resourceChannel <- *resource
			case <-doneChannel:
				fmt.Printf("Worker received done message (worker)...\n")
			case <-done:
				fmt.Printf("Worker received done message (master)...\n")
				return
			}
		}
	}
}