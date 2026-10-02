package liveries

import "time"

type AircraftLivery struct {
	LiveryID            string
	AircraftID          string
	AircraftName        string
	LiveryName          string
	DisplayAircraftName string
	DisplayLiveryName   string
	UpdatedAt           time.Time
}
