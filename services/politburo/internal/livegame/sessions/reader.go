package sessions

import (
	"context"
	"errors"
	"time"

	"infinite-experiment/politburo/internal/cache"
)

var (
	ErrCacheMiss    = errors.New("active sessions cache miss")
	ErrCacheCorrupt = errors.New("active sessions cache corrupt")
	ErrCacheRead    = errors.New("active sessions cache read failed")
)

type Reader struct {
	cache cache.Store
}

func NewReader(cacheStore cache.Store) *Reader {
	return &Reader{cache: cacheStore}
}

type ActiveSession struct {
	NormalizedName string
	UserCount      int
	Type           int
}

type ListActiveResult struct {
	Sessions   []ActiveSession
	LastCached time.Time
}

func (r *Reader) ListActive(ctx context.Context) (ListActiveResult, error) {
	snapshot := Snapshot{}
	if err := r.cache.GetJSON(ctx, cache.KeyActiveSessions, &snapshot); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return ListActiveResult{}, ErrCacheMiss
		}
		return ListActiveResult{}, ErrCacheRead
	}
	if snapshot.LastCached.IsZero() {
		return ListActiveResult{}, ErrCacheCorrupt
	}

	result := make([]ActiveSession, 0, len(snapshot.Result))
	for _, session := range snapshot.Result {
		result = append(result, ActiveSession{
			NormalizedName: session.NormalizedName,
			UserCount:      session.UserCount,
			Type:           session.Type,
		})
	}
	return ListActiveResult{Sessions: result, LastCached: snapshot.LastCached}, nil
}
