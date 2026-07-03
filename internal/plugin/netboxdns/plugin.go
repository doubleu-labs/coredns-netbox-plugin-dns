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

type serveRequest struct {
	ctx    context.Context
	w      dns.ResponseWriter
	msg    *dns.Msg
	qName  string
	family int
	qType  uint16
}

type serveResult struct {
	handled bool
	rcode   int
	err     error
}

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
	req := n.newServeRequest(ctx, w, r)

	if result := n.handleNoOp(req); result.handled {
		return result.rcode, result.err
	}

	if !n.matchesZone(req.qName) {
		return n.nextOrFailure(req)
	}

	if result := n.handleViewPollerDisabled(req); result.handled {
		return result.rcode, result.err
	}

	response, result := n.runLookup(req)
	if result.handled {
		return result.rcode, result.err
	}

	if nameErrorResult := n.handleNameError(
		req,
		response,
	); nameErrorResult.handled {
		return nameErrorResult.rcode, nameErrorResult.err
	}

	return n.writeLookupResponse(req, response)
}

func (n *netboxDNS) newServeRequest(
	ctx context.Context,
	w dns.ResponseWriter,
	msg *dns.Msg,
) serveRequest {
	s := request.Request{W: w, Req: msg}
	return serveRequest{
		ctx:    ctx,
		w:      w,
		msg:    msg,
		qName:  s.QName(),
		family: s.Family(),
		qType:  s.QType(),
	}
}

func (n *netboxDNS) handleNoOp(req serveRequest) serveResult {
	if !n.noop {
		return serveResult{}
	}

	if n.fall.Through(req.qName) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"`noop` enabled; forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.qType],
					req.qName,
				),
			),
		)
		rcode, err := n.nextOrFailure(req)
		return serveResult{handled: true, rcode: rcode, err: err}
	}

	n.logger.Error(
		core.ScopedMessage(
			"serve",
			"`noop` enabled but `fallthrough` not configured",
		),
	)
	return serveResult{handled: true, rcode: dns.RcodeServerFailure}
}

func (n *netboxDNS) matchesZone(qName string) bool {
	return plugin.Zones(n.zones).Matches(qName) != ""
}

func (n *netboxDNS) handleViewPollerDisabled(req serveRequest) serveResult {
	if n.viewPoller == nil || n.viewPoller.CanResolve() {
		return serveResult{}
	}

	if n.fall.Through(req.qName) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"netbox resolution disabled by view poller; "+
						"forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.qType],
					req.qName,
				),
			),
		)
		rcode, err := n.nextOrFailure(req)
		return serveResult{handled: true, rcode: rcode, err: err}
	}

	n.logger.Error(
		core.ScopedMessage(
			"serve",
			fmt.Sprintf(
				"netbox resolution disabled by view poller for [%s] %q; "+
					"fallthrough not enabled",
				dns.TypeToString[req.qType],
				req.qName,
			),
		),
	)
	return serveResult{handled: true, rcode: dns.RcodeServerFailure}
}

func (n *netboxDNS) lookupViews() *core.Views {
	if n.viewPoller == nil {
		return n.views
	}
	return new(n.viewPoller.Views())
}

func (n *netboxDNS) runLookup(req serveRequest) (
	*lookup.Response,
	serveResult,
) {
	lookupViews := n.lookupViews()

	lookupConfig := &lookup.Lookup{
		Client: n.client,
		Family: req.family,
		Logger: &n.logger,
		QName:  req.qName,
		QType:  req.qType,
		Views:  lookupViews,
	}

	response, err := lookupConfig.Run()
	if err != nil {
		return nil, serveResult{
			handled: true,
			rcode:   dns.RcodeServerFailure,
			err:     err,
		}
	}

	if response == nil {
		return nil, serveResult{
			handled: true,
			rcode:   dns.RcodeServerFailure,
		}
	}

	return response, serveResult{}
}

func (n *netboxDNS) handleNameError(
	req serveRequest,
	response *lookup.Response,
) serveResult {
	if response.Result != lookup.NameError {
		return serveResult{}
	}

	if n.fall.Through(req.qName) {
		n.logger.Debug(
			core.ScopedMessage(
				"serve",
				fmt.Sprintf(
					"forwarding request [%s] %q to next plugin",
					dns.TypeToString[req.qType],
					req.qName,
				),
			),
		)
		rcode, err := n.nextOrFailure(req)
		return serveResult{handled: true, rcode: rcode, err: err}
	}

	n.logger.Debug(
		core.ScopedMessage(
			"serve",
			fmt.Sprintf(
				"no records for [%s] %q; fallthrough not enabled",
				dns.TypeToString[req.qType],
				req.qName,
			),
		),
	)

	return serveResult{}
}

func (*netboxDNS) writeLookupResponse(
	req serveRequest,
	response *lookup.Response,
) (int, error) {
	responseMsg := &dns.Msg{
		Answer: response.Answer,
		Ns:     response.Ns,
		Extra:  response.Extra,
	}
	responseMsg.SetReply(req.msg)
	responseMsg.Authoritative = true

	switch response.Result {
	case lookup.Success:
		responseMsg.Rcode = dns.RcodeSuccess
	case lookup.NameError:
		responseMsg.Rcode = dns.RcodeNameError
	case lookup.Delegation:
		responseMsg.Authoritative = false
	}

	if err := req.w.WriteMsg(responseMsg); err != nil {
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}

func (n *netboxDNS) nextOrFailure(req serveRequest) (int, error) {
	return plugin.NextOrFailure(
		pluginName,
		n.Next,
		req.ctx,
		req.w,
		req.msg,
	)
}
