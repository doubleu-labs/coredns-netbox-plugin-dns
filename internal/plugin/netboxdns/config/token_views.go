package config

import (
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `views` sets the views that will be queried by the API client.

func init() {
	config.RegisterToken(&tokensOnce, &Tokens, tViewsName, new(tViews))
}

const tViewsName = "views"

type tViews struct{}

func (tViews) Parse(c *caddy.Controller, cfg *Config) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return config.ErrNoTokenValue(c, tViewsName)
	}
	if len(cfg.Views) != 0 {
		cfg.Views = append(cfg.Views, args...)
		return nil
	}
	cfg.Views = args
	return nil
}

func (tViews) Validate(_ *caddy.Controller, _ log.P, cfg *Config) error {
	slices.Sort(cfg.Views)
	cfg.Views = slices.Compact(cfg.Views)
	return nil
}
