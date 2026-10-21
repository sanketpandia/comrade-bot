package flightresolution

import (
	"fmt"

	"infinite-experiment/politburo/internal/livegame/flights"
)

var normalizedDisplayEffect = EffectDecision{
	Name:    "NormalizedDisplay",
	Applies: normalizedDisplayApplies,
	Apply:   applyNormalizedDisplay,
}

func normalizedDisplayApplies(in EffectInput) bool {
	return in.Facts.IsNew || in.Facts.MotionChanged
}

func applyNormalizedDisplay(in EffectInput, out *flights.Derived, _ *Outcome) {
	raw := in.Raw
	speed := flights.RoundedSpeed(raw.Speed)
	verticalSpeed := flights.RoundedVerticalSpeed(raw.VerticalSpeed)
	connected := "disconnected"
	if raw.IsConnected {
		connected = "connected"
	}
	out.Normalized = flights.Normalized{
		Speed:         fmt.Sprintf("%d kts", speed),
		VerticalSpeed: fmt.Sprintf("%.1f ft/min", verticalSpeed),
		PilotState:    flights.NameForPilotState(raw.PilotState),
		IsConnected:   connected,
	}
}
