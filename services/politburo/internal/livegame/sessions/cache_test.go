package sessions

import (
	"testing"
	"time"
)

func TestSnapshotFresh(t *testing.T) {
	t.Parallel()

	lastCached := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if !SnapshotFresh(lastCached, lastCached.Add(RefreshInterval)) {
		t.Fatal("expected fresh at exactly RefreshInterval")
	}
	if SnapshotFresh(lastCached, lastCached.Add(RefreshInterval+time.Second)) {
		t.Fatal("expected stale after RefreshInterval")
	}
	if SnapshotFresh(time.Time{}, time.Now()) {
		t.Fatal("expected zero lastCached to be stale")
	}
}
