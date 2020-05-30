package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
)

// Channel implements a Queue based on a go channel
type Channel struct {
	deadLetterHandler DeadLetterHandler
	resourceChannel   chan logging.Resource
}

// DeadLetterHandler defines dead letter handler function
type DeadLetterHandler func(msg logging.Resource) error

var _ Queue = &Channel{}

func NewChannelQueue(dlh DeadLetterHandler) (*Channel, error) {
	resourceChannel := make(chan logging.Resource, 50)

	return &Channel{
		resourceChannel:   resourceChannel,
		deadLetterHandler: dlh,
	}, nil
}

func (c Channel) Output() <-chan logging.Resource {
	return c.resourceChannel
}

func (c Channel) Push(raw []byte) error {
	resource, err := BodyToResource(raw)
	if err != nil {
		return err
	}
	c.resourceChannel <- *resource
	return nil
}

func (c Channel) Start() (chan bool, error) {
	d := make(chan bool)
	go func(done chan bool) {
		<-done
	}(d)
	return d, nil
}

func (c Channel) DeadLetter(msg logging.Resource) error {
	if c.deadLetterHandler != nil {
		return c.deadLetterHandler(msg)
	}
	return nil
}
