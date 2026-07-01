package lookup

import (
	"context"
	"slices"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

func (l *Lookup) processOrigin(z *api.Zone) (
	*Response,
	error,
) {
	var qTypes []string
	switch l.QType {
	case dns.TypeSOA:
		qTypes = []string{"SOA", "NS"}
	case dns.TypeNS:
		qTypes = []string{"NS"}
	default:
		return nil, nil
	}

	recordQuery := &api.RecordQuery{
		Name: "@",
		Type: qTypes,
		Zone: z,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	records, err := recordQuery.GetRecords(ctx, l.Client)
	if err != nil {
		return nil, err
	}
	rrs, err := records.ToRRs()
	if err != nil {
		return nil, err
	}
	answerRRs := l.filterRRByType(rrs, dns.TypeSOA)
	nameserverRRs := l.filterRRByType(rrs, dns.TypeNS)
	extraRecords, err := l.processExtra(z, nameserverRRs)
	if err != nil {
		return nil, err
	}
	if len(extraRecords) == 0 {
		var processExtraErr error
		extraRecords, processExtraErr = l.processExtra(
			nil,
			nameserverRRs,
		)
		if processExtraErr != nil {
			return nil, processExtraErr
		}
	}
	extraRRs := make([]dns.RR, 0, len(extraRecords))
	for record := range slices.Values(extraRecords) {
		var extraRRErr error
		rr, extraRRErr := record.ToRR()
		if extraRRErr != nil {
			return nil, extraRRErr
		}
		extraRRs = append(extraRRs, rr)
	}
	if l.QType == dns.TypeNS {
		answerRRs = nameserverRRs
		nameserverRRs = nil
	}
	out := &Response{
		Answer: answerRRs,
		Ns:     nameserverRRs,
		Extra:  extraRRs,
	}
	return out, nil
}

func (l *Lookup) filterRRByType(rrs []dns.RR, rt uint16) []dns.RR {
	var out []dns.RR
	for rr := range slices.Values(rrs) {
		if rr.Header().Rrtype == rt {
			out = append(out, rr)
		}
	}
	return out
}

func (l *Lookup) processExtra(z *api.Zone, rrs []dns.RR) (
	api.Records,
	error,
) {
	var out []api.Record
	for rr := range slices.Values(rrs) {
		var fqdn string
		switch t := rr.(type) {
		case *dns.CNAME:
			fqdn = t.Target
		case *dns.MX:
			fqdn = t.Mx
		case *dns.NS:
			fqdn = t.Ns
		case *dns.SRV:
			fqdn = t.Target
		default:
			break
		}
		if len(fqdn) == 0 {
			continue
		}
		rq := &api.RecordQuery{
			FQDN: fqdn,
			Zone: z,
		}
		records, err := l.runExtraRecordsRequest(rq)
		if err != nil {
			return nil, err
		}
		out = slices.Concat(out, records)
	}
	return out, nil
}

func (l *Lookup) runExtraRecordsRequest(rq *api.RecordQuery) (
	api.Records,
	error,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	records, err := rq.GetRecords(ctx, l.Client)
	if err != nil {
		return nil, err
	}
	return records, nil
}
