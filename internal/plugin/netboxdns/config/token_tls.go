package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/tls"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](&tokensOnce, &tokens, tTLSName, &tTLS{})
}

var tTLSName = "tls"

type tTLS struct{}

func (tTLS) Parse(c *caddy.Controller, cfg *Config) error {
	tc, err := tls.NewTLSConfigFromArgs(c.RemainingArgs()...)
	if err != nil {
		return core.ErrTokenParse(c, tokens.PluginName, tTLSName, err)
	}
	cfg.TLS = tc
	return nil
}

func (tTLS) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
