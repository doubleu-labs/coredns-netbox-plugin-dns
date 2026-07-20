package cache

import (
	"slices"

	"github.com/miekg/dns"
)

func snapshotZoneCache(zc *zoneCache) *zoneCache {
	out := &zoneCache{
		currentSerial: zc.currentSerial,
		currentZone:   cloneRRs(zc.currentZone),
		deltas:        cloneDeltas(zc.deltas),
	}
	if zc.soa != nil {
		if soa, ok := dns.Copy(zc.soa).(*dns.SOA); ok {
			out.soa = soa
		}
	}
	return out
}

func cloneDeltas(deltas []zoneDelta) []zoneDelta {
	out := slices.Clone(deltas)
	for i := range out {
		out[i].deleted = cloneRRs(out[i].deleted)
		out[i].added = cloneRRs(out[i].added)
	}
	return out
}

func cloneRRs(rrs []dns.RR) []dns.RR {
	if rrs == nil {
		return nil
	}
	out := make([]dns.RR, len(rrs))
	for i, rr := range rrs {
		if rr == nil {
			continue
		}
		out[i] = dns.Copy(rr)
	}
	return out
}
