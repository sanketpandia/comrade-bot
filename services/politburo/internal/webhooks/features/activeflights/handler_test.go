package activeflights

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/webhooks/dispatch"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHandleCapsAcrossServers(t *testing.T) {
	byServer := map[string][]flights.Flight{
		"alpha": sampleFlights("A", 60),
		"bravo": sampleFlights("B", 50),
	}
	poster := &fakePoster{}
	reg := metrics.NewRegistry()
	handler := New(&fakeFlights{byServer: byServer}, &fakeSessions{names: []string{"bravo", "alpha", "offline"}}, nil, poster, reg)

	result := handler.Handle(context.Background(), dispatch.Item{
		VAID:             "va-1",
		WebhookType:      FeatureKey,
		URL:              "https://discord.com/api/webhooks/1/token",
		PreferredServers: []string{"bravo", "alpha"},
		Filters:          []byte(`{}`),
	})
	if result.Outcome != dispatch.OutcomeSuccess {
		t.Fatalf("outcome = %s err=%s", result.Outcome, result.ErrorMessage)
	}
	body := strings.Join(poster.bodies, "\n")
	if !strings.Contains(body, "Matched: 110 · Reported: 100 (capped at 100)") {
		t.Fatalf("summary missing cap note:\n%s", poster.bodies[0])
	}
	if !strings.Contains(body, "A59") || !strings.Contains(body, "B39") || strings.Contains(body, "\nB40\n") || strings.HasPrefix(body, "B40\n") {
		t.Fatal("expected alpha 60 flights and the first 40 bravo flights")
	}
	if strings.Contains(body, "B40") {
		t.Fatal("B40 should be past the cap")
	}
	for _, message := range poster.bodies {
		if len([]rune(message)) > discordContentMax {
			t.Fatalf("posted message length %d", len([]rune(message)))
		}
	}
	if result.DiscordMessages != len(poster.bodies) || result.DiscordMessages < 2 {
		t.Fatalf("discord messages = %d posts = %d", result.DiscordMessages, len(poster.bodies))
	}
	if got := testutil.ToFloat64(reg.WebhookActiveFlightsMatched.WithLabelValues(FeatureKey)); got != 110 {
		t.Fatalf("matched = %v", got)
	}
	if got := testutil.ToFloat64(reg.WebhookActiveFlightsReported.WithLabelValues(FeatureKey)); got != 100 {
		t.Fatalf("reported = %v", got)
	}
	if got := testutil.ToFloat64(reg.WebhookActiveFlightsCapped.WithLabelValues(FeatureKey)); got != 1 {
		t.Fatalf("capped = %v", got)
	}
}

func TestHandleUnknownVACodeDoesNotPost(t *testing.T) {
	poster := &fakePoster{}
	handler := New(&fakeFlights{}, &fakeSessions{names: []string{"expert"}}, &fakeVAs{}, poster, metrics.NewRegistry())
	result := handler.Handle(context.Background(), dispatch.Item{
		VAID:             "va-1",
		URL:              "https://discord.com/api/webhooks/1/token",
		PreferredServers: []string{"expert"},
		Filters:          []byte(`{"vaCode":"NOPE"}`),
	})
	if result.Outcome != dispatch.OutcomeError || len(poster.bodies) != 0 {
		t.Fatalf("result = %+v posts = %d", result, len(poster.bodies))
	}
}

func TestHandleSkipsWithoutPreferredServers(t *testing.T) {
	flightsAPI := &fakeFlights{}
	poster := &fakePoster{}
	reg := metrics.NewRegistry()
	handler := New(flightsAPI, &fakeSessions{}, nil, poster, reg)

	result := handler.Handle(context.Background(), dispatch.Item{VAID: "va-1", Filters: []byte(`{}`)})
	if result.Outcome != dispatch.OutcomeSkipped || result.SkipReason != dispatch.SkipNoPreferredServers {
		t.Fatalf("result = %+v", result)
	}
	if flightsAPI.calls != 0 || len(poster.bodies) != 0 {
		t.Fatalf("calls = %d posts = %d", flightsAPI.calls, len(poster.bodies))
	}
	if got := testutil.ToFloat64(reg.WebhookActiveFlightsMatched.WithLabelValues(FeatureKey)); got != 0 {
		t.Fatalf("matched = %v", got)
	}
}

type fakeFlights struct {
	byServer map[string][]flights.Flight
	calls    int
}

func (f *fakeFlights) ListActiveTracks(ctx context.Context, input flights.ListActiveTracksInput) (flights.ListActiveTracksResult, error) {
	f.calls++
	all := f.byServer[input.Server]
	page := all
	if input.PageLength > 0 && len(page) > input.PageLength {
		page = page[:input.PageLength]
	}
	tracks := make([]flights.Track, len(page))
	for i, flight := range page {
		tracks[i] = flights.TrackFromFlight(flight)
	}
	return flights.ListActiveTracksResult{Tracks: tracks, TotalLength: len(all)}, nil
}

func (f *fakeFlights) GetActiveFlight(ctx context.Context, flightID string) (flights.Flight, error) {
	for _, list := range f.byServer {
		for _, flight := range list {
			if flight.FlightID == flightID {
				return flight, nil
			}
		}
	}
	return flights.Flight{}, flights.ErrCacheMiss
}

type fakeSessions struct {
	names []string
}

func (f *fakeSessions) ListActive(ctx context.Context) (sessions.ListActiveResult, error) {
	out := make([]sessions.ActiveSession, len(f.names))
	for i, name := range f.names {
		out[i] = sessions.ActiveSession{NormalizedName: name}
	}
	return sessions.ListActiveResult{Sessions: out}, nil
}

type fakePoster struct {
	bodies []string
}

func (f *fakePoster) PostContent(ctx context.Context, webhookType, webhookURL, content string) error {
	f.bodies = append(f.bodies, content)
	return nil
}

type fakeVAs struct {
	id string
}

func (f *fakeVAs) GetByCode(ctx context.Context, code string) (*virtualairlines.VirtualAirline, error) {
	if f.id == "" {
		return nil, nil
	}
	return &virtualairlines.VirtualAirline{ID: f.id, Code: code}, nil
}

func sampleFlights(prefix string, n int) []flights.Flight {
	out := make([]flights.Flight, n)
	for i := 0; i < n; i++ {
		out[i] = flights.Flight{
			FlightID:     fmt.Sprintf("%s-%03d", prefix, i),
			Callsign:     fmt.Sprintf("%s%02d", prefix, i),
			AircraftName: "A320",
			Altitude:     1000 + i,
			Speed:        200,
		}
	}
	return out
}
