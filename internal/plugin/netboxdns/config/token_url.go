package config

import (
	"net/url"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tURLName,
		&tURL{},
	)
}

var tURLName = "url"

type tURL struct{}

func (tURL) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		return core.ErrNoTokenValue(c, tokens.PluginName, tURLName)
	}
	u, err := url.Parse(c.Val())
	if err != nil {
		return core.ErrTokenParse(c, tokens.PluginName, tURLName, err)
	}
	cfg.NetboxURL = api.JoinAPIPath(u)
	return nil
}

func (tURL) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
