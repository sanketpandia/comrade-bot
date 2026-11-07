package infiniteflight

import (
	"errors"
	"fmt"
)

// ErrLiveOrganizationNotFound is returned when the upstream Live API responds with HTTP 404.
var ErrLiveOrganizationNotFound = errors.New("infinite flight live organization not found")

// HTTPStatusError is returned when Infinite Flight Live responds with a non-OK HTTP status.
type HTTPStatusError struct {
	Status int
	Action string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%s: HTTP %d", e.Action, e.Status)
}

// HTTPStatusFromError returns the upstream HTTP status when err is an HTTPStatusError.
func HTTPStatusFromError(err error) (int, bool) {
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) && httpErr != nil {
		return httpErr.Status, true
	}
	return 0, false
}
