package config

import (
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Token `views_exclude` sets the views that will be excluded from API queries.

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tViewsExcludeName,
		&tViewsExclude{},
	)
}

var tViewsExcludeName = "views_exclude"

type tViewsExclude struct{}

func (tViewsExclude) Parse(c *caddy.Controller, cfg *Config) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return core.ErrNoTokenValue(c, tViewsExcludeName)
	}
	if cfg.ViewsExclude != nil {
		cfg.ViewsExclude = append(cfg.ViewsExclude, args...)
	}
	cfg.ViewsExclude = args
	return nil
}

func (tViewsExclude) Validate(_ *caddy.Controller, _ log.P, cfg *Config) error {
	slices.Sort(cfg.ViewsExclude)
	cfg.ViewsExclude = slices.Compact(cfg.ViewsExclude)
	return nil
}
