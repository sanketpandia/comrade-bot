package activeflights

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/webhooks/discord"
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
	handler := New(&fakeFlights{byServer: byServer}, &fakeSessions{names: []string{"bravo", "alpha", "offline"}}, poster, reg)

	result := handler.Handle(context.Background(), dispatch.Item{
		VAID:             "va-1",
		WebhookType:      FeatureKey,
		URL:              "https://discord.com/api/webhooks/1/token",
		PreferredServers: []string{"bravo", "alpha"},
		Frequency:        "30m",
		Filters:          []byte(`{}`),
	})
	if result.Outcome != dispatch.OutcomeSuccess {
		t.Fatalf("outcome = %s err=%s", result.Outcome, result.ErrorMessage)
	}
	if len(poster.embeds) < 2 {
		t.Fatalf("expected summary plus flight pages, got %d posts", len(poster.embeds))
	}
	summary := poster.embeds[0]
	summaryText := strings.Join(embedFieldText(summary.Fields), " ")
	if !strings.Contains(summaryText, "110") || !strings.Contains(summaryText, "100 (capped at 100)") {
		t.Fatalf("summary missing cap note: %q", summaryText)
	}

	var callsigns []string
	for _, embed := range poster.embeds[1:] {
		for _, field := range embed.Fields {
			callsigns = append(callsigns, field.Name)
		}
	}
	body := strings.Join(callsigns, "\n")
	if !strings.Contains(body, "A59") || !strings.Contains(body, "B39") {
		t.Fatal("expected alpha 60 flights and the first 40 bravo flights")
	}
	if strings.Contains(body, "B40") {
		t.Fatal("B40 should be past the cap")
	}
	if result.DiscordMessages != len(poster.embeds) || result.DiscordMessages < 2 {
		t.Fatalf("discord messages = %d posts = %d", result.DiscordMessages, len(poster.embeds))
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
	handler := New(&fakeFlights{err: flights.ErrUnknownVACode}, &fakeSessions{names: []string{"expert"}}, poster, metrics.NewRegistry())
	result := handler.Handle(context.Background(), dispatch.Item{
		VAID:             "va-1",
		URL:              "https://discord.com/api/webhooks/1/token",
		PreferredServers: []string{"expert"},
		Filters:          []byte(`{"vaCode":"NOPE"}`),
	})
	if result.Outcome != dispatch.OutcomeError || len(poster.embeds) != 0 {
		t.Fatalf("result = %+v posts = %d", result, len(poster.embeds))
	}
}

func TestHandlePassesNormalizedVACodeToReader(t *testing.T) {
	flightsAPI := &fakeFlights{}
	handler := New(flightsAPI, &fakeSessions{names: []string{"expert"}}, &fakePoster{}, metrics.NewRegistry())
	handler.Handle(context.Background(), dispatch.Item{
		VAID:             "va-1",
		URL:              "https://discord.com/api/webhooks/1/token",
		PreferredServers: []string{"expert"},
		Filters:          []byte(`{"vaCode":" tst "}`),
	})
	if len(flightsAPI.vaCodes) != 1 || flightsAPI.vaCodes[0] != "TST" {
		t.Fatalf("vaCodes = %v", flightsAPI.vaCodes)
	}
}

func TestHandleSkipsWithoutPreferredServers(t *testing.T) {
	flightsAPI := &fakeFlights{}
	poster := &fakePoster{}
	reg := metrics.NewRegistry()
	handler := New(flightsAPI, &fakeSessions{}, poster, reg)

	result := handler.Handle(context.Background(), dispatch.Item{VAID: "va-1", Filters: []byte(`{}`)})
	if result.Outcome != dispatch.OutcomeSkipped || result.SkipReason != dispatch.SkipNoPreferredServers {
		t.Fatalf("result = %+v", result)
	}
	if flightsAPI.calls != 0 || len(poster.embeds) != 0 {
		t.Fatalf("calls = %d posts = %d", flightsAPI.calls, len(poster.embeds))
	}
	if got := testutil.ToFloat64(reg.WebhookActiveFlightsMatched.WithLabelValues(FeatureKey)); got != 0 {
		t.Fatalf("matched = %v", got)
	}
}

type fakeFlights struct {
	byServer map[string][]flights.Flight
	err      error
	calls    int
	vaCodes  []string
}

func (f *fakeFlights) ListActiveTracks(ctx context.Context, input flights.ListActiveTracksInput) (flights.ListActiveTracksResult, error) {
	f.calls++
	f.vaCodes = append(f.vaCodes, input.Filters.VACode)
	if f.err != nil {
		return flights.ListActiveTracksResult{}, f.err
	}
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
	embeds []discord.Embed
}

func (f *fakePoster) PostEmbeds(ctx context.Context, webhookType, webhookURL string, embeds []discord.Embed) error {
	if len(embeds) != 1 {
		return fmt.Errorf("expected one embed per post, got %d", len(embeds))
	}
	f.embeds = append(f.embeds, embeds[0])
	return nil
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
