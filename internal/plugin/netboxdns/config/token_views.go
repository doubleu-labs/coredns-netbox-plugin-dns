package config

import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tViewsName,
		&tViews{},
	)
}

var tViewsName = "views"

type tViews struct{}

func (tViews) Parse(c *caddy.Controller, cfg *Config) error {
	return nil
}

func (tViews) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
