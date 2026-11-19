package airtable

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestClientDeniedKeepsErrorCodeOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("authorization header missing")
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"type":"INVALID_PERMISSIONS_OR_MODEL_NOT_FOUND","message":"secret-token was rejected"}}`))
	}))
	defer srv.Close()

	client := NewClient(nil, time.Second).WithBaseURL(srv.URL)
	_, err := client.FetchSchema(context.Background(), "secret-token", "appBase")
	var resp *ResponseError
	if !errors.As(err, &resp) || resp.Status != http.StatusForbidden || resp.Code != "INVALID_PERMISSIONS_OR_MODEL_NOT_FOUND" {
		t.Fatalf("err = %#v", resp)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked token: %v", err)
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
