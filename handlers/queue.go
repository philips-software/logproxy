package handlers

import (
	"github.com/philips-software/go-hsdp-api/logging"
)

type Queue interface {
	Start() (chan bool, error)
	Output() <-chan logging.Resource
	Push([]byte) error
}
