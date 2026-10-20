package activeflights

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"infinite-experiment/politburo/internal/livegame/flights"
)

const (
	maxReportedFlights = 100
	discordContentMax  = 2000
)

type serverCount struct {
	Name    string
	Matched int
}

func FormatSummary(at time.Time, counts []serverCount, matched, reported int, capped bool) string {
	at = at.UTC()
	var b strings.Builder
	fmt.Fprintf(&b, "Active flights · %s\n", at.Format(time.RFC3339))
	if len(counts) == 0 {
		b.WriteString("Servers: —\n")
	} else {
		names := make([]string, len(counts))
		for i, count := range counts {
			names[i] = count.Name
		}
		fmt.Fprintf(&b, "Servers: %s\n", strings.Join(names, ", "))
	}
	if capped {
		fmt.Fprintf(&b, "Matched: %d · Reported: %d (capped at %d)\n", matched, reported, maxReportedFlights)
	} else {
		fmt.Fprintf(&b, "Matched: %d · Reported: %d\n", matched, reported)
	}
	for _, count := range counts {
		fmt.Fprintf(&b, "\t%s: %d\n", count.Name, count.Matched)
	}
	return strings.TrimRight(b.String(), "\n")
}

func FormatFlight(flight flights.Flight) string {
	callsign := strings.TrimSpace(flight.Callsign)
	if callsign == "" {
		callsign = "—"
	}
	pilot := strings.TrimSpace(flights.Username(flight))
	if pilot == "" {
		pilot = "—"
	}
	speed := strings.TrimSpace(flight.Normalized.Speed)
	if speed == "" {
		speed = fmt.Sprintf("%d kts", flight.Speed)
	}
	return fmt.Sprintf("%s\n\tPilot: %s\n\tAircraft: %s\n\tAltitude: %d ft\n\tSpeed: %s\n\tRoute: —",
		callsign, pilot, formatAircraft(flight.AircraftName, flight.LiveryName), flight.Altitude, speed)
}

func formatAircraft(aircraft, livery string) string {
	aircraft = strings.TrimSpace(aircraft)
	livery = strings.TrimSpace(livery)
	switch {
	case aircraft != "" && livery != "":
		return aircraft + " / " + livery
	case aircraft != "":
		return aircraft
	case livery != "":
		return livery
	default:
		return "—"
	}
}

// PackMessages keeps the summary as the first Discord message and packs flight
// blocks into later messages without splitting a block across messages.
func PackMessages(summary string, blocks []string) []string {
	messages := []string{clip(summary, discordContentMax)}
	var current strings.Builder
	for _, block := range blocks {
		block = clip(block, discordContentMax)
		if block == "" {
			continue
		}
		if current.Len() == 0 {
			current.WriteString(block)
			continue
		}
		if runeLen(current.String())+2+runeLen(block) > discordContentMax {
			messages = append(messages, current.String())
			current.Reset()
			current.WriteString(block)
			continue
		}
		current.WriteString("\n\n")
		current.WriteString(block)
	}
	if current.Len() > 0 {
		messages = append(messages, current.String())
	}
	return messages
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
}

func clip(value string, maxRunes int) string {
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	return string([]rune(value)[:maxRunes])
}
