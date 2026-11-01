package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type liveAircraftFleetView struct {
	OrganizationID string
	ErrorMessage   string
	CountLabel     string
	Rows           []liveAircraftRowView
}

type liveAircraftRowView struct {
	ID              string
	Registration    string
	AircraftName    string
	StatusLabel     string
	StatusClass     string
	VisibilityLabel string
	VisibilityClass string
	SlotLabel       string
	SlotClass       string
	FleetRank       string
	CreatedDisplay  string
	CreatedISO      string
	CreatedRelative string
}

func liveAircraftFleetViewFromUpstream(orgID string, rows []infiniteflight.LiveAircraft, lookup *gameliveries.Lookup) liveAircraftFleetView {
	slices.SortFunc(rows, func(a, b infiniteflight.LiveAircraft) int {
		return a.FleetRank - b.FleetRank
	})
	view := liveAircraftFleetView{OrganizationID: orgID}
	switch len(rows) {
	case 0:
		view.CountLabel = "No aircraft"
	case 1:
		view.CountLabel = "1 aircraft"
	default:
		view.CountLabel = fmt.Sprintf("%d aircraft", len(rows))
	}
	for _, row := range rows {
		name := "—"
		if lookup != nil {
			if resolved, outcome := lookup.Resolve(row.AircraftID, row.AircraftID); outcome != gameliveries.MatchMiss {
				if resolved.AircraftName != "" {
					name = resolved.AircraftName
				}
			}
		}
		statusLabel, statusClass := liveAircraftStatusBadge(row.Status)
		visLabel, visClass := liveAircraftVisibilityBadge(row.Visibility)
		slotLabel, slotClass := liveAircraftSlotBadge(row.IsFleetActiveSlot)
		createdDisplay, createdISO, createdRelative := formatLiveAircraftCreated(row.CreatedAt)
		view.Rows = append(view.Rows, liveAircraftRowView{
			ID:              row.ID,
			Registration:    displayOrDash(row.Registration),
			AircraftName:    name,
			StatusLabel:     statusLabel,
			StatusClass:     statusClass,
			VisibilityLabel: visLabel,
			VisibilityClass: visClass,
			SlotLabel:       slotLabel,
			SlotClass:       slotClass,
			FleetRank:       fmt.Sprintf("%d", row.FleetRank),
			CreatedDisplay:  createdDisplay,
			CreatedISO:      createdISO,
			CreatedRelative: createdRelative,
		})
	}
	return view
}

func displayOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func liveAircraftStatusBadge(status int) (label, class string) {
	switch status {
	case 0:
		return "Active", "badge--success"
	case 1:
		return "Deleted", "badge--error"
	default:
		return fmt.Sprintf("Status %d", status), "badge--muted"
	}
}

func liveAircraftVisibilityBadge(visibility int) (label, class string) {
	switch visibility {
	case 1:
		return "Visible", "badge--success"
	case 2:
		return "Hangared", "badge--muted"
	case 0:
		return "Unknown", "badge--muted"
	default:
		return fmt.Sprintf("Visibility %d", visibility), "badge--muted"
	}
}

func liveAircraftSlotBadge(active bool) (label, class string) {
	if active {
		return "Active slot", "badge--success"
	}
	return "Storage", "badge--muted"
}

func formatLiveAircraftCreated(at time.Time) (display, iso, relative string) {
	if at.IsZero() {
		return "—", "", ""
	}
	utc := at.UTC()
	display = utc.Format("2006-01-02 15:04:05 UTC")
	iso = utc.Format(time.RFC3339)
	relative = relativeTime(utc)
	return display, iso, relative
}
