package queue

import (
	"fmt"
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/philips-software/logproxy/handlers"
)

type Channel struct {
	resourceChannel chan logging.Resource
}

func NewChannelQueue() (*Channel, error) {
	resourceChannel := make(chan logging.Resource, 50)

	return &Channel{
		resourceChannel: resourceChannel,
	}, nil
}

func (c Channel) Output() <-chan logging.Resource {
	return c.resourceChannel
}

func (c Channel) Push(raw []byte) {
	resource, err := handlers.BodyToResource(raw)
	if err != nil {
		fmt.Printf("Dropped 1 message")
		return
	}
	c.resourceChannel <- *resource
}

func (c Channel) Start() (chan bool, error) {
	d := make(chan bool)
	return d, nil
}
