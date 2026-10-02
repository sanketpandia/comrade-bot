package infiniteflight

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infiniteflightapi "infinite-experiment/politburo/internal/api/generated/infiniteflight"
)

// ErrUserNotFound means the IFC username did not resolve to an Infinite Flight user.
var ErrUserNotFound = errors.New("infinite flight user not found")

const upstreamErrorCodeOK = 0
const upstreamErrorCodeUserNotFound = 1

// LogbookFlight is a minimal row from GET /users/{userId}/flights used for registration proof.
type LogbookFlight struct {
	Origin      string
	Destination string
}

// UsersClient resolves IFC usernames and reads recent logbook flights.
type UsersClient interface {
	ResolveUserIDByIFCUsername(ctx context.Context, ifcUsername string) (string, error)
	RecentLogbookFlights(ctx context.Context, userID string, limit int) ([]LogbookFlight, error)
}

// ResolveUserIDByIFCUsername calls POST /users with discourseNames (IFC username).
func (c *Client) ResolveUserIDByIFCUsername(ctx context.Context, ifcUsername string) (string, error) {
	name := strings.TrimSpace(ifcUsername)
	if name == "" {
		return "", fmt.Errorf("ifc username is required")
	}
	response, err := c.generated.LookupUsersWithResponse(ctx, infiniteflightapi.LookupUsersJSONRequestBody{
		DiscourseNames: &[]string{name},
	})
	if err != nil {
		return "", fmt.Errorf("lookup Infinite Flight user: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return "", fmt.Errorf("lookup Infinite Flight user: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return "", fmt.Errorf("lookup Infinite Flight user: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode == upstreamErrorCodeUserNotFound {
		return "", ErrUserNotFound
	}
	if response.JSON200.ErrorCode != upstreamErrorCodeOK {
		return "", fmt.Errorf("lookup Infinite Flight user: errorCode %d", response.JSON200.ErrorCode)
	}

	want := strings.EqualFold
	for _, item := range response.JSON200.Result {
		if item.ErrorCode == upstreamErrorCodeUserNotFound {
			continue
		}
		if item.ErrorCode != upstreamErrorCodeOK {
			continue
		}
		if item.DiscourseUsername != nil && want(*item.DiscourseUsername, name) {
			return item.UserId, nil
		}
		if item.UserId != "" {
			// Upstream may omit discourseUsername for unlinked accounts; accept sole result.
			if len(response.JSON200.Result) == 1 {
				return item.UserId, nil
			}
		}
	}
	return "", ErrUserNotFound
}

// RecentLogbookFlights returns logbook rows from page 1 in recency order.
// When limit is greater than zero, at most limit rows are returned after skipping
// rows with no origin and no destination. Partial rows (missing origin or destination)
// are included so callers can resolve the latest complete flight.
func (c *Client) RecentLogbookFlights(ctx context.Context, userID string, limit int) ([]LogbookFlight, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if limit <= 0 {
		limit = 3
	}
	page := 1
	response, err := c.generated.GetUserFlightsWithResponse(ctx, userID, &infiniteflightapi.GetUserFlightsParams{
		Page: &page,
	})
	if err != nil {
		return nil, fmt.Errorf("get Infinite Flight user flights: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return nil, fmt.Errorf("get Infinite Flight user flights: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return nil, fmt.Errorf("get Infinite Flight user flights: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode == upstreamErrorCodeUserNotFound {
		return nil, ErrUserNotFound
	}
	if response.JSON200.ErrorCode != upstreamErrorCodeOK {
		return nil, fmt.Errorf("get Infinite Flight user flights: errorCode %d", response.JSON200.ErrorCode)
	}

	rows := response.JSON200.Result.Data
	out := make([]LogbookFlight, 0, min(limit, len(rows)))
	for i := 0; i < len(rows) && len(out) < limit; i++ {
		origin := normalizeAirport(rows[i].OriginAirport)
		dest := normalizeAirport(rows[i].DestinationAirport)
		if origin == "" && dest == "" {
			continue
		}
		out = append(out, LogbookFlight{Origin: origin, Destination: dest})
	}
	return out, nil
}

func normalizeAirport(value *string) string {
	if value == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(*value))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
