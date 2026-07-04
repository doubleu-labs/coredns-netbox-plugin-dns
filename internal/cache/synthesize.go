package cache

import (
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

func SynthesizeSOA(zone *api.Zone, mname string) *dns.SOA {
	if mname == "" {
		mname = dns.Fqdn(zone.SOAMName.Name)
	}
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(zone.Name),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    zone.SOATTL,
		},
		Ns:      mname,
		Mbox:    dns.Fqdn(zone.SOARName),
		Serial:  zone.SOASerial,
		Refresh: zone.SOARefresh,
		Retry:   zone.SOARetry,
		Expire:  zone.SOAExpire,
		Minttl:  zone.SOAMinimum,
	}
}
