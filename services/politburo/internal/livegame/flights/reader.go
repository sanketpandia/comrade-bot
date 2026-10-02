package flights

import (
	"context"
	"errors"
	"time"

	"infinite-experiment/politburo/internal/cache"
)

type Reader struct {
	cache cache.Store
}

func NewReader(cacheStore cache.Store) *Reader {
	return &Reader{cache: cacheStore}
}

type ListActiveTracksInput struct {
	Server     string
	PageNumber int
	PageLength int
	Filters    FilterQuery
}

type ListActiveTracksResult struct {
	Tracks           []Track
	AvailableFilters []AvailableFilter
	LastCached       time.Time
	TotalLength      int
	PageNumber       int
	PageLength       int
}

// ListActiveTracks reads the per-server snapshot, applies filters, and paginates stable track rows.
func (r *Reader) ListActiveTracks(ctx context.Context, input ListActiveTracksInput) (ListActiveTracksResult, error) {
	if input.Server == "" {
		return ListActiveTracksResult{}, ErrInvalidFilter
	}

	known, err := r.knownServer(ctx, input.Server)
	if err != nil {
		return ListActiveTracksResult{}, err
	}
	if !known {
		return ListActiveTracksResult{}, ErrUnknownServer
	}

	snapshot, err := r.loadSnapshot(ctx, input.Server)
	if err != nil {
		return ListActiveTracksResult{}, err
	}

	tracks, err := r.buildTracks(ctx, snapshot, input.Filters)
	if err != nil {
		return ListActiveTracksResult{}, err
	}

	total := len(tracks)
	paged := Paginate(tracks, input.PageNumber, input.PageLength)

	return ListActiveTracksResult{
		Tracks:           paged,
		AvailableFilters: BuildAvailableFilters(input.Filters),
		LastCached:       snapshot.LastCached,
		TotalLength:      total,
		PageNumber:       input.PageNumber,
		PageLength:       input.PageLength,
	}, nil
}

func (r *Reader) loadSnapshot(ctx context.Context, server string) (Snapshot, error) {
	snapshot := Snapshot{}
	if err := r.cache.GetJSON(ctx, cache.KeyActiveFlights(server), &snapshot); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return Snapshot{}, ErrCacheMiss
		}
		return Snapshot{}, ErrCacheRead
	}
	if snapshot.LastCached.IsZero() {
		return Snapshot{}, ErrCacheCorrupt
	}
	if snapshot.Tracks == nil {
		snapshot.Tracks = make(map[string]FlightMotion)
	}
	return snapshot, nil
}

func (r *Reader) knownServer(ctx context.Context, serverID string) (bool, error) {
	var names []string
	if err := r.cache.GetJSON(ctx, cache.KeySessionNames, &names); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return true, nil
		}
		return false, ErrSessionNames
	}
	for _, name := range names {
		if name == serverID {
			return true, nil
		}
	}
	return false, nil
}

func (r *Reader) buildTracks(ctx context.Context, snapshot Snapshot, query FilterQuery) ([]Track, error) {
	ids := SortedTrackFlightIDs(snapshot.Tracks)
	pilotSeen := pilotStateSet(query.PilotStates)
	hasFilters := len(query.PilotStates) > 0 || query.UserName != "" || query.CallSign != ""

	out := make([]Track, 0, len(ids))
	for _, id := range ids {
		motion := snapshot.Tracks[id]
		if !hasFilters {
			out = append(out, TrackFromMotion(id, motion))
			continue
		}

		flight, err := r.loadFlightRecord(ctx, id)
		if err != nil {
			if errors.Is(err, ErrCacheMiss) {
				continue
			}
			return nil, err
		}
		if !MatchesFilters(flight, query, pilotSeen) {
			continue
		}
		out = append(out, TrackFromFlight(flight))
	}
	return out, nil
}

func (r *Reader) loadFlightRecord(ctx context.Context, flightID string) (Flight, error) {
	flight := Flight{}
	if err := r.cache.GetJSON(ctx, cache.KeyFlightRecord(flightID), &flight); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return Flight{}, ErrCacheMiss
		}
		return Flight{}, ErrCacheRead
	}
	return flight, nil
}
