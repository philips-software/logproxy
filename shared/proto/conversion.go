package proto

import (
	"encoding/json"

	"github.com/philips-software/go-hsdp-api/logging"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
)

func FromResource(src logging.Resource) (*Resource, error) {
	msg := &Resource{
		LogData: &LogData{},
	}
	js, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	err = jsonpb.Unmarshal(js, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (x *Resource) ToResource() (*logging.Resource, error) {
	var dest logging.Resource

	js, err := jsonpb.Marshal(x)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(js, &dest)
	if err != nil {
		return nil, err
	}
	return &dest, nil
}
