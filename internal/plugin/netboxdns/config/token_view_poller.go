package config

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

// Token `view_poller` enables the plugin to periodically validate that the
// configured views are still valid. This can prevent the plugin from returning
// SERVFAIL if a configured view is deleted.

func init() {
	config.RegisterToken(
		&tokensOnce,
		&Tokens,
		tViewPollerName,
		new(tViewPoller),
	)
}

const tViewPollerName = "view_poller"

type tViewPoller struct{}

func (tViewPoller) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		cfg.ViewPollerEnabled = true
		return nil
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tViewPollerName, err)
	}
	cfg.ViewPollerEnabled = true
	cfg.ViewPollerDuration = d
	return nil
}

func (tViewPoller) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
