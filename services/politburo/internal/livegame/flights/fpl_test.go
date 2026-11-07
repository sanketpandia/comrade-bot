package flights

import (
	"testing"
	"time"
)

func TestFPLSyncDueActivePilotWithoutPriorSync(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if !FPLSyncDue(PilotStateActive, FPLSyncAction{}, now) {
		t.Fatal("expected FPL sync due")
	}
}

func TestFPLSyncDueRespectsInterval(t *testing.T) {
	last := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	fpl := FPLSyncAction{LastAt: last}
	if FPLSyncDue(PilotStateActive, fpl, last.Add(FPLSyncInterval-time.Second)) {
		t.Fatal("expected FPL sync not due yet")
	}
	if !FPLSyncDue(PilotStateActive, fpl, last.Add(FPLSyncInterval)) {
		t.Fatal("expected FPL sync due")
	}
}

func TestFPLSyncDueIgnoresNonActivePilot(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if FPLSyncDue(PilotStateInBackground, FPLSyncAction{}, now) {
		t.Fatal("expected no FPL sync for background pilot")
	}
}
