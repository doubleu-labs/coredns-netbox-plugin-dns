package netboxdns

// noinspection LongLine
import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	iplugin "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/lookup"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/poller"
	"github.com/miekg/dns"
)

const pluginName = "netboxdns"

type netboxDNS struct {
	Next plugin.Handler

	client      *api.Client
	fall        fall.F
	logger      log.P
	noop        bool
	serverState *iplugin.ServerContext
	viewPoller  *poller.ViewPoller
	views       *core.Views
	zones       []string
}

// Name implements the plugin.Handler interface.
func (*netboxDNS) Name() string {
	return pluginName
}

// ServeDNS implements the plugin.Handler interface.
func (n *netboxDNS) ServeDNS(
	ctx context.Context,
	w dns.ResponseWriter,
	r *dns.Msg,
) (int, error) {
	s := request.Request{W: w, Req: r}
	qName := s.QName()
	family := s.Family()
	qType := s.QType()

	if n.noop {
		if n.fall.Through(qName) {
			n.logger.Debug(
				core.ScopedMessage(
					"serve",
					fmt.Sprintf(
						"`noop` enabled; forwarding request [%s] %q to "+
							"next plugin",
						dns.TypeToString[qType],
						qName,
					),
				),
			)
			return plugin.NextOrFailure(pluginName, n.Next, ctx, w, r)
		}
		n.logger.Error(
			core.ScopedMessage(
				"serve",
				"`noop` enabled but `fallthrough` not configured",
			),
		)
		return dns.RcodeServerFailure, nil
	}

	zone := plugin.Zones(n.zones).Matches(qName)
	if zone == "" {
		return plugin.NextOrFailure(pluginName, n.Next, ctx, w, r)
	}

	lookupConfig := &lookup.Lookup{
		Client: n.client,
		Family: family,
		Logger: &n.logger,
		QName:  qName,
		QType:  qType,
		Views:  n.views,
	}
	response, err := lookupConfig.Run()
	if err != nil {
		return dns.RcodeServerFailure, err
	}
	if response == nil {
		return dns.RcodeServerFailure, nil
	}
	if response.Result == lookup.NameError {
		if n.fall.Through(qName) {
			n.logger.Debug(
				core.ScopedMessage(
					"serve",
					fmt.Sprintf(
						"forwarding request [%s] %q to next plugin",
						dns.TypeToString[qType],
						qName,
					),
				),
			)
			return plugin.NextOrFailure(pluginName, n.Next, ctx, w, r)
		}
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"no records for [%s] %q; fallthrough not enabled",
					dns.TypeToString[qType],
					qName,
				),
			),
		)
	}

	responseMsg := &dns.Msg{
		Answer: response.Answer,
		Ns:     response.Ns,
		Extra:  response.Extra,
	}
	responseMsg.SetReply(r)
	responseMsg.Authoritative = true

	switch response.Result {
	case lookup.Success:
		responseMsg.Rcode = dns.RcodeSuccess
	case lookup.NameError:
		responseMsg.Rcode = dns.RcodeNameError
	case lookup.Delegation:
		responseMsg.Authoritative = false
	}

	if writeMsgErr := w.WriteMsg(responseMsg); writeMsgErr != nil {
		return dns.RcodeServerFailure, writeMsgErr
	}
	return dns.RcodeSuccess, nil
}
