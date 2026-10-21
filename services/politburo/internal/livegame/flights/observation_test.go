package flights

import (
	"math"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestNewObservationPreservesUpstreamValues(t *testing.T) {
	username := "Hantder_Broncano_Jar"
	org := "IFAET"
	obs := NewObservation(infiniteflight.Flight{
		Username:            &username,
		Callsign:            "Swiss 39 Heavy",
		Latitude:            63.042991638183594,
		Longitude:           -30.311138153076172,
		Altitude:            35999.953,
		Speed:               525.6486,
		VerticalSpeed:       0.00010479759,
		Track:               106.800766,
		LastReport:          "2026-08-15 05:09:53Z",
		FlightID:            "c34118e7-cbdd-4e22-8751-0cda93e41d75",
		UserID:              "0d85b360-92b5-4e62-ac64-41bd1c829772",
		AircraftID:          "e258f6d4-4503-4dde-b25c-1fb9067061e2",
		LiveryID:            "df597aaf-456c-4878-9d84-45201f2aae74",
		VirtualOrganization: &org,
		PilotState:          PilotStateInBackground,
		IsConnected:         false,
	}, infiniteflight.Session{ID: "session-1", NormalizedName: "casual"}, time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC))

	if obs.Altitude != 35999.953 || obs.Speed != 525.6486 {
		t.Fatalf("expected unrounded upstream numerics, got alt %v speed %v", obs.Altitude, obs.Speed)
	}
	if obs.Latitude != 63.042991638183594 {
		t.Fatalf("latitude = %v", obs.Latitude)
	}
	if !obs.LastReportValid || !obs.LastReport.Equal(time.Date(2026, time.August, 15, 5, 9, 53, 0, time.UTC)) {
		t.Fatalf("lastReport = %s valid=%v", obs.LastReport, obs.LastReportValid)
	}
	if RoundedSpeed(obs.Speed) != 526 || RoundedAltitude(obs.Altitude) != 36000 {
		t.Fatalf("presentation rounding mismatch")
	}
	if math.Round(obs.Latitude*10000)/10000 != RoundedLatitude(obs.Latitude) {
		t.Fatal("latitude rounding mismatch")
	}
}

func TestNewObservationInvalidLastReport(t *testing.T) {
	obs := NewObservation(infiniteflight.Flight{FlightID: "f1", LastReport: "not-a-time"}, infiniteflight.Session{NormalizedName: "expert"}, time.Now().UTC())
	if obs.LastReportValid {
		t.Fatal("expected invalid last report")
	}
	if !obs.LastReport.IsZero() {
		t.Fatalf("lastReport = %s", obs.LastReport)
	}
}
