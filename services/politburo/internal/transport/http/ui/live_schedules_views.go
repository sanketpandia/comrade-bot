package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type liveSchedulesView struct {
	OrganizationID   string
	FilterAircraftID string
	ErrorMessage     string
	CountLabel       string
	TruncatedNote    string
	FilterOptions    []liveScheduleFilterOption
	Rows             []liveScheduleRowView
}

type liveScheduleFilterOption struct {
	Value    string
	Label    string
	Selected bool
}

type liveScheduleRowView struct {
	ScheduleID       string
	AircraftID       string
	Registration     string
	AircraftName     string
	Callsign         string
	Route            string
	StatusLabel      string
	StatusClass      string
	DepartureDisplay string
	DepartureISO     string
	DepartureRelative string
	IsTimeless       bool
}

type liveAircraftDisplay struct {
	Registration string
	AircraftName string
}

func liveSchedulesViewFromUpstream(
	orgID string,
	filterAircraftID string,
	aircraft []infiniteflight.LiveAircraft,
	schedules []infiniteflight.LiveSchedule,
	truncated bool,
	lookup *gameliveries.Lookup,
) liveSchedulesView {
	view := liveSchedulesView{
		OrganizationID:   orgID,
		FilterAircraftID: filterAircraftID,
	}
	if truncated {
		view.TruncatedNote = "Showing the first 1000 schedules. Narrow the aircraft filter for more."
	}

	byID := make(map[string]liveAircraftDisplay, len(aircraft))
	sortedAircraft := slices.Clone(aircraft)
	slices.SortFunc(sortedAircraft, func(a, b infiniteflight.LiveAircraft) int {
		return a.FleetRank - b.FleetRank
	})
	for _, row := range sortedAircraft {
		name := "—"
		if lookup != nil {
			if resolved, outcome := lookup.Resolve(row.AircraftID, row.AircraftID); outcome != gameliveries.MatchMiss {
				if resolved.AircraftName != "" {
					name = resolved.AircraftName
				}
			}
		}
		byID[row.ID] = liveAircraftDisplay{
			Registration: displayOrDash(row.Registration),
			AircraftName: name,
		}
	}

	view.FilterOptions = append(view.FilterOptions, liveScheduleFilterOption{
		Value:    "",
		Label:    "All aircraft",
		Selected: filterAircraftID == "",
	})
	knownFilter := false
	for _, row := range sortedAircraft {
		display := byID[row.ID]
		label := fmt.Sprintf("%s · %s", display.Registration, display.AircraftName)
		selected := filterAircraftID != "" && row.ID == filterAircraftID
		if selected {
			knownFilter = true
		}
		view.FilterOptions = append(view.FilterOptions, liveScheduleFilterOption{
			Value:    row.ID,
			Label:    label,
			Selected: selected,
		})
	}
	if filterAircraftID != "" && !knownFilter {
		view.FilterOptions = append(view.FilterOptions, liveScheduleFilterOption{
			Value:    filterAircraftID,
			Label:    filterAircraftID,
			Selected: true,
		})
	}

	count := len(schedules)
	switch count {
	case 0:
		view.CountLabel = "No schedules"
	case 1:
		view.CountLabel = "1 schedule"
	default:
		view.CountLabel = fmt.Sprintf("%d schedules", count)
	}

	for _, sched := range schedules {
		display := byID[sched.AircraftID]
		reg := display.Registration
		name := display.AircraftName
		if reg == "" {
			reg = "—"
		}
		if name == "" {
			name = "—"
		}
		statusLabel, statusClass := liveScheduleStatusBadge(sched.Status)
		depDisplay, depISO, depRelative := formatLiveScheduleDeparture(sched.IsTimeless, sched.ScheduledDepartureUtc)
		route := formatScheduleRoute(sched.OriginIcao, sched.DestinationIcao)
		view.Rows = append(view.Rows, liveScheduleRowView{
			ScheduleID:        sched.ID,
			AircraftID:        sched.AircraftID,
			Registration:      reg,
			AircraftName:      name,
			Callsign:          displayOrDash(sched.Callsign),
			Route:             route,
			StatusLabel:       statusLabel,
			StatusClass:       statusClass,
			DepartureDisplay:  depDisplay,
			DepartureISO:      depISO,
			DepartureRelative: depRelative,
			IsTimeless:        sched.IsTimeless,
		})
	}
	return view
}

func formatScheduleRoute(origin, dest string) string {
	origin = strings.TrimSpace(origin)
	dest = strings.TrimSpace(dest)
	if origin == "" {
		origin = "—"
	}
	if dest == "" {
		dest = "—"
	}
	return origin + " → " + dest
}

func liveScheduleStatusBadge(status int) (label, class string) {
	switch status {
	case 0:
		return "Unknown", "badge--muted"
	case 1:
		return "Scheduled", "badge--success"
	case 2:
		return "Boarding", "badge--success"
	case 3:
		return "Boarded", "badge--success"
	case 4:
		return "Taxiing to runway", "badge--muted"
	case 6:
		return "In flight", "badge--success"
	case 7:
		return "Diverted", "badge--error"
	case 8:
		return "Delayed", "badge--error"
	case 9:
		return "Cancelled", "badge--error"
	case 10:
		return "Taxiing to parking", "badge--muted"
	case 11:
		return "Arrived", "badge--muted"
	default:
		return fmt.Sprintf("Status %d", status), "badge--muted"
	}
}

func formatLiveScheduleDeparture(timeless bool, at time.Time) (display, iso, relative string) {
	if timeless {
		return "Timeless", "", ""
	}
	return formatLiveAircraftCreated(at)
}
