// Package sessions defines the shared cache contract for active game sessions.
package sessions

import (
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const (
	RefreshSchedule = "0 */5 * * * *"
	RefreshInterval = 5 * time.Minute
	CacheTTL        = 24 * time.Hour
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
	return now.Sub(lastCached) <= RefreshInterval
}
