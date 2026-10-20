package flights

import (
	"fmt"
	"strings"
)

var callsignWeightClassTrailers = buildCallsignWeightClassTrailers()

func buildCallsignWeightClassTrailers() []string {
	trailers := []string{" Heavy", " Super"}
	for n := 10; n >= 2; n-- {
		trailers = append(trailers, fmt.Sprintf(" Flight of %d", n))
	}
	return trailers
}

// NormalizedCallsignForAffix strips Infinite Flight weight / formation trailers from the
// end of the upstream callsign for VA affix matching and cache display.
func NormalizedCallsignForAffix(callsign string) string {
	s := strings.TrimSpace(callsign)
	for {
		trimmed := false
		for _, trailer := range callsignWeightClassTrailers {
			if strings.HasSuffix(s, trailer) {
				s = strings.TrimSuffix(s, trailer)
				trimmed = true
				break
			}
		}
		if !trimmed {
			return strings.TrimSpace(s)
		}
	}
}
