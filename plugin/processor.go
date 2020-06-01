package plugin

import (
	"github.com/hashicorp/go-plugin"
	"github.com/philips-software/go-hsdp-api/logging"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "logproxy",
}

type Processor interface {
	Process(msg logging.Resource) bool
}
