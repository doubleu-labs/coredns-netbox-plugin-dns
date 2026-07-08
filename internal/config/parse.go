package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
)

// Parse parses the plugin configuration.
func Parse[T Config](
	c *caddy.Controller,
	l log.P,
	tokens TokenMap[T],
	cfg *T,
) error {
	var i int
	for c.Next() {
		if i > 0 {
			return plugin.ErrOnce
		}
		i++

		(*cfg).SetZones(parseZones(c))

		if tokenErr := tokens.process(c, l, cfg); tokenErr != nil {
			return tokenErr
		}
	}

	if validationErr := validate(c, cfg); validationErr != nil {
		return validationErr
	}

	return nil
}
