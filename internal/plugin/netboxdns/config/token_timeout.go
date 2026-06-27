package config

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](&tokensOnce, &tokens, tTimeoutName, &tTimeout{})
}

var tTimeoutName = "timeout"

type tTimeout struct{}

func (tTimeout) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return core.ErrNoTokenValue(c, tokens.PluginName, tTimeoutName)
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return core.ErrTokenParse(c, tokens.PluginName, tTimeoutName, err)
	}
	cfg.Timeout = d
	return nil
}

func (tTimeout) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
