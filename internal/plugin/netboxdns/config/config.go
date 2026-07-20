package config

import (
	"crypto/tls"
	"net/url"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/config"
)

var (
	Tokens     config.TokenMap[*Config]
	tokensOnce sync.Once
)

// Config implements config.Config.
type Config struct {
	ActiveZoneStatus   []string      `name:"active_zone_status"`
	CacheHistory       int           `name:"cache_history"`
	Fall               *fall.F       `name:"fallthrough"`
	NetboxURL          *url.URL      `name:"url" required:"true"`
	NoOp               bool          `name:"noop"`
	Timeout            time.Duration `name:"timeout"`
	TLS                *tls.Config   `name:"tls"`
	Token              string        `name:"token" required:"true"`
	Views              []string      `name:"views"`
	ViewsExclude       []string      `name:"views_exclude"`
	ViewPollerDuration time.Duration `name:"view_poller"`
	ViewPollerEnabled  bool
	ZonePollerDuration time.Duration `name:"zone_poller"`
	ZonePollerEnabled  bool
	Zones              []string
}

func (c *Config) SetZones(zones []string) {
	c.Zones = zones
}
