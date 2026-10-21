package flights

import (
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

// Observation is upstream flight data as received; only NewObservation constructs it.
type Observation struct {
	FlightID            string    `json:"flightId"`
	UserID              string    `json:"userId"`
	AircraftID          string    `json:"aircraftId"`
	LiveryID            string    `json:"liveryId"`
	Callsign            string    `json:"callsign"`
	Username            *string   `json:"username"`
	VirtualOrganization *string   `json:"virtualOrganization"`
	Latitude            float64   `json:"latitude"`
	Longitude           float64   `json:"longitude"`
	Altitude            float64   `json:"altitude"`
	Speed               float64   `json:"speed"`
	VerticalSpeed       float64   `json:"verticalSpeed"`
	Track               float64   `json:"track"`
	LastReport          time.Time `json:"lastReport"`
	LastReportValid     bool      `json:"lastReportValid"`
	ObservedAt          time.Time `json:"observedAt"`
	PilotState          int       `json:"pilotState"`
	IsConnected         bool      `json:"isConnected"`
	SessionID           string    `json:"sessionId"`
	NormalizedName      string    `json:"normalizedName"`
}

// NewObservation maps one upstream flight tick without rounding or derived enrichment.
func NewObservation(upstream infiniteflight.Flight, session infiniteflight.Session, observedAt time.Time) Observation {
	lastReport, valid := parseUpstreamLastReport(upstream.LastReport)
	return Observation{
		FlightID:            upstream.FlightID,
		UserID:              upstream.UserID,
		AircraftID:          upstream.AircraftID,
		LiveryID:            upstream.LiveryID,
		Username:            upstream.Username,
		VirtualOrganization: upstream.VirtualOrganization,
		Callsign:            upstream.Callsign,
		Latitude:            upstream.Latitude,
		Longitude:           upstream.Longitude,
		Altitude:            upstream.Altitude,
		Speed:               upstream.Speed,
		VerticalSpeed:       upstream.VerticalSpeed,
		Track:               upstream.Track,
		LastReport:          lastReport,
		LastReportValid:     valid,
		PilotState:          upstream.PilotState,
		IsConnected:         upstream.IsConnected,
		SessionID:           session.ID,
		NormalizedName:      session.NormalizedName,
		ObservedAt:          observedAt.UTC(),
	}
}

func parseUpstreamLastReport(raw string) (time.Time, bool) {
	parsed, err := time.Parse(lastReportLayout, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// UsernameFromObservation returns the pilot display name when present.
func UsernameFromObservation(obs Observation) string {
	if obs.Username == nil {
		return ""
	}
	return *obs.Username
}
