package core

import (
	"slices"
	"sync"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

type NetboxPlugin struct {
	name      string
	setupFunc func(*caddy.Controller) error
	before    string
	after     string
}

var plugins []*NetboxPlugin
var pluginsOnce sync.Once

// RegisterPlugin registers a Netbox plugin with the orchestrator. Plugins are
// not registered until FinalizePlugins is called.
func RegisterPlugin(
	name string,
	fn func(controller *caddy.Controller) error,
) *NetboxPlugin {

	pluginsOnce.Do(
		func() {
			plugins = make([]*NetboxPlugin, 0)
		},
	)

	np := &NetboxPlugin{
		name:      name,
		setupFunc: fn,
	}
	plugins = append(plugins, np)

	return np
}

func (np *NetboxPlugin) After(plugin string) {
	np.after = plugin
}

func (np *NetboxPlugin) Before(plugin string) {
	np.before = plugin
}

// FinalizePlugins registers all Netbox plugins. The dnsserver.Directives list
// is then modified to respect the Before/After relationships specified by each
// plugin. If no Before/After relationships are specified, the order in
// `plugin.cfg` is respected. Since, typically, only the core `netboxdns` plugin
// is specified in `plugin.cfg`, sub-plugins will not be called unless a
// Before/After relationship is specified.
func FinalizePlugins() {
	for p := range slices.Values(plugins) {
		plugin.Register(p.name, p.setupFunc)
	}

	for p := range slices.Values(plugins) {
		if p.after == "" && p.before == "" {
			continue
		}
		if p.after != "" {
			directivesInsertAfter(p.after, p.name)
		}
		if p.before != "" {
			directivesInsertBefore(p.before, p.name)
		}
	}
}

func directivesInsertAfter(anchor, target string) {
	i := slices.Index(dnsserver.Directives, anchor)
	if i == -1 {
		return
	}
	dnsserver.Directives = slices.Insert(dnsserver.Directives, i+1, target)
}

func directivesInsertBefore(anchor, target string) {
	i := slices.Index(dnsserver.Directives, anchor)
	if i == -1 {
		return
	}
	dnsserver.Directives = slices.Insert(dnsserver.Directives, i, target)
}
