package cache

import "time"

// Redis TTLs for cached snapshots and records.
const (
	SessionsCacheTTL = 24 * time.Hour
	ActiveFlightsTTL = 3 * 24 * time.Hour
)

// Refresh intervals align with job cadence; used for meta.refreshIntervalMins and freshness checks.
const (
	SessionsRefreshInterval = 5 * time.Minute
	FlightsRefreshInterval  = time.Minute
	LiveriesRefreshInterval = 30 * time.Minute
)
