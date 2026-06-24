package netboxdns

import (
	"github.com/miekg/dns"
)

type lookupResult int

const (
	lookupSuccess lookupResult = iota
)

type lookupResponse struct {
	Answer       []dns.RR
	Ns           []dns.RR
	Extra        []dns.RR
	LookupResult lookupResult
}
