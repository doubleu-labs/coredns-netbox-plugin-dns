package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Parse parses the plugin configuration.
func Parse(c *caddy.Controller, l log.P) (
	*Config,
	error,
) {
	out := new(Config)
	var i int
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++

		out.Zones = core.ParseZones(c)

		if err := core.ProcessTokens(
			c,
			l,
			tokens,
			out,
		); err != nil {
			return out, err
		}
	}

	if err := core.ValidateConfig(c, out); err != nil {
		return out, err
	}

	return out, nil
}
