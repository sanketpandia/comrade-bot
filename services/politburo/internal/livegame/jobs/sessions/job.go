package sessions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const jobName = "infinite-flight-sessions"

type Job struct {
	client infiniteflight.SessionsClient
	cache  cache.Store
	now    func() time.Time
}

func New(client infiniteflight.SessionsClient, cacheStore cache.Store) *Job {
	return &Job{client: client, cache: cacheStore, now: time.Now}
}

func (j *Job) Name() string {
	return jobName
}

func (j *Job) Run(ctx context.Context) error {
	sessions, err := j.client.GetSessions(ctx)
	if err != nil {
		return fmt.Errorf("refresh sessions: %w", err)
	}

	// time.Time marshals as an ISO 8601/RFC 3339 timestamp. Capture it once so
	// every session and its enclosing snapshot have the same refresh time.
	refreshedAt := j.now().UTC()
	sessions = prepareCurrentSessions(sessions, refreshedAt)

	snapshot := gamesessions.Snapshot{
		Result:     sessions,
		LastCached: refreshedAt,
	}

	if err := j.cache.SetJSON(ctx, cache.KeyActiveSessions, snapshot, cache.SessionsCacheTTL); err != nil {
		return fmt.Errorf("cache sessions: %w", err)
	}

	sessionNames := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionNames = append(sessionNames, session.NormalizedName)
	}
	if err := j.cache.SetJSON(ctx, cache.KeySessionNames, sessionNames, 0); err != nil {
		return fmt.Errorf("cache session names: %w", err)
	}

	userCount := 0
	for _, session := range sessions {
		userCount += session.UserCount
	}
	slog.Info("Infinite Flight sessions refreshed", "sessions", len(sessions), "users", userCount)
	return nil
}

// prepareCurrentSessions creates a new result slice containing only the latest
// upstream value for each session.
func prepareCurrentSessions(upstream []infiniteflight.Session, refreshedAt time.Time) []infiniteflight.Session {
	current := make([]infiniteflight.Session, 0, len(upstream))
	indexes := make(map[string]int, len(upstream))
	for _, session := range upstream {
		session.Timestamp = refreshedAt
		session.NormalizedName = infiniteflight.NormalizeSessionName(session.Name)
		identity := session.ID
		if identity == "" {
			identity = "name:" + session.NormalizedName
		}
		if index, exists := indexes[identity]; exists {
			current[index] = session
			continue
		}
		indexes[identity] = len(current)
		current = append(current, session)
	}
	return current
}
