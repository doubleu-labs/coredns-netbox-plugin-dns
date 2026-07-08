package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `token` sets the Netbox API client token. Required.

func init() {
	config.RegisterToken(&tokensOnce, &Tokens, tTokenName, new(tToken))
}

const tTokenName = "token"

type tToken struct{}

func (tToken) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tTokenName)
	}
	cfg.Token = c.Val()
	return nil
}

func (tToken) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
