package activeflights

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/webhooks/discord"
)

func TestBuildPostsPagination(t *testing.T) {
	at := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	loaded := make([]flights.Record, 26)
	for i := range loaded {
		loaded[i] = flights.Record{
			Raw: flights.Observation{Callsign: fmt.Sprintf("F%02d", i), Altitude: 1000, Speed: 200},
		}
	}

	posts := BuildPosts(SnapshotInput{
		At: at, Servers: []string{"expert"}, Matched: 26, Reported: 26,
		Frequency: "30m", Filters: json.RawMessage(`{}`), Flights: loaded,
	})
	if len(posts) != 3 {
		t.Fatalf("posts = %d, want summary + 2 flight pages", len(posts))
	}
	if posts[0].Title != "Active Flights — Summary" {
		t.Fatalf("summary title = %q", posts[0].Title)
	}
	if len(posts[1].Fields) != 25 {
		t.Fatalf("page 1 fields = %d", len(posts[1].Fields))
	}
	if len(posts[2].Fields) != 1 {
		t.Fatalf("page 2 fields = %d", len(posts[2].Fields))
	}
	if !strings.Contains(posts[2].Description, "Page 2 of 2") {
		t.Fatalf("page 2 description = %q", posts[2].Description)
	}
}

func TestBuildPostsSummaryAndFlightField(t *testing.T) {
	username := "alice"
	at := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	posts := BuildPosts(SnapshotInput{
		At:        at,
		Servers:   []string{"expert", "casual"},
		Matched:   120,
		Reported:  100,
		Capped:    true,
		Frequency: "30m",
		Filters:   json.RawMessage(`{"vaCode":"TEST","pilotState":["active"]}`),
		Flights: []flights.Record{{
			Raw: flights.Observation{
				Callsign:    "IE123",
				Username:    &username,
				UserID:      "u1",
				Altitude:    35000,
				Speed:       450,
				IsConnected: true,
			},
			Derived: flights.Derived{
				AircraftName: "A320",
				LiveryName:   "House",
				Normalized:   flights.Normalized{Speed: "450 kts", PilotState: "active"},
			},
		}},
	})

	summary := posts[0]
	if summary.Title != "Active Flights — Summary" {
		t.Fatalf("title = %q", summary.Title)
	}
	summaryText := strings.Join(embedFieldText(summary.Fields), " ")
	for _, part := range []string{"120", "100 (capped at 100)", "expert, casual", "TEST", "Active", "30 min", "House (1)", "35000 ft", "450 kts", "1 / 1", "1"} {
		if !strings.Contains(summaryText, part) {
			t.Fatalf("summary missing %q in %q", part, summaryText)
		}
	}

	if len(posts) != 2 {
		t.Fatalf("posts = %d", len(posts))
	}
	value := posts[1].Fields[0].Value
	for _, line := range []string{"Pilot: alice", "State: active", "Aircraft: A320 / House", "Altitude: 35000 ft", "Speed: 450 kts", "Route: —"} {
		if !strings.Contains(value, line) {
			t.Fatalf("flight value missing %q:\n%s", line, value)
		}
	}
	if posts[1].Footer == nil || posts[1].Footer.Text != "Flight Monitor | expert, casual" {
		t.Fatalf("footer = %+v", posts[1].Footer)
	}
}

func TestMostFrequentKeyTieBreak(t *testing.T) {
	got := mostFrequentKey(map[string]int{"Bravo": 2, "Alpha": 2})
	if got != "Alpha (2)" {
		t.Fatalf("got %q", got)
	}
}

func embedFieldText(fields []discord.EmbedField) []string {
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = field.Name + " " + field.Value
	}
	return out
}
