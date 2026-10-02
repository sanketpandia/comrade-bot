package sessions

import (
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
)

func TestSnapshotFresh(t *testing.T) {
	t.Parallel()

	lastCached := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if !SnapshotFresh(lastCached, lastCached.Add(cache.SessionsRefreshInterval)) {
		t.Fatal("expected fresh at exactly SessionsRefreshInterval")
	}
	if SnapshotFresh(lastCached, lastCached.Add(cache.SessionsRefreshInterval+time.Second)) {
		t.Fatal("expected stale after SessionsRefreshInterval")
	}
	if SnapshotFresh(time.Time{}, time.Now()) {
		t.Fatal("expected zero lastCached to be stale")
	}
}
