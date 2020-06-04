package proto_test

import (
	"context"
	"log"
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/philips-software/logproxy/shared/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

type server struct {
}

func (s server) Filter(ctx context.Context, request *proto.FilterRequest) (*proto.FilterResponse, error) {
	return &proto.FilterResponse{
		Resource: request.Resource,
		Drop:     false,
		Modified: false,
		Error:    "",
	}, nil
}

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	proto.RegisterFilterServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestProtoResource(t *testing.T) {
	r := &proto.Resource{}

	msg, err := r.ToResource()
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, msg) {
		return
	}
	assert.Equal(t, "", r.GetApplicationInstance())
	assert.Equal(t, "", r.GetApplicationName())
	assert.Equal(t, "", r.GetApplicationVersion())
	assert.Equal(t, "", r.GetCategory())
	assert.Equal(t, "", r.GetComponent())
	assert.Nil(t, "", r.GetCustom())
	assert.Equal(t, "", r.GetEventId())
	assert.Equal(t, "", r.GetId())
	if !assert.NotNil(t, "", r.GetLogData()) {
		return
	}
	r.GetLogData().Reset()
	assert.Equal(t, "", r.GetLogTime())
	assert.Equal(t, "", r.GetOriginatingUser())
	assert.Equal(t, "", r.GetResourceType())
	assert.Equal(t, "", r.GetServerName())
	assert.Equal(t, "", r.GetServiceName())
	assert.Equal(t, "", r.GetSeverity())
	assert.Equal(t, "", r.GetTransactionId())

	assert.Equal(t, "", r.String())
	assert.NotNil(t, r.ProtoReflect())
}

func TestFilter(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	resource := &proto.Resource{
		LogData: &proto.LogData{
			Message: "foo",
		},
	}
	defer conn.Close()
	client := proto.NewFilterClient(conn)
	resp, err := client.Filter(ctx, &proto.FilterRequest{
		Resource: resource,
	})
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, resp) {
		return
	}
	assert.Equal(t, "", resource.String())
	x, y := resource.Descriptor()
	assert.NotNil(t, x)
	assert.Equal(t, 1, len(y))
	x, y = resp.Descriptor()
	assert.NotNil(t, x)
	assert.Equal(t, 1, len(y))
	// Test for output here.
}
