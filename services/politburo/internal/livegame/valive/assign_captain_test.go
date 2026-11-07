package valive

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"infinite-experiment/politburo/internal/community/registration/proof"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type fakeScheduleCrewLive struct {
	aircraft []infiniteflight.LiveAircraft
	pages    []infiniteflight.LiveSchedulePage
	assigned []struct {
		scheduleID, userID string
		body               json.RawMessage
	}
	assignErr error
}

func (f *fakeScheduleCrewLive) ListOrganizationAircraft(_ context.Context, _, _ string) ([]infiniteflight.LiveAircraft, error) {
	return f.aircraft, nil
}

func (f *fakeScheduleCrewLive) ListOrganizationSchedules(_ context.Context, _, _ string, page, _ int, _ []string) (infiniteflight.LiveSchedulePage, error) {
	if page < 1 || page > len(f.pages) {
		return infiniteflight.LiveSchedulePage{}, nil
	}
	return f.pages[page-1], nil
}

func (f *fakeScheduleCrewLive) AssignScheduleCrew(_ context.Context, _, scheduleID, userID string, body json.RawMessage) error {
	f.assigned = append(f.assigned, struct {
		scheduleID, userID string
		body               json.RawMessage
	}{scheduleID, userID, body})
	return f.assignErr
}

func TestFindFirstMatchingScheduleOrder(t *testing.T) {
	live := &fakeScheduleCrewLive{
		pages: []infiniteflight.LiveSchedulePage{
			{
				Items: []infiniteflight.LiveSchedule{
					{ID: "skip", AircraftID: "ac-1", OriginIcao: "EHAM", DestinationIcao: "KJFK"},
					{ID: "win", AircraftID: "ac-1", OriginIcao: "EHAM", DestinationIcao: "CYYZ"},
				},
				Page: 1, TotalPages: 2,
			},
			{
				Items: []infiniteflight.LiveSchedule{
					{ID: "late", AircraftID: "ac-1", OriginIcao: "EHAM", DestinationIcao: "CYYZ"},
				},
				Page: 2, TotalPages: 2,
			},
		},
	}
	got, err := findFirstMatchingSchedule(context.Background(), live, "token", "org-1", []string{"ac-1"}, "EHAM", "CYYZ")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got == nil || got.ID != "win" {
		t.Fatalf("got = %+v, want win", got)
	}
}

func TestFindFirstMatchingScheduleSecondPage(t *testing.T) {
	live := &fakeScheduleCrewLive{
		pages: []infiniteflight.LiveSchedulePage{
			{
				Items:      []infiniteflight.LiveSchedule{{ID: "a", OriginIcao: "EHAM", DestinationIcao: "KJFK"}},
				Page:       1,
				TotalPages: 2,
			},
			{
				Items:      []infiniteflight.LiveSchedule{{ID: "b", OriginIcao: "EHAM", DestinationIcao: "CYYZ"}},
				Page:       2,
				TotalPages: 2,
			},
		},
	}
	got, err := findFirstMatchingSchedule(context.Background(), live, "token", "org-1", []string{"ac-1"}, "EHAM", "CYYZ")
	if err != nil || got == nil || got.ID != "b" {
		t.Fatalf("got = %+v err = %v", got, err)
	}
}

func TestAssignCaptainInvalidRoute(t *testing.T) {
	svc := NewAssignCaptainService(nil, nil, &fakeScheduleCrewLive{})
	_, err := svc.AssignCaptain(context.Background(), AssignCaptainInput{
		VAID: "va", OrganizationID: "org", IFUserID: "u", Registration: "PH-ZNB", Route: "BAD",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, proof.ErrInvalidRoute) {
		t.Fatalf("err = %v want ErrInvalidRoute", err)
	}
}
