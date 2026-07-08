package config

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `timeout` sets the Netbox API client timeout duration.

func init() {
	config.RegisterToken(&tokensOnce, &Tokens, tTimeoutName, new(tTimeout))
}

const tTimeoutName = "timeout"

type tTimeout struct{}

func (tTimeout) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tTimeoutName)
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tTimeoutName, err)
	}
	cfg.Timeout = d
	return nil
}

func (tTimeout) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
