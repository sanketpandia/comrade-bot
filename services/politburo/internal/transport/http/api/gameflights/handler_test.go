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
	flights   domainflights.Snapshot
	records   map[string]domainflights.Record
	vaCodes   map[string]string
	vaIndexes map[string][]string
	names     []string
	namesErr  error
	err       error
	server    string
}

func (s cacheStub) GetJSON(_ context.Context, key string, destination any) error {
	switch {
	case key == cache.KeySessionNames:
		if s.namesErr != nil {
			return s.namesErr
		}
		*(destination.(*[]string)) = s.names
		return nil
	case key == cache.KeyVACodeIndex:
		if s.vaCodes == nil {
			return cache.ErrMiss
		}
		*(destination.(*map[string]string)) = s.vaCodes
		return nil
	case strings.HasPrefix(key, cache.PrefixGameFlightsActive):
		if s.err != nil {
			return s.err
		}
		*(destination.(*domainflights.Snapshot)) = s.flights
		return nil
	case strings.HasPrefix(key, cache.PrefixGameFlightsVA):
		if s.err != nil {
			return s.err
		}
		if s.vaIndexes != nil {
			vaID := strings.TrimPrefix(key, cache.PrefixGameFlightsVA)
			if ids, ok := s.vaIndexes[vaID]; ok {
				*(destination.(*[]string)) = ids
				return nil
			}
		}
		return cache.ErrMiss
	default:
		if strings.HasPrefix(key, cache.PrefixGameFlightsRecord) {
			if s.err != nil {
				return s.err
			}
		}
		if s.records != nil {
			for id, flight := range s.records {
				if key == cache.KeyFlightRecord(id) {
					*(destination.(*domainflights.Record)) = flight
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

func flightsFixture(lastCached time.Time, records ...domainflights.Record) (domainflights.Snapshot, map[string]domainflights.Record) {
	tracks := make(map[string]domainflights.FlightMotion, len(records))
	byID := make(map[string]domainflights.Record, len(records))
	for i := range records {
		if records[i].Raw.FlightID == "" {
			records[i].Raw.FlightID = "c34118e7-cbdd-4e22-8751-0cda93e41d75"
		}
		id := records[i].Raw.FlightID
		tracks[id] = domainflights.MotionFromRecord(records[i])
		byID[id] = records[i]
	}
	return domainflights.Snapshot{LastCached: lastCached, Tracks: tracks}, byID
}

func sampleFlight(state string) domainflights.Record {
	return domainflights.Record{
		Raw: domainflights.Observation{
			FlightID:       "c34118e7-cbdd-4e22-8751-0cda93e41d75",
			Callsign:       "Swiss 39 Heavy",
			Latitude:       47.45,
			Longitude:      8.56,
			Track:          329.2,
			NormalizedName: "casual",
		},
		Derived: domainflights.Derived{
			Normalized: domainflights.Normalized{PilotState: state, Speed: "526 kts", VerticalSpeed: "0.0 ft/min", IsConnected: "disconnected"},
			PathSync:   domainflights.PathSync{FPLSyncRequired: false},
		},
	}
}
