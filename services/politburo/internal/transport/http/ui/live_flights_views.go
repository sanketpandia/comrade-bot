package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type liveFlightsView struct {
	OrganizationID   string
	FilterAircraftID string
	ErrorMessage     string
	PromptMessage    string
	CountLabel       string
	TruncatedNote    string
	FilterOptions    []liveFlightFilterOption
	Rows             []liveFlightRowView
}

type liveFlightFilterOption struct {
	Value    string
	Label    string
	Selected bool
}

type liveFlightRowView struct {
	FlightID           string
	AircraftID         string
	Callsign           string
	Route              string
	StatusLabel        string
	StatusClass        string
	DepartureDisplay   string
	DepartureISO       string
	DepartureRelative  string
	ArrivalDisplay     string
	ArrivalISO         string
	ArrivalRelative    string
	FlightTimeDisplay  string
	CrewDisplay        string
}

func liveFlightsViewFromUpstream(
	orgID string,
	filterAircraftID string,
	aircraft []infiniteflight.LiveAircraft,
	flights []infiniteflight.LiveAircraftFlight,
	truncated bool,
	lookup *gameliveries.Lookup,
) liveFlightsView {
	view := liveFlightsView{
		OrganizationID:   orgID,
		FilterAircraftID: filterAircraftID,
	}
	if truncated {
		view.TruncatedNote = "Showing the first 1000 flights."
	}

	sortedAircraft := slices.Clone(aircraft)
	slices.SortFunc(sortedAircraft, func(a, b infiniteflight.LiveAircraft) int {
		return a.FleetRank - b.FleetRank
	})

	view.FilterOptions = append(view.FilterOptions, liveFlightFilterOption{
		Value:    "",
		Label:    "Select an aircraft",
		Selected: filterAircraftID == "",
	})
	knownFilter := false
	for _, row := range sortedAircraft {
		name := "—"
		if lookup != nil {
			if resolved, outcome := lookup.Resolve(row.AircraftID, row.AircraftID); outcome != gameliveries.MatchMiss {
				if resolved.AircraftName != "" {
					name = resolved.AircraftName
				}
			}
		}
		reg := displayOrDash(row.Registration)
		label := fmt.Sprintf("%s · %s", reg, name)
		selected := filterAircraftID != "" && row.ID == filterAircraftID
		if selected {
			knownFilter = true
		}
		view.FilterOptions = append(view.FilterOptions, liveFlightFilterOption{
			Value:    row.ID,
			Label:    label,
			Selected: selected,
		})
	}
	if filterAircraftID != "" && !knownFilter {
		view.FilterOptions = append(view.FilterOptions, liveFlightFilterOption{
			Value:    filterAircraftID,
			Label:    filterAircraftID,
			Selected: true,
		})
	}

	if filterAircraftID == "" {
		view.PromptMessage = "Select an aircraft to load flight history."
		return view
	}

	count := len(flights)
	switch count {
	case 0:
		view.CountLabel = "No flights"
	case 1:
		view.CountLabel = "1 flight"
	default:
		view.CountLabel = fmt.Sprintf("%d flights", count)
	}

	for _, flight := range flights {
		statusLabel, statusClass := liveAircraftFlightStatusBadge(flight.Status)
		depDisplay, depISO, depRelative := formatOptionalLiveTime(flight.ActualDepartureUtc)
		arrDisplay, arrISO, arrRelative := formatOptionalLiveTime(flight.ActualArrivalUtc)
		view.Rows = append(view.Rows, liveFlightRowView{
			FlightID:          flight.ID,
			AircraftID:        flight.AircraftID,
			Callsign:          displayOrDash(flight.Callsign),
			Route:             formatFlightRoute(flight.OriginIcao, flight.DestinationIcao),
			StatusLabel:       statusLabel,
			StatusClass:       statusClass,
			DepartureDisplay:  depDisplay,
			DepartureISO:      depISO,
			DepartureRelative: depRelative,
			ArrivalDisplay:    arrDisplay,
			ArrivalISO:        arrISO,
			ArrivalRelative:   arrRelative,
			FlightTimeDisplay: formatFlightTimeMinutes(flight.FlightTimeMinutes),
			CrewDisplay:       formatFlightCrew(flight.Crew),
		})
	}
	return view
}

func formatFlightRoute(origin, dest *string) string {
	var o, d string
	if origin != nil {
		o = *origin
	}
	if dest != nil {
		d = *dest
	}
	return formatScheduleRoute(o, d)
}

func formatOptionalLiveTime(at *time.Time) (display, iso, relative string) {
	if at == nil || at.IsZero() {
		return "—", "", ""
	}
	return formatLiveAircraftCreated(*at)
}

func formatFlightTimeMinutes(minutes float64) string {
	total := int(math.Round(minutes))
	if total < 0 {
		total = 0
	}
	h := total / 60
	m := total % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func formatFlightCrew(crew []infiniteflight.LiveAircraftFlightCrewMember) string {
	if len(crew) == 0 {
		return "—"
	}
	names := make([]string, 0, len(crew))
	for _, member := range crew {
		if member.Username != nil && strings.TrimSpace(*member.Username) != "" {
			names = append(names, strings.TrimSpace(*member.Username))
			continue
		}
		if strings.TrimSpace(member.UserID) != "" {
			names = append(names, member.UserID)
		}
	}
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, ", ")
}

func liveAircraftFlightStatusBadge(status int) (label, class string) {
	switch status {
	case 0:
		return "Active", "badge--success"
	case 1:
		return "Completed", "badge--muted"
	case 2:
		return "Cancelled", "badge--error"
	default:
		return fmt.Sprintf("Status %d", status), "badge--muted"
	}
}
