package shared_test

import (
	"context"
	"testing"

	"github.com/philips-software/logproxy/shared/proto"

	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/stretchr/testify/assert"

	"github.com/philips-software/logproxy/shared"
)

type testImpl struct{}

func (t *testImpl) Filter(in logging.Resource) (out logging.Resource, drop bool, modified bool, err error) {
	return in, false, false, nil
}

func TestGRPCServer(t *testing.T) {
	resource := logging.Resource{}
	protoResource, _ := proto.FromResource(resource)
	impl := &testImpl{}
	var srv = &shared.FilterGRPCServer{
		Impl: impl,
	}
	assert.NotNil(t, srv)

	resp, err := srv.Filter(context.Background(), &proto.FilterRequest{
		Resource: protoResource,
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}
