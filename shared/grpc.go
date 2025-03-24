package shared

import (
	"context"

	"github.com/dip-software/go-dip-api/logging"
	"github.com/philips-software/logproxy/shared/proto"
)

type FilterGRPCClient struct{ client proto.FilterClient }

func (m *FilterGRPCClient) Filter(msg logging.Resource) (logging.Resource, bool, bool, error) {
	in, err := proto.FromResource(msg)
	if err != nil {
		return msg, false, false, err
	}
	resp, err := m.client.Filter(context.Background(), &proto.FilterRequest{
		Resource: in,
	})
	if err != nil {
		return msg, false, false, err
	}
	res, err := resp.Resource.ToResource()
	if err != nil {
		return msg, false, false, err
	}
	return *res, resp.Drop, resp.Modified, nil
}

// Here is the gRPC server that FilterGRPCClient talks to.
type FilterGRPCServer struct {
	proto.UnimplementedFilterServer
	// This is the real implementation
	Impl Filter
}

func (m *FilterGRPCServer) Filter(ctx context.Context, req *proto.FilterRequest) (*proto.FilterResponse, error) {
	msg, err := req.Resource.ToResource()
	if err != nil {
		return nil, err
	}
	newMsg, drop, modified, err := m.Impl.Filter(*msg)
	if err != nil {
		return nil, err
	}
	protoResource, err := proto.FromResource(newMsg)
	if err != nil {
		return nil, err
	}
	return &proto.FilterResponse{
		Resource: protoResource,
		Drop:     drop,
		Modified: modified,
		Error:    "",
	}, nil
}
