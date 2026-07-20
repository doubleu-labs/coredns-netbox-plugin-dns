package cache

import (
	"slices"

	"github.com/miekg/dns"
)

func sendBatched(ch chan<- []dns.RR, rrs []dns.RR) {
	for start := 0; start < len(rrs); start += transferBatchSize {
		end := start + transferBatchSize
		if end > len(rrs) {
			end = len(rrs)
		}
		ch <- slices.Clone(rrs[start:end])
	}
}

func soaBatch(soa *dns.SOA) []dns.RR {
	if soa == nil {
		return nil
	}
	return []dns.RR{dns.Copy(soa)}
}

func soaBatchWithSerial(soa *dns.SOA, serial uint32) []dns.RR {
	if soa == nil {
		return nil
	}
	copied, ok := dns.Copy(soa).(*dns.SOA)
	if !ok {
		return nil
	}
	copied.Serial = serial
	return []dns.RR{copied}
}

func streamAXFR(ch chan []dns.RR, zCache *zoneCache) {
	defer close(ch)
	soa := zCache.soa
	ch <- soaBatch(soa)
	sendBatched(ch, zCache.currentZone)
	ch <- soaBatch(soa)
}

func streamIXFR(ch chan []dns.RR, soa *dns.SOA, deltas []zoneDelta) {
	defer close(ch)
	for delta := range slices.Values(deltas) {
		ch <- soaBatchWithSerial(soa, delta.fromSerial)
		sendBatched(ch, delta.deleted)
		ch <- soaBatchWithSerial(soa, delta.toSerial)
		sendBatched(ch, delta.added)
	}
	ch <- soaBatch(soa)
}

func streamCurrentSOA(ch chan []dns.RR, soa *dns.SOA) {
	defer close(ch)
	ch <- soaBatch(soa)
}
