package config

import (
	"crypto/tls"
	"net/url"
	"time"

	"github.com/coredns/coredns/plugin/pkg/fall"
)

// Config represents the configuration for the NetboxDNS plugin.
type Config struct {
	pluginName   string
	Fall         fall.F        `name:"fallthrough"`
	NetboxURL    *url.URL      `name:"url" required:"true"`
	NoOp         bool          `name:"noop"`
	Timeout      time.Duration `name:"timeout"`
	TLS          *tls.Config   `name:"tls"`
	Token        string        `name:"token" required:"true"`
	Views        []string      `name:"views"`
	ViewsExclude []string      `name:"views_exclude"`
	Zones        []string
}

// PluginName returns the name of the plugin. This is passed from the plugin
// setup function for use in logging.
func (c Config) PluginName() string {
	return c.pluginName
}
