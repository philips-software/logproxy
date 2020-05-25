package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
)

// Channel implements a Queue based on a go channel
type Channel struct {
	resourceChannel chan logging.Resource
}

var _ Queue = &Channel{}

func NewChannelQueue() (*Channel, error) {
	resourceChannel := make(chan logging.Resource, 50)

	return &Channel{
		resourceChannel: resourceChannel,
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
		<- done
	}(d)
	return d, nil
}

func (c Channel) DeadLetter(msg logging.Resource) error {
	// TODO: implement
	return nil
}