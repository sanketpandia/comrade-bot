package activeflights

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestPackMessagesSplitsOnFlightBoundaries(t *testing.T) {
	summary := "Active flights · 2026-10-20T14:30:00Z"
	shortA := strings.Repeat("a", 900)
	shortB := strings.Repeat("b", 900)
	wideA := strings.Repeat("c", 1500)
	wideB := strings.Repeat("d", 1500)
	exactA := strings.Repeat("e", 1000)
	exactB := strings.Repeat("f", 1000)

	packedShort := PackMessages(summary, []string{shortA, shortB})
	if len(packedShort) != 2 {
		t.Fatalf("short pack len = %d, want summary + one flight message", len(packedShort))
	}
	if !strings.Contains(packedShort[1], shortA) || !strings.Contains(packedShort[1], shortB) {
		t.Fatal("short blocks should share one message")
	}

	packedWide := PackMessages(summary, []string{wideA, wideB})
	if len(packedWide) != 3 {
		t.Fatalf("wide pack len = %d, want summary + two flight messages", len(packedWide))
	}
	if packedWide[1] != wideA || packedWide[2] != wideB {
		t.Fatal("wide blocks should each occupy their own message")
	}

	packedExact := PackMessages(summary, []string{exactA, exactB})
	if len(packedExact) != 3 {
		t.Fatalf("boundary pack len = %d, want a new message when the next block would pass 2000", len(packedExact))
	}

	for _, messages := range [][]string{packedShort, packedWide, packedExact} {
		if messages[0] != summary {
			t.Fatalf("summary = %q", messages[0])
		}
		for _, message := range messages {
			if utf8.RuneCountInString(message) > discordContentMax {
				t.Fatalf("message length %d exceeds %d", utf8.RuneCountInString(message), discordContentMax)
			}
		}
	}
}

func TestFormatFlightAndSummary(t *testing.T) {
	username := "alice"
	block := FormatFlight(flights.Flight{
		Callsign:     "IE123",
		Username:     &username,
		AircraftName: "A320",
		LiveryName:   "House",
		Altitude:     35000,
		Normalized:   flights.Normalized{Speed: "450 kts"},
	})
	wantLines := []string{
		"IE123",
		"\tPilot: alice",
		"\tAircraft: A320 / House",
		"\tAltitude: 35000 ft",
		"\tSpeed: 450 kts",
		"\tRoute: —",
	}
	for _, line := range wantLines {
		if !strings.Contains(block, line) {
			t.Fatalf("block missing %q:\n%s", line, block)
		}
	}

	at := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	summary := FormatSummary(at, []serverCount{{Name: "expert", Matched: 80}, {Name: "casual", Matched: 40}}, 120, 100, true)
	for _, part := range []string{
		"Active flights · 2026-10-20T14:30:00Z",
		"Servers: expert, casual",
		"Matched: 120 · Reported: 100 (capped at 100)",
		"\texpert: 80",
		"\tcasual: 40",
	} {
		if !strings.Contains(summary, part) {
			t.Fatalf("summary missing %q:\n%s", part, summary)
		}
	}
}
