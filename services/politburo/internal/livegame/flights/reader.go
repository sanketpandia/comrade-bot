package flights

import (
	"context"
	"errors"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
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

// CountActive returns how many flights are in the server snapshot.
func (r *Reader) CountActive(ctx context.Context, server string) (int, error) {
	if strings.TrimSpace(server) == "" {
		return 0, ErrInvalidFilter
	}
	snapshot, err := r.loadSnapshot(ctx, server)
	if err != nil {
		return 0, err
	}
	return len(snapshot.Tracks), nil
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

// GetActiveFlight returns the enriched per-flight cache record for flightID.
func (r *Reader) GetActiveFlight(ctx context.Context, flightID string) (Record, error) {
	if strings.TrimSpace(flightID) == "" {
		return Record{}, ErrInvalidFilter
	}
	return r.loadFlightRecord(ctx, flightID)
}

// GetFlightHistory returns temporal estimator history for flightID (cache miss yields empty history).
func (r *Reader) GetFlightHistory(ctx context.Context, flightID string) (flightstate.History, error) {
	if strings.TrimSpace(flightID) == "" {
		return flightstate.History{}, ErrInvalidFilter
	}
	var history flightstate.History
	if err := r.cache.GetJSON(ctx, cache.KeyFlightHistory(flightID), &history); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return flightstate.History{}, nil
		}
		return flightstate.History{}, ErrCacheRead
	}
	return history, nil
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
	if query.VACode != "" {
		vaFlightIDs, err := r.loadVAActiveFlightIDs(ctx, query.VACode)
		if err != nil {
			return nil, err
		}
		allowed := make(map[string]bool, len(vaFlightIDs))
		for _, id := range vaFlightIDs {
			allowed[id] = true
		}
		ids = filterIDsBySet(ids, allowed)
	}
	pilotSeen := pilotStateSet(query.PilotStates)
	hasFilters := query.VACode != "" || len(query.PilotStates) > 0 || query.UserName != "" || query.CallSign != ""

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
		out = append(out, TrackFromRecord(flight))
	}
	return out, nil
}

// loadVAActiveFlightIDs resolves vaCode through the cached code index, then loads that
// VA's flight ID list. The VA flight index job writes both for every active VA.
func (r *Reader) loadVAActiveFlightIDs(ctx context.Context, vaCode string) ([]string, error) {
	var codes map[string]string
	if err := r.cache.GetJSON(ctx, cache.KeyVACodeIndex, &codes); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return nil, ErrCacheMiss
		}
		return nil, ErrCacheRead
	}
	vaID, ok := codes[strings.ToUpper(strings.TrimSpace(vaCode))]
	if !ok {
		return nil, ErrUnknownVACode
	}

	var ids []string
	if err := r.cache.GetJSON(ctx, cache.KeyVAActiveFlights(vaID), &ids); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return nil, ErrCacheMiss
		}
		return nil, ErrCacheRead
	}
	return ids, nil
}

func filterIDsBySet(ids []string, allowed map[string]bool) []string {
	if len(allowed) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if allowed[id] {
			out = append(out, id)
		}
	}
	return out
}

func (r *Reader) loadFlightRecord(ctx context.Context, flightID string) (Record, error) {
	record := Record{}
	if err := r.cache.GetJSON(ctx, cache.KeyFlightRecord(flightID), &record); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return Record{}, ErrCacheMiss
		}
		return Record{}, ErrCacheRead
	}
	return record, nil
}
