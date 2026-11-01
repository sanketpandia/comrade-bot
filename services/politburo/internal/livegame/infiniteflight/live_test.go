package infiniteflight

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLiveClientListOrganizationsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live/organizations" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"result": []map[string]any{
				{"id": "org-1", "name": "PacificJet", "role": 2},
			},
		})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	orgs, err := client.ListOrganizations(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("ListOrganizations() error = %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != "org-1" || orgs[0].Role != 2 {
		t.Fatalf("orgs = %+v", orgs)
	}
}

func TestLiveClientListOrganizationsNonZeroErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errorCode": 3, "result": []any{}})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizations(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListOrganizationsNonOKHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizations(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListOrganizationAircraftSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live/organizations/org-1/aircraft" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"result": []map[string]any{
				{
					"id":                "ac-persist-1",
					"aircraftId":        "content-1",
					"organizationId":    "org-1",
					"registration":      "N682XL",
					"status":            0,
					"visibility":        1,
					"fleetPriority":     1000,
					"fleetRank":         1,
					"isFleetActiveSlot": true,
					"createdAt":         "2026-07-03T12:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	rows, err := client.ListOrganizationAircraft(context.Background(), "access-token", "org-1")
	if err != nil {
		t.Fatalf("ListOrganizationAircraft() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "ac-persist-1" || rows[0].Registration != "N682XL" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestLiveClientListOrganizationAircraftNonZeroErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errorCode": 3, "result": []any{}})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationAircraft(context.Background(), "token", "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListOrganizationAircraftNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationAircraft(context.Background(), "token", "org-1")
	if !errors.Is(err, ErrLiveOrganizationNotFound) {
		t.Fatalf("err = %v, want ErrLiveOrganizationNotFound", err)
	}
}

func TestLiveClientListOrganizationAircraftNonOKHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationAircraft(context.Background(), "token", "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListOrganizationSchedulesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live/organizations/org-1/schedules" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page = %q", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "100" {
			t.Fatalf("pageSize = %q", got)
		}
		if ids := r.URL.Query()["aircraftId"]; len(ids) != 1 || ids[0] != "ac-1" {
			t.Fatalf("aircraftId = %v", r.URL.Query()["aircraftId"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"result": map[string]any{
				"items": []map[string]any{
					{
						"id":                    "sched-1",
						"status":                1,
						"callsign":              "PJX421",
						"organizationId":        "org-1",
						"aircraftId":            "ac-1",
						"flightType":            1,
						"originIcao":            "KJFK",
						"destinationIcao":       "KLAX",
						"isTimeless":            false,
						"scheduledDepartureUtc": "2026-07-03T14:00:00Z",
						"sequence":              1,
					},
				},
				"page":       1,
				"pageSize":   100,
				"totalCount": 1,
				"totalPages": 1,
			},
		})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	page, err := client.ListOrganizationSchedules(context.Background(), "access-token", "org-1", 1, 100, []string{"ac-1"})
	if err != nil {
		t.Fatalf("ListOrganizationSchedules() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "sched-1" || page.Items[0].Callsign != "PJX421" {
		t.Fatalf("page = %+v", page)
	}
}

func TestLiveClientListOrganizationSchedulesNonZeroErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errorCode": 3, "result": map[string]any{}})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationSchedules(context.Background(), "token", "org-1", 1, 100, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListOrganizationSchedulesNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationSchedules(context.Background(), "token", "org-1", 1, 100, nil)
	if !errors.Is(err, ErrLiveOrganizationNotFound) {
		t.Fatalf("err = %v, want ErrLiveOrganizationNotFound", err)
	}
}

func TestLiveClientListOrganizationSchedulesNonOKHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	_, err = client.ListOrganizationSchedules(context.Background(), "token", "org-1", 1, 100, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientListAircraftSchedulesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live/aircraft/ac-persist-1/schedules" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"result": []map[string]any{
				{
					"id":         "sched-1",
					"status":     1,
					"callsign":   "PJX421",
					"aircraftId": "ac-persist-1",
					"sequence":   1,
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	rows, err := client.ListAircraftSchedules(context.Background(), "access-token", "ac-persist-1")
	if err != nil {
		t.Fatalf("ListAircraftSchedules() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "sched-1" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestLiveClientListAircraftFlightsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live/aircraft/ac-persist-1/flights" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "50" {
			t.Fatalf("pageSize = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"result": map[string]any{
				"items": []map[string]any{
					{
						"id":                "flight-1",
						"aircraftId":        "ac-persist-1",
						"organizationId":    "org-1",
						"callsign":          "N682XL",
						"originIcao":        "KLAX",
						"destinationIcao":   "KJFK",
						"status":            1,
						"flightTimeMinutes": 309,
						"crew": []map[string]any{
							{"userId": "user-1", "username": "PilotOne"},
						},
						"pilotFlightTimes": []map[string]any{},
					},
				},
				"page":       2,
				"pageSize":   50,
				"totalCount": 1,
				"totalPages": 2,
			},
		})
	}))
	defer server.Close()

	client, err := NewLiveClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewLiveClient() error = %v", err)
	}
	page, err := client.ListAircraftFlights(context.Background(), "access-token", "ac-persist-1", 2, 50)
	if err != nil {
		t.Fatalf("ListAircraftFlights() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "flight-1" || page.Items[0].Callsign != "N682XL" {
		t.Fatalf("page = %+v", page)
	}
	if page.Page != 2 || page.PageSize != 50 {
		t.Fatalf("page meta = %+v", page)
	}
}
