package live

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

// encodeScheduleUpsertForUpstream converts API string enums to Infinite Flight integer fields.
func encodeScheduleUpsertForUpstream(body politburoapi.LiveScheduleUpsertBody) (json.RawMessage, error) {
	out := map[string]any{}
	if body.Status != nil {
		code, err := infiniteflight.ParseScheduledFlightStatus(string(*body.Status))
		if err != nil {
			return nil, err
		}
		out["status"] = code
	}
	if body.FlightType != nil {
		code, err := infiniteflight.ParsePersistentFlightType(string(*body.FlightType))
		if err != nil {
			return nil, err
		}
		out["flightType"] = code
	}
	if body.Callsign != nil {
		out["callsign"] = strings.TrimSpace(*body.Callsign)
	}
	if body.OriginIcao != nil {
		out["originIcao"] = strings.TrimSpace(*body.OriginIcao)
	}
	if body.DestinationIcao != nil {
		out["destinationIcao"] = strings.TrimSpace(*body.DestinationIcao)
	}
	if body.IsTimeless != nil {
		out["isTimeless"] = *body.IsTimeless
	}
	if body.ScheduledDepartureUtc != nil {
		out["scheduledDepartureUtc"] = body.ScheduledDepartureUtc.UTC().Format(time.RFC3339)
	}
	if body.ScheduledArrivalUtc != nil {
		out["scheduledArrivalUtc"] = body.ScheduledArrivalUtc.UTC().Format(time.RFC3339)
	}
	if body.Briefing != nil {
		out["briefing"] = *body.Briefing
	}
	if body.Debriefing != nil {
		out["debriefing"] = *body.Debriefing
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty schedule body")
	}
	return json.Marshal(out)
}
