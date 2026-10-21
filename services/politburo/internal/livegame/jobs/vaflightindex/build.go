package vaflightindex

import (
	"sort"

	gameflights "infinite-experiment/politburo/internal/livegame/flights"
)

// BuildInput is the in-memory data needed to build a VA → flight ID index.
type BuildInput struct {
	// ActiveFlightIDs lists flight IDs currently on each normalized server snapshot.
	ActiveFlightIDs map[string][]string
	// Records holds enriched flight cache records keyed by flight ID.
	Records map[string]gameflights.Record
}

// BuildVAFlightIndex aggregates matched VA IDs into sorted, deduped flight ID lists.
func BuildVAFlightIndex(input BuildInput) map[string][]string {
	sets := map[string]map[string]bool{}
	for _, flightIDs := range input.ActiveFlightIDs {
		for _, flightID := range flightIDs {
			record, ok := input.Records[flightID]
			if !ok {
				continue
			}
			for _, match := range record.Derived.MatchedVirtualAirlines {
				if match.VAID == "" {
					continue
				}
				if sets[match.VAID] == nil {
					sets[match.VAID] = map[string]bool{}
				}
				sets[match.VAID][flightID] = true
			}
		}
	}

	out := make(map[string][]string, len(sets))
	for vaID, ids := range sets {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		sort.Strings(list)
		out[vaID] = list
	}
	return out
}
