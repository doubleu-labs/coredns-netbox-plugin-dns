package core

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

type ServeRequest struct {
	ctx    context.Context
	w      dns.ResponseWriter
	msg    *dns.Msg
	qName  string
	family int
	qType  uint16
}

func (sr *ServeRequest) QName() string {
	return sr.qName
}

func (sr *ServeRequest) QType() uint16 {
	return sr.qType
}

func (sr *ServeRequest) Family() int {
	return sr.family
}

func (sr *ServeRequest) Msg() *dns.Msg {
	return sr.msg
}

func (sr *ServeRequest) ResponseWriter() dns.ResponseWriter {
	return sr.w
}

func NewServeRequest(
	ctx context.Context,
	w dns.ResponseWriter,
	msg *dns.Msg,
) ServeRequest {
	s := request.Request{W: w, Req: msg}
	return ServeRequest{
		ctx:    ctx,
		w:      w,
		msg:    msg,
		qName:  s.Name(),
		family: s.Family(),
		qType:  s.QType(),
	}
}

type ServeResult struct {
	Handled bool
	Rcode   int
	Err     error
}

func ServeNextOrFailure(
	pluginName string,
	next plugin.Handler,
	req ServeRequest,
) (int, error) {
	return plugin.NextOrFailure(
		pluginName,
		next,
		req.ctx,
		req.w,
		req.msg,
	)
}
