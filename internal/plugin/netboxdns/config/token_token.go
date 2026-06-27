package config

import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](&tokensOnce, &tokens, tTokenName, &tToken{})
}

var tTokenName = "token"

type tToken struct{}

func (tToken) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return core.ErrNoTokenValue(c, tokens.PluginName, tTokenName)
	}
	cfg.Token = c.Val()
	return nil
}

func (tToken) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
