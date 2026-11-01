package ui

import (
	"context"
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestFilterOpenLiveSchedules(t *testing.T) {
	rows := []infiniteflight.LiveSchedule{
		{ID: "open", Status: 1},
		{ID: "cancelled", Status: 9},
		{ID: "arrived", Status: 11},
	}
	filtered := filterOpenLiveSchedules(rows)
	if len(filtered) != 1 || filtered[0].ID != "open" {
		t.Fatalf("filtered = %+v", filtered)
	}
}

func TestFetchLiveSchedulesUpstreamSingleAircraftUsesAircraftEndpoint(t *testing.T) {
	stub := &schedulesLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1"}},
		schedules: []infiniteflight.LiveSchedule{
			{ID: "s1", AircraftID: "ac-1", Status: 1},
			{ID: "s2", AircraftID: "ac-1", Status: 11},
		},
	}
	data, err := fetchLiveSchedulesUpstream(context.Background(), stub, "tok", "org-1", "ac-1", nil)
	if err != nil {
		t.Fatalf("fetchLiveSchedulesUpstream() error = %v", err)
	}
	if stub.aircraftScheduleCalls != 1 || stub.orgScheduleCalls != 0 {
		t.Fatalf("calls aircraft=%d org=%d", stub.aircraftScheduleCalls, stub.orgScheduleCalls)
	}
	if len(data.Schedules) != 1 || data.Schedules[0].ID != "s1" {
		t.Fatalf("schedules = %+v", data.Schedules)
	}
}

func TestFetchLiveSchedulesUpstreamAllAircraftUsesOrgPages(t *testing.T) {
	stub := &schedulesLiveStub{
		schedules: []infiniteflight.LiveSchedule{{ID: "s1", Status: 1}},
	}
	data, err := fetchLiveSchedulesUpstream(context.Background(), stub, "tok", "org-1", "", nil)
	if err != nil {
		t.Fatalf("fetchLiveSchedulesUpstream() error = %v", err)
	}
	if stub.orgScheduleCalls != 1 || stub.aircraftScheduleCalls != 0 {
		t.Fatalf("calls aircraft=%d org=%d", stub.aircraftScheduleCalls, stub.orgScheduleCalls)
	}
	if len(data.Schedules) != 1 {
		t.Fatalf("schedules = %+v", data.Schedules)
	}
}
