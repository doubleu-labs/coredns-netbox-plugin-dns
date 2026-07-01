package config

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Token `timeout` sets the Netbox API client timeout duration.

func init() {
	core.RegisterToken[Config](&tokensOnce, &tokens, tTimeoutName, &tTimeout{})
}

var tTimeoutName = "timeout"

type tTimeout struct{}

func (tTimeout) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return core.ErrNoTokenValue(c, tTimeoutName)
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return core.ErrTokenParse(c, tTimeoutName, err)
	}
	cfg.Timeout = d
	return nil
}

func (tTimeout) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
