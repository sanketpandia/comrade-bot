package activeflights

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/discord"
)

const (
	maxReportedFlights = 100
	maxFlightsPerEmbed = 25
	embedColor         = 0x4A5568
)

// SnapshotInput is the data needed to build Discord embed posts for one dispatch.
type SnapshotInput struct {
	At        time.Time
	Servers   []string
	Matched   int
	Reported  int
	Capped    bool
	Frequency string
	Filters   json.RawMessage
	Flights   []flights.Record
}

type filterDisplay struct {
	VACode      string
	PilotStates []string
}

type fleetStats struct {
	mostUsedLivery string
	avgAltitude    *int
	avgSpeed       *int
	connected      int
	total          int
	uniquePilots   int
}

// BuildPosts returns one Discord embed per webhook post: summary first, then flight pages.
func BuildPosts(in SnapshotInput) []discord.Embed {
	at := in.At.UTC()
	serverLabel := formatServerList(in.Servers)
	filter := parseFilterDisplay(in.Filters)
	stats := computeFleetStats(in.Flights)

	pages := paginateFlights(in.Flights)
	posts := []discord.Embed{buildSummaryEmbed(at, serverLabel, in, filter, stats)}
	for i, page := range pages {
		posts = append(posts, buildFlightPageEmbed(at, serverLabel, page, i+1, len(pages)))
	}
	return posts
}

func buildSummaryEmbed(at time.Time, serverLabel string, in SnapshotInput, filter filterDisplay, stats fleetStats) discord.Embed {
	reportedValue := fmt.Sprintf("%d", in.Reported)
	if in.Capped {
		reportedValue = fmt.Sprintf("%d (capped at %d)", in.Reported, maxReportedFlights)
	}

	vaLabel := strings.TrimSpace(filter.VACode)
	if vaLabel == "" {
		vaLabel = "Any"
	} else {
		vaLabel = strings.ToUpper(vaLabel)
	}

	fields := []discord.EmbedField{
		{Name: "Matched", Value: fmt.Sprintf("%d", in.Matched), Inline: true},
		{Name: "Reported", Value: reportedValue, Inline: true},
		{Name: "Server", Value: serverLabel, Inline: true},
		{Name: "Virtual Airline", Value: vaLabel, Inline: true},
		{Name: "Pilot State Filter", Value: formatPilotStateFilter(filter.PilotStates), Inline: true},
		{Name: "Frequency", Value: webhookconfig.FormatFrequencyDisplay(in.Frequency), Inline: true},
	}

	if stats.mostUsedLivery != "" {
		fields = append(fields, discord.EmbedField{Name: "Most-Used Livery", Value: stats.mostUsedLivery, Inline: true})
	}
	if stats.avgAltitude != nil {
		fields = append(fields, discord.EmbedField{Name: "Average Altitude", Value: fmt.Sprintf("%d ft", *stats.avgAltitude), Inline: true})
	}
	if stats.avgSpeed != nil {
		fields = append(fields, discord.EmbedField{Name: "Average Speed", Value: fmt.Sprintf("%d kts", *stats.avgSpeed), Inline: true})
	}
	if stats.total > 0 {
		fields = append(fields, discord.EmbedField{
			Name:   "Connected Aircraft",
			Value:  fmt.Sprintf("%d / %d", stats.connected, stats.total),
			Inline: true,
		})
	}
	if stats.uniquePilots > 0 {
		fields = append(fields, discord.EmbedField{Name: "Unique Pilots", Value: fmt.Sprintf("%d", stats.uniquePilots), Inline: true})
	}

	return discord.Embed{
		Title:       "Active Flights — Summary",
		Description: formatSnapshotLine(at),
		Color:       embedColor,
		Timestamp:   at.Format(time.RFC3339),
		Fields:      fields,
	}
}

func buildFlightPageEmbed(at time.Time, serverLabel string, page []flights.Record, pageNum, totalPages int) discord.Embed {
	desc := formatSnapshotLine(at)
	if totalPages > 1 {
		desc += fmt.Sprintf("\nPage %d of %d", pageNum, totalPages)
	}

	fields := make([]discord.EmbedField, len(page))
	for i, flight := range page {
		fields[i] = discord.EmbedField{
			Name:   flightFieldName(flight),
			Value:  flightFieldValue(flight),
			Inline: false,
		}
	}

	return discord.Embed{
		Title:       "Active Flights",
		Description: desc,
		Color:       embedColor,
		Timestamp:   at.Format(time.RFC3339),
		Fields:      fields,
		Footer:      &discord.EmbedFooter{Text: "Flight Monitor | " + serverLabel},
	}
}

func formatSnapshotLine(at time.Time) string {
	return "Snapshot: " + at.UTC().Format("2006-01-02 15:04:05 UTC")
}

func formatServerList(servers []string) string {
	if len(servers) == 0 {
		return "—"
	}
	return strings.Join(servers, ", ")
}

func parseFilterDisplay(raw json.RawMessage) filterDisplay {
	var out filterDisplay
	if len(strings.TrimSpace(string(raw))) == 0 || string(raw) == "null" {
		return out
	}
	var filters struct {
		PilotState []string `json:"pilotState"`
		VACode     string   `json:"vaCode"`
	}
	if err := json.Unmarshal(raw, &filters); err != nil {
		return out
	}
	out.VACode = strings.TrimSpace(filters.VACode)
	out.PilotStates = filters.PilotState
	return out
}

func formatPilotStateFilter(states []string) string {
	if len(states) == 0 {
		return "Any"
	}
	labels := make([]string, 0, len(states))
	for _, state := range states {
		label := formatPilotStateLabel(state)
		if label != "" {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return "Any"
	}
	sort.Strings(labels)
	return strings.Join(labels, ", ")
}

func formatPilotStateLabel(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	words := strings.Fields(strings.ReplaceAll(name, "_", " "))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

func computeFleetStats(reported []flights.Record) fleetStats {
	stats := fleetStats{total: len(reported)}
	if len(reported) == 0 {
		return stats
	}

	liveryCounts := map[string]int{}
	altSum, altCount := 0, 0
	speedSum, speedCount := 0, 0
	pilots := map[string]struct{}{}

	for _, flight := range reported {
		livery := strings.TrimSpace(flight.Derived.LiveryName)
		if livery != "" {
			liveryCounts[livery]++
		}
		altSum += flights.RoundedAltitude(flight.Raw.Altitude)
		altCount++
		speedSum += flights.RoundedSpeed(flight.Raw.Speed)
		speedCount++

		if flight.Raw.IsConnected {
			stats.connected++
		}
		if key := pilotKey(flight); key != "" {
			pilots[key] = struct{}{}
		}
	}

	stats.mostUsedLivery = mostFrequentKey(liveryCounts)
	if altCount > 0 {
		avg := altSum / altCount
		stats.avgAltitude = &avg
	}
	if speedCount > 0 {
		avg := speedSum / speedCount
		stats.avgSpeed = &avg
	}
	stats.uniquePilots = len(pilots)
	return stats
}

func mostFrequentKey(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	bestName := ""
	bestCount := -1
	for name, count := range counts {
		if count > bestCount || (count == bestCount && (bestName == "" || name < bestName)) {
			bestName = name
			bestCount = count
		}
	}
	if bestCount <= 0 {
		return ""
	}
	return fmt.Sprintf("%s (%d)", bestName, bestCount)
}

func pilotKey(flight flights.Record) string {
	if id := strings.TrimSpace(flight.Raw.UserID); id != "" {
		return "id:" + id
	}
	if name := strings.TrimSpace(flights.Username(flight.Raw)); name != "" {
		return "name:" + strings.ToLower(name)
	}
	return ""
}

func paginateFlights(reported []flights.Record) [][]flights.Record {
	if len(reported) == 0 {
		return nil
	}
	var pages [][]flights.Record
	for i := 0; i < len(reported); i += maxFlightsPerEmbed {
		end := i + maxFlightsPerEmbed
		if end > len(reported) {
			end = len(reported)
		}
		pages = append(pages, reported[i:end])
	}
	return pages
}

func flightFieldName(flight flights.Record) string {
	callsign := strings.TrimSpace(flight.Raw.Callsign)
	if callsign == "" {
		return "—"
	}
	return callsign
}

func flightFieldValue(flight flights.Record) string {
	pilot := strings.TrimSpace(flights.Username(flight.Raw))
	if pilot == "" {
		pilot = "—"
	}
	speed := strings.TrimSpace(flight.Derived.Normalized.Speed)
	if speed == "" {
		speed = fmt.Sprintf("%d kts", flights.RoundedSpeed(flight.Raw.Speed))
	}
	flightID := strings.TrimSpace(flight.Raw.FlightID)
	if flightID == "" {
		flightID = "—"
	}
	return fmt.Sprintf(
		"Pilot: %s\nState: %s\nAircraft: %s\nAltitude: %d ft\nSpeed: %s\nRoute: —\nFlight ID: %s",
		pilot,
		formatFlightState(flight),
		formatAircraft(flight.Derived.AircraftName, flight.Derived.LiveryName),
		flights.RoundedAltitude(flight.Raw.Altitude),
		speed,
		flightID,
	)
}

func formatFlightState(flight flights.Record) string {
	state := strings.TrimSpace(flight.Derived.Normalized.PilotState)
	if state == "" {
		state = flights.NameForPilotState(flight.Raw.PilotState)
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return "—"
	}
	return state
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
