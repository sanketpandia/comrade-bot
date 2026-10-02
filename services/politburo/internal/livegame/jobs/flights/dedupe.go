package flights

import "infinite-experiment/politburo/internal/livegame/infiniteflight"

// dedupeUpstream keeps one row per flight ID; later upstream rows win.
func dedupeUpstream(flights []infiniteflight.Flight) []infiniteflight.Flight {
	if len(flights) == 0 {
		return nil
	}
	order := make([]string, 0, len(flights))
	latest := make(map[string]infiniteflight.Flight, len(flights))
	for _, item := range flights {
		if item.FlightID == "" {
			continue
		}
		if _, seen := latest[item.FlightID]; !seen {
			order = append(order, item.FlightID)
		}
		latest[item.FlightID] = item
	}
	out := make([]infiniteflight.Flight, 0, len(order))
	for _, id := range order {
		out = append(out, latest[id])
	}
	return out
}
