package shared

import (
	"context"

	"github.com/philips-software/logproxy/shared/proto"

	"github.com/hashicorp/go-plugin"
	"github.com/philips-software/go-hsdp-api/logging"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by shared and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "logproxy",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"process": &ProcessGRPCPlugin{},
}

type Processor interface {
	Process(msg logging.Resource) error
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type ProcessGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Processor
}

func (p *ProcessGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterProcessorServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *ProcessGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewProcessorClient(c)}, nil
}
