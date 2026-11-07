package flights

import (
	"fmt"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

// NormalizeRecord builds the full serve-time display envelope from a cached record.
func NormalizeRecord(record Record, lookup *gameliveries.Lookup) Normalized {
	out := Normalize(record.Raw, lookup)
	out.Departure = record.Normalized.Departure
	out.Destination = record.Normalized.Destination
	return out
}

// Normalize builds serve-time display strings from an observation and the livery catalog.
func Normalize(raw Observation, lookup *gameliveries.Lookup) Normalized {
	speed := RoundedSpeed(raw.Speed)
	verticalSpeed := RoundedVerticalSpeed(raw.VerticalSpeed)
	connected := "disconnected"
	if raw.IsConnected {
		connected = "connected"
	}
	out := Normalized{
		NormalizedCallsign: NormalizedCallsignForAffix(raw.Callsign),
		Speed:              fmt.Sprintf("%d kts", speed),
		VerticalSpeed:      fmt.Sprintf("%.1f ft/min", verticalSpeed),
		PilotState:         NameForPilotState(raw.PilotState),
		IsConnected:        connected,
	}
	if lookup != nil {
		names, _ := lookup.Resolve(raw.LiveryID, raw.AircraftID)
		out.AircraftName = names.AircraftName
		out.LiveryName = names.LiveryName
	}
	return out
}
