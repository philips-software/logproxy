package queue

import (
	"errors"
	"fmt"

	"github.com/loafoe/go-rabbitmq"
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/streadway/amqp"
)

var (
	Exchange           = "logproxy"
	RoutingKey         = "new.rfc5424"
	ErrInvalidProducer = errors.New("RabbitMQ producer is nil or invalid")
)

// RabbitMQ implements Queue backed by RabbitMQ
type RabbitMQ struct {
	producer        rabbitmq.Producer
	resourceChannel chan logging.Resource
	metrics         Metrics
}

func (r *RabbitMQ) SetMetrics(m Metrics) {
	r.metrics = m
}

var _ Queue = &RabbitMQ{}

func consumerTag() string {
	return "logproxy"
}

// RFC5424QueueName returns the queue name to use
func RFC5424QueueName() string {
	return "logproxy_rfc5424"
}

func setupProducer() (rabbitmq.Producer, error) {
	producer, err := rabbitmq.NewProducer(rabbitmq.Config{
		Exchange:     Exchange,
		ExchangeType: "topic",
		Durable:      false,
	})
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func NewRabbitMQQueue(p rabbitmq.Producer, opts ...OptionFunc) (*RabbitMQ, error) {
	var producer rabbitmq.Producer
	var err error
	resourceChannel := make(chan logging.Resource)
	if p != nil {
		producer = p
	} else {
		producer, err = setupProducer()
	}
	if err != nil {
		return nil, err
	}
	ch := &RabbitMQ{
		producer:        producer,
		resourceChannel: resourceChannel,
	}
	for _, o := range opts {
		if err := o(ch); err != nil {
			return nil, err
		}
	}
	return ch, nil
}

func (r *RabbitMQ) Output() <-chan logging.Resource {
	return r.resourceChannel
}

func (r *RabbitMQ) Push(raw []byte) error {
	if r.producer == nil {
		return ErrInvalidProducer
	}
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
		return err
	}
	if r.metrics != nil {
		r.metrics.IncProcessed()
	}
	return nil
}

func (r *RabbitMQ) Start() (chan bool, error) {
	doneChannel := make(chan bool)
	// Consumer
	consumer, err := rabbitmq.NewConsumer(rabbitmq.Config{
		RoutingKey:   RoutingKey,
		Exchange:     Exchange,
		ExchangeType: "topic",
		Durable:      false,
		AutoDelete:   true,
		QueueName:    RFC5424QueueName(),
		CTag:         consumerTag(),
		HandlerFunc:  RabbitMQRFC5424Worker(r.resourceChannel, doneChannel, r.metrics),
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

func RabbitMQRFC5424Worker(resourceChannel chan<- logging.Resource, done <-chan bool, m Metrics) rabbitmq.ConsumerHandlerFunc {
	return func(deliveries <-chan amqp.Delivery, doneChannel <-chan bool) {
		for {
			select {
			case d := <-deliveries:
				resource, err := BodyToResource(d.Body, m)
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

func (r *RabbitMQ) DeadLetter(_ logging.Resource) error {
	// TODO: implement
	return nil
}
