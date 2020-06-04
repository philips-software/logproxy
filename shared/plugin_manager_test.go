package shared_test

import (
	"testing"

	"github.com/philips-software/logproxy/shared"
	"github.com/stretchr/testify/assert"
)

func TestPluginManager(t *testing.T) {
	pluginManager := &shared.PluginManager{}
	err := pluginManager.Discover()
	assert.Nil(t, err)

	err = pluginManager.LoadAll()
	assert.Nil(t, err)

	assert.Equal(t, 0, len(pluginManager.Plugins()))
}
