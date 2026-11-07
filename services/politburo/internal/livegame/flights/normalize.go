package flights

import (
	"fmt"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

// Normalize builds serve-time display strings from a cached record and the livery catalog.
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
