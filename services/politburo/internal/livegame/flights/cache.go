// Package flights defines the shared cache contract for active Infinite Flight flights.
package flights

import "time"

const (
	MaxFlightsPerRequest = 5000
	MaxPageLength        = 5000
	DefaultPageLength    = 50
	DefaultPageNumber    = 1
	FPLSyncInterval      = 5 * time.Minute
	lastReportLayout     = "2006-01-02 15:04:05Z07:00"
)

type PathSync struct {
	FPLSyncRequired bool      `json:"fplSyncRequired"`
	LastFPLSyncAt   time.Time `json:"lastFPLSyncAt,omitempty"`
}

type Normalized struct {
	Speed         string `json:"speed"`
	VerticalSpeed string `json:"verticalSpeed"`
	PilotState    string `json:"pilotState"`
	IsConnected   string `json:"isConnected"`
}

type Flight struct {
	FlightID            string     `json:"flightId"`
	UserID              string     `json:"userId"`
	AircraftID          string     `json:"aircraftId"`
	LiveryID            string     `json:"liveryId"`
	Username            *string    `json:"username"`
	VirtualOrganization *string    `json:"virtualOrganization"`
	Callsign            string     `json:"callsign"`
	Latitude            float64    `json:"latitude"`
	Longitude           float64    `json:"longitude"`
	Altitude            int        `json:"altitude"`
	Speed               int        `json:"speed"`
	VerticalSpeed       float64    `json:"verticalSpeed"`
	Track               float64    `json:"track"`
	LastReport          time.Time  `json:"lastReport"`
	PilotState          int        `json:"pilotState"`
	IsConnected         bool       `json:"isConnected"`
	AircraftName        string     `json:"aircraftName,omitempty"`
	LiveryName          string     `json:"liveryName,omitempty"`
	SessionID           string     `json:"sessionId"`
	NormalizedName      string     `json:"normalizedName"`
	Normalized          Normalized `json:"normalized"`
	PathSync            *PathSync  `json:"pathSync,omitempty"`
}

type FlightMotion struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     int     `json:"speed"`
	Callsign  string  `json:"callsign"`
	Track     float64 `json:"track,omitempty"`
}

// Snapshot is the per-server motion index written to game:flights:active:<server>.
type Snapshot struct {
	Tracks     map[string]FlightMotion `json:"tracks,omitempty"`
	LastCached time.Time               `json:"lastCached"`
}
