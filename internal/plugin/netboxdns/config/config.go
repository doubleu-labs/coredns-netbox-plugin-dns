package config

import (
	"crypto/tls"
	"net/url"
	"time"

	"github.com/coredns/coredns/plugin/pkg/fall"
)

// Config represents the configuration for the NetboxDNS plugin.
type Config struct {
	ActiveZoneStatus   []string      `name:"active_zone_status"`
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
	Zones              []string
}
