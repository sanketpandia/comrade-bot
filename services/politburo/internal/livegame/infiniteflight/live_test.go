package infiniteflight

import (
	"context"
	"encoding/json"
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
