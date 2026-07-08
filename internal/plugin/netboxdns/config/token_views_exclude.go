package config

import (
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `views_exclude` sets the views that will be excluded from API queries.

func init() {
	config.RegisterToken(
		&tokensOnce,
		&Tokens,
		tViewsExcludeName,
		new(tViewsExclude),
	)
}

const tViewsExcludeName = "views_exclude"

type tViewsExclude struct{}

func (tViewsExclude) Parse(c *caddy.Controller, cfg *Config) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return config.ErrNoTokenValue(c, tViewsExcludeName)
	}
	if len(cfg.ViewsExclude) != 0 {
		cfg.ViewsExclude = append(cfg.ViewsExclude, args...)
		return nil
	}
	cfg.ViewsExclude = args
	return nil
}

func (tViewsExclude) Validate(_ *caddy.Controller, _ log.P, cfg *Config) error {
	slices.Sort(cfg.ViewsExclude)
	cfg.ViewsExclude = slices.Compact(cfg.ViewsExclude)
	return nil
}
