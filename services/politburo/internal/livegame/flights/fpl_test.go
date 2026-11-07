package flights

import (
	"testing"
	"time"
)

func TestPlanFreshMissing(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if PlanFresh(time.Time{}, now) {
		t.Fatal("expected missing plan to be stale")
	}
}

func TestPlanFreshRespectsInterval(t *testing.T) {
	last := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if !PlanFresh(last, last.Add(FPLSyncInterval-time.Second)) {
		t.Fatal("expected fresh plan inside interval")
	}
	if PlanFresh(last, last.Add(FPLSyncInterval)) {
		t.Fatal("expected stale plan at interval boundary")
	}
}

func TestPlanSyncDueRequiresVALink(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if PlanSyncDue(false, PilotStateActive, time.Time{}, now) {
		t.Fatal("expected no sync without VA link")
	}
	if !PlanSyncDue(true, PilotStateActive, time.Time{}, now) {
		t.Fatal("expected sync for VA-linked active pilot without plan")
	}
}

func TestPlanSyncDueIgnoresNonActivePilot(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if PlanSyncDue(true, PilotStateInBackground, time.Time{}, now) {
		t.Fatal("expected no sync for background pilot")
	}
}
