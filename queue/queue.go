package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
)

// Queue implements a queue mechanism. The queue can be
// backed by e.g. RabbitMQ or a simple Go channel. Both
// of these are provided as part of logproxy.
// Internally the queue is driven by the Deliverer which
// transforms the raw payload to a logging.Resource
// and than pushes it to HSDP logging infrastructure
type Queue interface {
	// Start initializes the and returns a stop channel
	Start() (chan bool, error)
	// Output should return a channel fed by the queue raw data
	Output() <-chan logging.Resource
	// Push should queue the raw payload
	Push([]byte) error
	// DeadLetter should store a rejected logging.Resource for later processing
	DeadLetter(msg logging.Resource) error
}
