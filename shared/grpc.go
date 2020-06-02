package shared

import (
	"context"

	"github.com/philips-software/go-hsdp-api/logging"

	"github.com/philips-software/logproxy/shared/proto"
)

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCClient struct{ client proto.ProcessorClient }

func (m *GRPCClient) Process(msg logging.Resource) error {
	in, err := proto.FromResource(msg)
	if err != nil {
		return err
	}
	_, err = m.client.Process(context.Background(), &proto.ProcessRequest{
		Resource: in,
	})
	return err
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl Processor
}

func (m *GRPCServer) Process(
	ctx context.Context,
	req *proto.ProcessRequest) (*proto.ProcessResponse, error) {
	msg, _ := req.Resource.ToResource()
	return &proto.ProcessResponse{}, m.Impl.Process(*msg)
}
