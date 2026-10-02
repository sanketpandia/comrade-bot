package flights

import (
	"math"
	"sort"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func MotionFromUpstream(upstream infiniteflight.Flight) FlightMotion {
	return FlightMotion{
		Latitude:  math.Round(upstream.Latitude*10000) / 10000,
		Longitude: math.Round(upstream.Longitude*10000) / 10000,
		Speed:     int(math.Round(upstream.Speed)),
		Callsign:  upstream.Callsign,
		Track:     math.Round(upstream.Track*10) / 10,
	}
}

func MotionFromFlight(flight Flight) FlightMotion {
	return FlightMotion{
		Latitude:  flight.Latitude,
		Longitude: flight.Longitude,
		Speed:     flight.Speed,
		Callsign:  flight.Callsign,
		Track:     flight.Track,
	}
}

func MotionEqual(a, b FlightMotion) bool {
	return a.Latitude == b.Latitude && a.Longitude == b.Longitude && a.Speed == b.Speed
}

// SortedTrackFlightIDs returns track keys sorted for stable API ordering.
func SortedTrackFlightIDs(tracks map[string]FlightMotion) []string {
	if len(tracks) == 0 {
		return nil
	}
	ids := make([]string, 0, len(tracks))
	for id := range tracks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
