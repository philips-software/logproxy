package shared_test

import (
	"context"
	"testing"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/philips-software/logproxy/shared/proto"

	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/stretchr/testify/assert"

	"github.com/philips-software/logproxy/shared"
)

type testImpl struct{}

func (t *testImpl) Filter(in logging.Resource) (out logging.Resource, drop bool, modified bool, err error) {
	return in, false, false, nil
}

func TestFilterGRPCClient(t *testing.T) {
	resource := logging.Resource{}
	impl := &testImpl{}
	broker := &plugin.GRPCBroker{}
	s := grpc.NewServer()

	p := &shared.FilterGRPCPlugin{
		Impl: impl,
	}
	err := p.GRPCServer(broker, s)
	assert.Nil(t, err)

	res, drop, modified, err := p.Impl.Filter(resource)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.False(t, drop)
	assert.False(t, modified)

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
