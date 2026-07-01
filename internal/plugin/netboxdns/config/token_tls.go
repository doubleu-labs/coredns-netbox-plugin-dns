package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/tls"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Token `tls` sets the Netbox API client TLS configuration.

func init() {
	core.RegisterToken[Config](&tokensOnce, &tokens, tTLSName, &tTLS{})
}

var tTLSName = "tls"

type tTLS struct{}

func (tTLS) Parse(c *caddy.Controller, cfg *Config) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return core.ErrNoTokenValue(c, tTLSName)
	}
	tc, err := tls.NewTLSConfigFromArgs(args...)
	if err != nil {
		return core.ErrTokenParse(c, tTLSName, err)
	}
	cfg.TLS = tc
	return nil
}

func (tTLS) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
