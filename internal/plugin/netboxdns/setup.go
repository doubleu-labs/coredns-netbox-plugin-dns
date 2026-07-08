package netboxdns

// noinspection LongLine
import (
	"context"
	"fmt"
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	iplugin "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/config"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/poller"
)

var (
	isSetupTest    bool
	isTestRegister bool
	isTransferTest bool
)

const defaultCacheHistory = 16

func SetTestRegister() {
	isTestRegister = true
}

func Register() {
	if isTestRegister {
		core.RegisterPlugin(pluginName, setup).After("transfer")
		return
	}
	core.RegisterPlugin(pluginName, setup)
}

func setup(c *caddy.Controller) error {
	logger := log.NewWithPlugin(pluginName)

	cfg, err := config.Parse(c, logger)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	ctx := getServerContext(c, cfg)

	views := &core.Views{
		Include: cfg.Views,
		Exclude: cfg.ViewsExclude,
	}

	netboxdns := &netboxDNS{
		activeZoneStatus: cfg.ActiveZoneStatus,
		client:           ctx.APIClient,
		logger:           logger,
		noop:             cfg.NoOp,
		serverState:      ctx,
		views:            views,
		zones:            cfg.Zones,
	}

	if vpErr := setupViewPoller(c, netboxdns, cfg); vpErr != nil {
		return vpErr
	}

	if cfg.Fall != nil {
		netboxdns.fall = *cfg.Fall
	}

	if cacheErr := setupCache(c, netboxdns, cfg); cacheErr != nil {
		return cacheErr
	}

	dnsserver.GetConfig(c).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	return nil
}

func getServerContext(
	c *caddy.Controller,
	cfg *config.Config,
) *iplugin.ServerContext {
	var ctx *iplugin.ServerContext
	if value, ok := c.Get(iplugin.ContextKey).(*iplugin.ServerContext); ok {
		ctx = value
	} else {
		ctx = &iplugin.ServerContext{
			APIClient: api.NewClient(cfg.Token, cfg.NetboxURL),
		}
		if cfg.Timeout != 0 {
			ctx.APIClient.SetTimeout(cfg.Timeout)
		}
		if cfg.TLS != nil {
			ctx.APIClient.SetTLSConfig(cfg.TLS)
		}
		ctx.APIClient.SetUserAgent(
			fmt.Sprintf("coredns.plugin.%s", pluginName),
		)
	}
	return ctx
}

func setupViewPoller(
	c *caddy.Controller,
	n *netboxDNS,
	cfg *config.Config,
) error {
	if !cfg.ViewPollerEnabled ||
		(len(n.views.Include) == 0 && len(n.views.Exclude) == 0) {
		return nil
	}
	viewPoller, err := poller.NewViewPoller(
		n.serverState.APIClient,
		n.views,
		cfg.ViewPollerDuration,
	)
	if err != nil {
		return err
	}
	n.viewPoller = viewPoller

	c.OnStartup(
		func() error {
			n.viewPoller.Start()
			return nil
		},
	)

	c.OnShutdown(
		func() error {
			n.viewPoller.Stop()
			return nil
		},
	)

	return nil
}

func notifyCachedZones(n *netboxDNS) {
	for zone := range slices.Values(n.zoneCache.GetZoneNames()) {
		// ignore the error here since the transfer plugin will
		// log errors if notify fails.
		_ = n.xfer.Notify(zone)
	}
}

func setupCache(c *caddy.Controller, n *netboxDNS, cfg *config.Config) error {
	if !cfg.ZonePollerEnabled {
		return nil
	}

	cacheHistory := defaultCacheHistory
	if cfg.CacheHistory > 0 {
		cacheHistory = cfg.CacheHistory
	}

	n.zoneCache = cache.NewCache(cacheHistory)

	c.OnStartup(
		func() error {
			t := dnsserver.GetConfig(c).Handler("transfer")
			if t == nil {
				return nil
			}
			n.xfer = t.(*transfer.Transfer)
			go notifyCachedZones(n)
			return nil
		},
	)

	c.OnRestartFailed(
		func() error {
			t := dnsserver.GetConfig(c).Handler("transfer")
			if t == nil {
				return nil
			}
			go notifyCachedZones(n)
			return nil
		},
	)

	zonePoller, zonePollerErr := poller.NewZonePoller(
		n.client,
		n.zoneCache.(*cache.Cache),
		n.views,
		n.activeZoneStatus,
		&n.logger,
		cfg.ZonePollerDuration,
	)
	if zonePollerErr != nil {
		return zonePollerErr
	}
	n.zonePoller = zonePoller

	if isTransferTest {
		if pollErr := n.zonePoller.PollFunc(context.Background()); pollErr != nil {
			return pollErr
		}
	}

	c.OnStartup(
		func() error {
			n.zonePoller.Start()
			return nil
		},
	)

	c.OnShutdown(
		func() error {
			n.zonePoller.Stop()
			return nil
		},
	)

	return nil
}
