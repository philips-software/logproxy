package shared_test

import (
	"log"
	"os"
	"testing"

	"github.com/philips-software/logproxy/shared"
	"github.com/stretchr/testify/assert"
)

func TestPluginManager(t *testing.T) {
	cwd, err := os.Getwd()
	if !assert.Nil(t, err) {
		return
	}
	file, err := os.CreateTemp(cwd, "logproxy-filter-testrun")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.Remove(file.Name())
	}()

	pluginManager := &shared.PluginManager{}
	pluginManager.PluginDirs = append(pluginManager.PluginDirs, cwd)

	err = pluginManager.Discover()
	if !assert.Nil(t, err) {
		return
	}
	err = pluginManager.LoadAll()
	assert.NotNil(t, err)

	assert.Equal(t, 1, len(pluginManager.Plugins()))
}
