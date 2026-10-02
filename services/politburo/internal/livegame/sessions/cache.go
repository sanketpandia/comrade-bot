// Package sessions defines the shared cache contract for active game sessions.
package sessions

import (
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type Snapshot struct {
	Result     []infiniteflight.Session `json:"result"`
	LastCached time.Time                `json:"lastCached"`
}

// SnapshotFresh reports whether lastCached is within the scheduled refresh window.
func SnapshotFresh(lastCached, now time.Time) bool {
	if lastCached.IsZero() {
		return false
	}
	return now.Sub(lastCached) <= cache.SessionsRefreshInterval
}
