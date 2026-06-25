package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

func Parse(c *caddy.Controller) (*Config, error) {
	if err := ensureSingleInstance(c); err != nil {
		return nil, err
	}

	out := &Config{
		Zones: parseZones(c),
	}

	if err := tokens.parse(c, out); err != nil {
		return out, err
	}

	if err := tokens.validate(c, out); err != nil {
		return out, err
	}

	if err := out.validate(c); err != nil {
		return out, err
	}

	return out, nil
}

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

func parseZones(c *caddy.Controller) []string {
	z := plugin.OriginsFromArgsOrServerBlock(
		c.RemainingArgs(),
		c.ServerBlockKeys,
	)
	if len(z) == 0 {
		z = []string{"."}
	}
	return z
}
