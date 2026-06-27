package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Parse parses the plugin configuration.
func Parse(c *caddy.Controller, pluginName string) (*Config, error) {
	out := &Config{
		pluginName: pluginName,
	}
	tokens.PluginName = pluginName

	if err := ensureSingleInstance(c); err != nil {
		return nil, err
	}

	out.Zones = core.ParseZones(c)

	if err := core.ProcessTokens(
		c,
		core.TokenMap[*Config]{},
		&out,
	); err != nil {
		return out, err
	}

	if err := core.ValidateConfig(c, out); err != nil {
		return out, err
	}

	return out, nil
}

// Only one instance of this plugin is allowed, so ensure another configuration
// block is not present in the server block.
func ensureSingleInstance(c *caddy.Controller) error {
	var i int
	for c.Next() {
		if i > 0 {
			return plugin.ErrOnce
		}
		i++
	}
	return nil
}
