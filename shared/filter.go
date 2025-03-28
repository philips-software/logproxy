package shared

import (
	"context"

	"github.com/philips-software/logproxy/shared/proto"

	"github.com/dip-software/go-dip-api/logging"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Filter interface {
	Filter(in logging.Resource) (out logging.Resource, drop bool, modified bool, err error)
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type FilterGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Filter
}

func (p *FilterGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterFilterServer(s, &FilterGRPCServer{Impl: p.Impl})
	return nil
}

func (p *FilterGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &FilterGRPCClient{client: proto.NewFilterClient(c)}, nil
}
