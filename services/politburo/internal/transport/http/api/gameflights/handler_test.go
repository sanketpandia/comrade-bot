package gameflights

import (
	"context"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/cache"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/metrics"
)

type cacheStub struct {
	flights  domainflights.Snapshot
	records  map[string]domainflights.Flight
	names    []string
	namesErr error
	err      error
	server   string
}

func (s cacheStub) GetJSON(_ context.Context, key string, destination any) error {
	switch {
	case key == cache.KeySessionNames:
		if s.namesErr != nil {
			return s.namesErr
		}
		*(destination.(*[]string)) = s.names
		return nil
	case strings.HasPrefix(key, cache.PrefixGameFlightsActive):
		if s.err != nil {
			return s.err
		}
		*(destination.(*domainflights.Snapshot)) = s.flights
		return nil
	default:
		if s.records != nil {
			for id, flight := range s.records {
				if key == cache.KeyFlightRecord(id) {
					*(destination.(*domainflights.Flight)) = flight
					return nil
				}
			}
		}
		return cache.ErrMiss
	}
}

func (cacheStub) SetJSON(context.Context, string, any, time.Duration) error { return nil }

func testHandler(store cache.Store) *Handler {
	return NewHandler(domainflights.NewReader(store), metrics.NewRegistry())
}

func flightsFixture(lastCached time.Time, flights ...domainflights.Flight) (domainflights.Snapshot, map[string]domainflights.Flight) {
	tracks := make(map[string]domainflights.FlightMotion, len(flights))
	records := make(map[string]domainflights.Flight, len(flights))
	for i := range flights {
		if flights[i].FlightID == "" {
			flights[i].FlightID = "c34118e7-cbdd-4e22-8751-0cda93e41d75"
		}
		id := flights[i].FlightID
		tracks[id] = domainflights.MotionFromFlight(flights[i])
		records[id] = flights[i]
	}
	return domainflights.Snapshot{LastCached: lastCached, Tracks: tracks}, records
}

func sampleFlight(state string) domainflights.Flight {
	return domainflights.Flight{
		FlightID:       "c34118e7-cbdd-4e22-8751-0cda93e41d75",
		Callsign:       "Swiss 39 Heavy",
		Latitude:       47.45,
		Longitude:      8.56,
		Track:          329.2,
		NormalizedName: "casual",
		Normalized:     domainflights.Normalized{PilotState: state, Speed: "526 kts", VerticalSpeed: "0.0 ft/min", IsConnected: "disconnected"},
		PathSync:       &domainflights.PathSync{FPLSyncRequired: false},
	}
}
