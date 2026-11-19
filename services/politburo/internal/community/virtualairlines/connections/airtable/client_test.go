package airtable

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/metrics"
)

func TestClientFetchSchemaSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/meta/bases/appBase/tables" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(SchemaResponse{Tables: []Table{{ID: "tbl", Name: "Pilots", Fields: []Field{{ID: "fld", Name: "Callsign", Type: "singleLineText"}}}}})
	}))
	defer srv.Close()

	client := NewClient(metrics.NewRegistry(), 5*time.Second).WithBaseURL(srv.URL)
	tables, err := client.FetchSchema(context.Background(), "token", "appBase")
	if err != nil {
		t.Fatalf("FetchSchema() = %v", err)
	}
	if len(tables) != 1 || tables[0].ID != "tbl" {
		t.Fatalf("tables = %+v", tables)
	}
}

func TestClientThrottled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := NewClient(metrics.NewRegistry(), time.Second).WithBaseURL(srv.URL)
	_, err := client.FetchSchema(context.Background(), "token", "appBase")
	var resp *ResponseError
	if !errors.As(err, &resp) || !errors.Is(err, ErrThrottle) {
		t.Fatalf("err = %v", err)
	}
	if resp.RetryAfter != 12*time.Second {
		t.Fatalf("retry = %s", resp.RetryAfter)
	}
}