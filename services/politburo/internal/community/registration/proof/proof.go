package proof

import (
	"errors"
	"fmt"
	"strings"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

var (
	ErrInvalidRoute = errors.New("invalid route proof")
)

// ParseRouteProof parses ORIG-DEST (4-letter ICAO codes) from registration input.
func ParseRouteProof(raw string) (origin, destination string, err error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: expected ORIG-DEST", ErrInvalidRoute)
	}
	origin = strings.TrimSpace(parts[0])
	destination = strings.TrimSpace(parts[1])
	if len(origin) != 4 || len(destination) != 4 {
		return "", "", fmt.Errorf("%w: ICAO codes must be 4 letters", ErrInvalidRoute)
	}
	return origin, destination, nil
}

// MatchesLatestCompleteFlight returns true when the newest logbook row with both
// origin and destination matches the given ICAO pair.
func MatchesLatestCompleteFlight(flights []infiniteflight.LogbookFlight, origin, destination string) bool {
	wantOrigin := strings.ToUpper(strings.TrimSpace(origin))
	wantDest := strings.ToUpper(strings.TrimSpace(destination))
	for _, flight := range flights {
		if flight.Origin == "" || flight.Destination == "" {
			continue
		}
		return flight.Origin == wantOrigin && flight.Destination == wantDest
	}
	return false
}
