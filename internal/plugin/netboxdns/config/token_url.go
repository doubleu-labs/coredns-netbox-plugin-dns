package config

import (
	"net/url"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `url` set the URL of the Netbox instance that will be queried by the
// API client. Required.

func init() {
	config.RegisterToken(&tokensOnce, &Tokens, tURLName, new(tURL))
}

const tURLName = "url"

type tURL struct{}

func (tURL) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tURLName)
	}
	u, err := url.Parse(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tURLName, err)
	}
	cfg.NetboxURL = api.JoinAPIPath(u)
	return nil
}

func (tURL) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
