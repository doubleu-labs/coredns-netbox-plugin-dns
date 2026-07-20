package cache

import "slices"

type ixfrResult struct {
	deltas       []zoneDelta
	fallbackAXFR bool
}

func ixfrDeltasFrom(zCache *zoneCache, serial uint32) ixfrResult {
	if len(zCache.deltas) == 0 {
		return ixfrResult{fallbackAXFR: true}
	}

	// check if requested serial is older than the oldest delta in this zone
	if serialBefore(serial, zCache.deltas[0].fromSerial) {
		return ixfrResult{fallbackAXFR: true}
	}

	var chain []zoneDelta
	currentSearch := serial

	for delta := range slices.Values(zCache.deltas) {
		if serialBefore(delta.fromSerial, currentSearch) {
			continue
		}
		if delta.fromSerial == currentSearch {
			chain = append(chain, delta)
			currentSearch = delta.toSerial
		}
		if currentSearch == zCache.currentSerial {
			return ixfrResult{deltas: chain}
		}
	}

	return ixfrResult{fallbackAXFR: true}
}
