// Package vaflightindex builds per-VA Redis indexes of active flight IDs after flights sync.
package vaflightindex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
)

const jobName = "infinite-flight-va-flight-index"

type Job struct {
	cache   cache.Store
	metrics *metrics.Registry
}

func New(cacheStore cache.Store, metricsRegistry *metrics.Registry) *Job {
	return &Job{cache: cacheStore, metrics: metricsRegistry}
}

func (j *Job) Name() string {
	return jobName
}

func (j *Job) Run(ctx context.Context) error {
	var sessionsSnapshot gamesessions.Snapshot
	if err := j.cache.GetJSON(ctx, cache.KeyActiveSessions, &sessionsSnapshot); err != nil {
		slog.Warn("skipping VA flight index; active sessions cache unavailable", "error", err)
		return nil
	}
	if len(sessionsSnapshot.Result) == 0 {
		slog.Warn("skipping VA flight index; no active sessions cached")
		return nil
	}

	buildInput := BuildInput{
		ActiveFlightIDs: map[string][]string{},
		Records:         map[string]gameflights.Flight{},
	}
	flightsConsidered := 0
	recordMisses := 0

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
			return fmt.Errorf("read active flights snapshot %s: %w", server, err)
		}
		if snapshot.Tracks == nil {
			continue
		}
		flightIDs := make([]string, 0, len(snapshot.Tracks))
		for flightID := range snapshot.Tracks {
			flightIDs = append(flightIDs, flightID)
		}
		sort.Strings(flightIDs)
		buildInput.ActiveFlightIDs[server] = flightIDs

		for _, flightID := range flightIDs {
			flightsConsidered++
			if _, loaded := buildInput.Records[flightID]; loaded {
				continue
			}
			var record gameflights.Flight
			if err := j.cache.GetJSON(ctx, cache.KeyFlightRecord(flightID), &record); err != nil {
				if errors.Is(err, cache.ErrMiss) {
					recordMisses++
					slog.Debug("skip flight in VA index; record cache miss", "flightId", flightID, "server", server)
					continue
				}
				return fmt.Errorf("read flight record %s: %w", flightID, err)
			}
			buildInput.Records[flightID] = record
		}
	}

	index := BuildVAFlightIndex(buildInput)
	for vaID, ids := range index {
		if err := j.cache.SetJSON(ctx, cache.KeyVAActiveFlights(vaID), ids, cache.ActiveFlightsTTL); err != nil {
			return fmt.Errorf("cache VA active flights %s: %w", vaID, err)
		}
		if j.metrics != nil {
			j.metrics.VAActiveFlights.WithLabelValues(vaID).Set(float64(len(ids)))
		}
	}

	slog.Info("VA active flight index refreshed",
		"sessions", len(sessionsSnapshot.Result),
		"flightsConsidered", flightsConsidered,
		"recordMisses", recordMisses,
		"vaKeysWritten", len(index),
	)
	return nil
}
