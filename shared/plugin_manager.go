package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"golang.org/x/sync/semaphore"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
)

// Handshake is a common handshake that is shared by shared and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "6656191c-a4af-48a7-934f-ed05cd91dcc8",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"filter": &FilterGRPCPlugin{},
}

// PluginGlob is the glob pattern used to find plugins.
const PluginGlob = "logproxy-filter-*"

// Based on: https://github.com/hashicorp/otto/blob/v0.2.0/command/plugin_manager.go

// PluginManager is responsible for discovering and starting plugins.
type PluginManager struct {
	// PluginDirs are the directories where plugins can be found.
	// Any plugins with the same types found later (higher index) will
	// override earlier (lower index) directories.
	PluginDirs []string

	plugins []*Plugin
}

// Plugin is a single plugin that has been loaded.
type Plugin struct {
	// Path and Args are the method used to invocate this plugin.
	// These are the only two values that need to be set manually. Once
	// these are set, call Load to load the plugin.
	Path string   `json:"path,omitempty"`
	Args []string `json:"args"`
	App  Filter
}

// Load loads the plugin specified by the Path and instantiates the
// other fields on this structure.
func (p *Plugin) Load() error {
	// If it is builtin, then we always use our own path
	path := p.Path

	// Create the plugin client to communicate with the process
	pluginClient := plugin.NewClient(&plugin.ClientConfig{
		Cmd:              exec.Command(path, p.Args...),
		Managed:          true,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
	})

	// Request the client
	rpcClient, err := pluginClient.Client()
	if err != nil {
		return err
	}

	raw, err := rpcClient.Dispense("filter")
	if err != nil {
		return err
	}
	p.App = raw.(Filter)

	return nil
}

func (p *Plugin) String() string {
	path := p.Path

	return fmt.Sprintf("%s %v", path, p.Args)
}

// Plugins returns the loaded plugins.
func (m *PluginManager) Plugins() []*Plugin {
	return m.plugins
}

// Discover will find all the available plugin binaries. Each time this
// is called it will override any previously discovered plugins.
func (m *PluginManager) Discover() error {
	result := make([]*Plugin, 0, 20)

	for _, dir := range m.PluginDirs {
		log.Printf("[DEBUG] Looking for plugins in: %s", dir)
		paths, err := plugin.Discover(PluginGlob, dir)
		if err != nil {
			return fmt.Errorf(
				"Error discovering plugins in %s: %s", dir, err)
		}

		for _, path := range paths {
			result = append(result, &Plugin{
				Path: path,
			})
		}
	}

	// Reverse the list of plugins. We do this because we want custom
	// plugins to take priority over built-in plugins, and the PluginDirs
	// ordering also defines this priority.
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	// Log it
	for _, r := range result {
		log.Printf("[DEBUG] Detected plugin: %s", r)
	}

	// Save our result
	m.plugins = result

	return nil
}

// LoadAll will launch every plugin and add it to the CoreConfig given.
func (m *PluginManager) LoadAll() error {
	// If we've never loaded plugin paths, then let's discover those first
	if m.Plugins() == nil {
		if err := m.Discover(); err != nil {
			return err
		}
	}

	// Go through each plugin path and load single
	var merr error
	var merrLock sync.Mutex
	var wg sync.WaitGroup
	sema := semaphore.NewWeighted(1)
	for _, plugin := range m.Plugins() {
		wg.Add(1)
		go func(plugin *Plugin) {
			defer wg.Done()

			_ = sema.Acquire(context.Background(), 1)
			defer sema.Release(1)

			if err := plugin.Load(); err != nil {
				merrLock.Lock()
				defer merrLock.Unlock()
				merr = multierror.Append(merr, fmt.Errorf(
					"Error loading plugin %s: %s",
					plugin.Path, err))
			}
		}(plugin)
	}

	// Wait for all the plugins to load
	wg.Wait()

	return merr
}
