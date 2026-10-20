// Package vaflightindex builds the Redis indexes behind the active-flights vaCode filter:
// a VA code → VA ID map and a per-VA list of active flight IDs. It runs on its own
// schedule and reads the snapshots and records written by the flights job.
package vaflightindex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
)

const jobName = "infinite-flight-va-flight-index"

type activeVALister interface {
	ListActive(ctx context.Context) ([]virtualairlines.VirtualAirline, error)
}

type Job struct {
	cache   cache.Store
	vas     activeVALister
	metrics *metrics.Registry
}

func New(cacheStore cache.Store, vas activeVALister, metricsRegistry *metrics.Registry) *Job {
	return &Job{cache: cacheStore, vas: vas, metrics: metricsRegistry}
}

func (j *Job) Name() string {
	return jobName
}

// Run writes the code map and one flight ID list per active VA (empty when the VA has
// no matched flights), so readers can treat a missing key as a cache miss.
func (j *Job) Run(ctx context.Context) error {
	vas, err := j.vas.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active virtual airlines: %w", err)
	}
	codes := make(map[string]string, len(vas))
	for _, va := range vas {
		codes[strings.ToUpper(va.Code)] = va.ID
	}
	if err := j.cache.SetJSON(ctx, cache.KeyVACodeIndex, codes, cache.ActiveFlightsTTL); err != nil {
		return fmt.Errorf("cache VA code index: %w", err)
	}

	var sessionsSnapshot gamesessions.Snapshot
	if err := j.cache.GetJSON(ctx, cache.KeyActiveSessions, &sessionsSnapshot); err != nil {
		slog.Warn("skipping VA flight index; active sessions cache unavailable", "error", err)
		return nil
	}
	if len(sessionsSnapshot.Result) == 0 {
		slog.Warn("skipping VA flight index; no active sessions cached")
		return nil
	}

	buildInput, stats, err := j.loadBuildInput(ctx, sessionsSnapshot)
	if err != nil {
		return err
	}
	index := BuildVAFlightIndex(buildInput)

	if j.metrics != nil {
		j.metrics.VAActiveFlights.Reset()
	}
	matchedVAs := 0
	for _, va := range vas {
		ids := index[va.ID]
		if ids == nil {
			ids = []string{}
		} else {
			matchedVAs++
		}
		if err := j.cache.SetJSON(ctx, cache.KeyVAActiveFlights(va.ID), ids, cache.ActiveFlightsTTL); err != nil {
			return fmt.Errorf("cache VA active flights %s: %w", va.ID, err)
		}
		if j.metrics != nil {
			j.metrics.VAActiveFlights.WithLabelValues(va.ID).Set(float64(len(ids)))
		}
	}

	slog.Info("VA active flight index refreshed",
		"sessions", len(sessionsSnapshot.Result),
		"flightsConsidered", stats.flightsConsidered,
		"recordMisses", stats.recordMisses,
		"activeVAs", len(vas),
		"matchedVAs", matchedVAs,
	)
	return nil
}

type buildStats struct {
	flightsConsidered int
	recordMisses      int
}

func (j *Job) loadBuildInput(ctx context.Context, sessionsSnapshot gamesessions.Snapshot) (BuildInput, buildStats, error) {
	input := BuildInput{
		ActiveFlightIDs: map[string][]string{},
		Records:         map[string]gameflights.Flight{},
	}
	var stats buildStats
	for _, session := range sessionsSnapshot.Result {
		server := session.NormalizedName
		if server == "" {
			continue
		}
		var snapshot gameflights.Snapshot
		if err := j.cache.GetJSON(ctx, cache.KeyActiveFlights(server), &snapshot); err != nil {
			if errors.Is(err, cache.ErrMiss) {
				continue
			}
			return BuildInput{}, stats, fmt.Errorf("read active flights snapshot %s: %w", server, err)
		}
		flightIDs := gameflights.SortedTrackFlightIDs(snapshot.Tracks)
		input.ActiveFlightIDs[server] = flightIDs

		for _, flightID := range flightIDs {
			stats.flightsConsidered++
			if _, loaded := input.Records[flightID]; loaded {
				continue
			}
			var record gameflights.Flight
			if err := j.cache.GetJSON(ctx, cache.KeyFlightRecord(flightID), &record); err != nil {
				if errors.Is(err, cache.ErrMiss) {
					stats.recordMisses++
					slog.Debug("skip flight in VA index; record cache miss", "flightId", flightID, "server", server)
					continue
				}
				return BuildInput{}, stats, fmt.Errorf("read flight record %s: %w", flightID, err)
			}
			input.Records[flightID] = record
		}
	}
	return input, stats, nil
}
