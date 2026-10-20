package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPostEmbedsSuccess(t *testing.T) {
	var got webhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	reg := metrics.NewRegistry()
	client := NewClient(reg)
	embed := Embed{Title: "Active Flights", Color: 1}
	if err := client.PostEmbeds(context.Background(), "active_flights", server.URL, []Embed{embed}); err != nil {
		t.Fatalf("PostEmbeds() = %v", err)
	}
	if len(got.Embeds) != 1 || got.Embeds[0].Title != "Active Flights" {
		t.Fatalf("embeds = %+v", got.Embeds)
	}
	if n := testutil.ToFloat64(reg.WebhookDiscordPosts.WithLabelValues("active_flights", "success")); n != 1 {
		t.Fatalf("success posts = %v", n)
	}
}

func TestPostEmbedsTransportErrorOmitsURL(t *testing.T) {
	reg := metrics.NewRegistry()
	client := NewClient(reg)
	webhookURL := "http://127.0.0.1:1/api/webhooks/1/super-secret-token"
	err := client.PostEmbeds(context.Background(), "active_flights", webhookURL, []Embed{{Title: "x"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "super-secret-token") || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("error leaked url: %v", err)
	}
}

func TestPostEmbedsErrorOmitsURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer server.Close()

	reg := metrics.NewRegistry()
	client := NewClient(reg)
	webhookURL := server.URL + "/api/webhooks/1/super-secret-token"
	err := client.PostEmbeds(context.Background(), "active_flights", webhookURL, []Embed{{Title: "x"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "super-secret-token") || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("error leaked url: %v", err)
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("error = %v", err)
	}
	if n := testutil.ToFloat64(reg.WebhookDiscordPosts.WithLabelValues("active_flights", "error")); n != 1 {
		t.Fatalf("error posts = %v", n)
	}
}
