package queue

import (
	"github.com/philips-software/go-hsdp-api/logging"
)

// Channel implements a Queue based on a go channel
type Channel struct {
	resourceChannel chan logging.Resource
	metrics         Metrics
}

func (c Channel) SetMetrics(m Metrics) {
	c.metrics = m
}

var _ Queue = &Channel{}

func NewChannelQueue(opts ...OptionFunc) (*Channel, error) {
	resourceChannel := make(chan logging.Resource, 50)
	ch := &Channel{
		resourceChannel: resourceChannel,
	}
	for _, o := range opts {
		if err := o(ch); err != nil {
			return nil, err
		}
	}
	return ch, nil
}

func (c Channel) Output() <-chan logging.Resource {
	return c.resourceChannel
}

func (c Channel) Push(raw []byte) error {
	resource, err := BodyToResource(raw, c.metrics)
	if err != nil {
		return err
	}
	if c.metrics != nil {
		c.metrics.IncProcessed()
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

func (c Channel) DeadLetter(_ logging.Resource) error {
	// TODO: implement
	return nil
}
