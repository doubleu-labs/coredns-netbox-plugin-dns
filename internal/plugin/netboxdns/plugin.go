package netboxdns

// noinspection LongLine
import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
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

	activeZoneStatus []string
	client           *api.Client
	fall             fall.F
	logger           log.P
	noop             bool
	serverState      *iplugin.ServerContext
	viewPoller       *poller.ViewPoller
	views            *core.Views
	zones            []string
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
	req := core.NewServeRequest(ctx, w, r)

	if result := n.handleNoOp(req); result.Handled {
		return result.Rcode, result.Err
	}

	if result := n.handleZoneTransfer(req); result.Handled {
		return result.Rcode, result.Err
	}

	if !core.MatchesZone(n.zones, req.QName()) {
		return core.ServeNextOrFailure(pluginName, n.Next, req)
	}

	if result := n.handleViewPollerDisabled(req); result.Handled {
		return result.Rcode, result.Err
	}

	response, result := n.runLookup(req)
	if result.Handled {
		return result.Rcode, result.Err
	}

	if nameErrorResult := n.handleNameError(
		req,
		response,
	); nameErrorResult.Handled {
		return nameErrorResult.Rcode, nameErrorResult.Err
	}

	return n.writeLookupResponse(req, response)
}

func (n *netboxDNS) handleNoOp(req core.ServeRequest) core.ServeResult {
	if !n.noop {
		return core.ServeResult{}
	}

	if n.fall.Through(req.QName()) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"`noop` enabled; forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.QType()],
					req.QName(),
				),
			),
		)
		rcode, err := core.ServeNextOrFailure(pluginName, n.Next, req)
		return core.ServeResult{Handled: true, Rcode: rcode, Err: err}
	}

	n.logger.Error(
		core.ScopedMessage(
			"serve",
			"`noop` enabled but `fallthrough` not configured",
		),
	)
	return core.ServeResult{Handled: true, Rcode: dns.RcodeServerFailure}
}

func (n *netboxDNS) handleZoneTransfer(req core.ServeRequest) core.ServeResult {
	if req.QType() == dns.TypeAXFR || req.QType() == dns.TypeIXFR {
		rcode, err := core.ServeNextOrFailure(pluginName, n.Next, req)
		return core.ServeResult{Handled: true, Rcode: rcode, Err: err}
	}
	return core.ServeResult{}
}

func (n *netboxDNS) handleViewPollerDisabled(
	req core.ServeRequest,
) core.ServeResult {
	if n.viewPoller == nil || n.viewPoller.CanResolve() {
		return core.ServeResult{}
	}

	if n.fall.Through(req.QName()) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"netbox resolution disabled by view poller; "+
						"forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.QType()],
					req.QName(),
				),
			),
		)
		rcode, err := core.ServeNextOrFailure(pluginName, n.Next, req)
		return core.ServeResult{Handled: true, Rcode: rcode, Err: err}
	}

	n.logger.Error(
		core.ScopedMessage(
			"serve",
			fmt.Sprintf(
				"netbox resolution disabled by view poller for [%s] %q; "+
					"fallthrough not enabled",
				dns.TypeToString[req.QType()],
				req.QName(),
			),
		),
	)
	return core.ServeResult{Handled: true, Rcode: dns.RcodeServerFailure}
}

func (n *netboxDNS) lookupViews() *core.Views {
	if n.viewPoller == nil {
		return n.views
	}
	return new(n.viewPoller.Views())
}

func (n *netboxDNS) runLookup(req core.ServeRequest) (
	*lookup.Response,
	core.ServeResult,
) {
	lookupViews := n.lookupViews()

	lookupConfig := &lookup.Lookup{
		ActiveZoneStatus: n.activeZoneStatus,
		Client:           n.client,
		Family:           req.Family(),
		Logger:           &n.logger,
		QName:            req.QName(),
		QType:            req.QType(),
		Views:            lookupViews,
	}

	response, err := lookupConfig.Run()
	if err != nil {
		return nil, core.ServeResult{
			Handled: true,
			Rcode:   dns.RcodeServerFailure,
			Err:     err,
		}
	}

	if response == nil {
		return nil, core.ServeResult{
			Handled: true,
			Rcode:   dns.RcodeServerFailure,
		}
	}

	return response, core.ServeResult{}
}

func (n *netboxDNS) handleNameError(
	req core.ServeRequest,
	response *lookup.Response,
) core.ServeResult {
	if response.Result != lookup.NameError {
		return core.ServeResult{}
	}

	if n.fall.Through(req.QName()) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.QType()],
					req.QName(),
				),
			),
		)
		rcode, err := core.ServeNextOrFailure(pluginName, n.Next, req)
		return core.ServeResult{Handled: true, Rcode: rcode, Err: err}
	}

	n.logger.Debug(
		core.ScopedMessage(
			"serve",
			fmt.Sprintf(
				"no records for [%s] %q; fallthrough not enabled",
				dns.TypeToString[req.QType()],
				req.QName(),
			),
		),
	)

	return core.ServeResult{}
}

func (*netboxDNS) writeLookupResponse(
	req core.ServeRequest,
	response *lookup.Response,
) (int, error) {
	responseMsg := &dns.Msg{
		Answer: response.Answer,
		Ns:     response.Ns,
		Extra:  response.Extra,
	}
	responseMsg.SetReply(req.Msg())
	responseMsg.Authoritative = true

	switch response.Result {
	case lookup.Success:
		responseMsg.Rcode = dns.RcodeSuccess
	case lookup.NameError:
		responseMsg.Rcode = dns.RcodeNameError
	case lookup.Delegation:
		responseMsg.Authoritative = false
	}

	if err := req.ResponseWriter().WriteMsg(responseMsg); err != nil {
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}
