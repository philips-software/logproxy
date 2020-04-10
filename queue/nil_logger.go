package queue

import "github.com/philips-software/go-hsdp-api/logging"

type nilLogger struct {
}

func (n *nilLogger) Debugf(format string, args ...interface{}) {
	// Don't log anything
}

type nilStorer struct {
}

func (n *nilStorer) StoreResources(msgs []logging.Resource, count int) (*logging.Response, error) {
	return &logging.Response{}, nil
}
