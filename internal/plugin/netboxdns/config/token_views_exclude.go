package config

import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tViewsExcludeName,
		&tViewsExclude{},
	)
}

var tViewsExcludeName = "views_exclude"

type tViewsExclude struct{}

func (tViewsExclude) Parse(c *caddy.Controller, cfg *Config) error {
	return nil
}

func (tViewsExclude) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
