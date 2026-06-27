package config

import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tFallthroughName,
		&tFallthrough{},
	)
}

var tFallthroughName = "fallthrough"

type tFallthrough struct{}

func (*tFallthrough) Parse(c *caddy.Controller, cfg *Config) error {
	cfg.Fall.SetZonesFromArgs(c.RemainingArgs())
	return nil
}

func (*tFallthrough) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
