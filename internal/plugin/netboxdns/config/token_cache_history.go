package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

func init() {
	config.RegisterToken(
		&tokensOnce,
		&Tokens,
		tCacheHistoryName,
		new(tCacheHistory),
	)
}

const tCacheHistoryName = "cache_history"

type tCacheHistory struct{}

func (tCacheHistory) Parse(c *caddy.Controller, cfg *Config) error {
	return nil
}

func (tCacheHistory) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
