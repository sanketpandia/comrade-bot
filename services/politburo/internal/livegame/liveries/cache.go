// Package liveries defines aircraft livery catalog storage and lookup.
package liveries

import "time"

const (
	RefreshSchedule = "0 */30 * * * *"
	RefreshInterval = 30 * time.Minute
)

// UnrecognizedLiveryName is used when only aircraft_id matches the catalog.
const UnrecognizedLiveryName = "Unrecognized"
