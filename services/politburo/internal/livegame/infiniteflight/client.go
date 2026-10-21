package infiniteflight

import (
	"context"
	"fmt"
	"net/http"
	"time"

	infiniteflightapi "infinite-experiment/politburo/internal/api/generated/infiniteflight"
)

type Session struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	WorldType         int       `json:"worldType"`
	Type              int       `json:"type"`
	MinimumGradeLevel int       `json:"minimumGradeLevel"`
	UserCount         int       `json:"userCount"`
	MinimumAppVersion string    `json:"minimumAppVersion"`
	MaximumAppVersion *string   `json:"maximumAppVersion"`
	MaxUsers          int       `json:"maxUsers"`
	NormalizedName    string    `json:"normalizedName"`
	Timestamp         time.Time `json:"timestamp"`
}

type Flight struct {
	Username            *string
	Callsign            string
	Latitude            float64
	Longitude           float64
	Altitude            float64
	Speed               float64
	VerticalSpeed       float64
	Track               float64
	LastReport          string
	FlightID            string
	UserID              string
	AircraftID          string
	LiveryID            string
	VirtualOrganization *string
	PilotState          int
	IsConnected         bool
}

type Livery struct {
	ID           string `json:"id"`
	AircraftID   string `json:"aircraftId"`
	AircraftName string `json:"aircraftName"`
	LiveryName   string `json:"liveryName"`
}

type SessionsClient interface {
	GetSessions(context.Context) ([]Session, error)
}

type Coordinate struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

type FlightPlanItem struct {
	Name       string
	Type       int
	Children   []FlightPlanItem
	Identifier *string
	Altitude   int
	Location   Coordinate
}

type FlightPlan struct {
	FlightPlanID   string
	FlightID       string
	Waypoints      []string
	LastUpdate     string
	Items          []FlightPlanItem
	FlightPlanType *int
}

type FlightsClient interface {
	GetSessionFlights(context.Context, string) ([]Flight, error)
	GetFlightPlan(context.Context, string, string) (FlightPlan, error)
}

type LiveriesClient interface {
	GetAircraftLiveries(context.Context) ([]Livery, error)
}

type ClientAPI interface {
	SessionsClient
	FlightsClient
	LiveriesClient
	UsersClient
}

type Client struct {
	generated *infiniteflightapi.ClientWithResponses
}

func NewClient(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	return newClient(baseURL, apiKey, &http.Client{Timeout: timeout})
}

func newClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	generated, err := infiniteflightapi.NewClientWithResponses(
		baseURL,
		infiniteflightapi.WithHTTPClient(httpClient),
		infiniteflightapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create Infinite Flight client: %w", err)
	}
	return &Client{generated: generated}, nil
}

func (c *Client) GetSessions(ctx context.Context) ([]Session, error) {
	response, err := c.generated.GetSessionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get Infinite Flight sessions: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return nil, fmt.Errorf("get Infinite Flight sessions: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return nil, fmt.Errorf("get Infinite Flight sessions: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode != 0 {
		return nil, fmt.Errorf("get Infinite Flight sessions: errorCode %d", response.JSON200.ErrorCode)
	}

	sessions := make([]Session, 0, len(response.JSON200.Result))
	for _, item := range response.JSON200.Result {
		sessions = append(sessions, Session{
			ID: item.Id, Name: item.Name, WorldType: item.WorldType, Type: item.Type,
			MinimumGradeLevel: item.MinimumGradeLevel, UserCount: item.UserCount,
			MinimumAppVersion: item.MinimumAppVersion, MaximumAppVersion: item.MaximumAppVersion,
			MaxUsers: item.MaxUsers,
		})
	}
	return sessions, nil
}

func (c *Client) GetSessionFlights(ctx context.Context, sessionID string) ([]Flight, error) {
	response, err := c.generated.GetSessionFlightsWithResponse(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get Infinite Flight session flights: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return nil, fmt.Errorf("get Infinite Flight session flights: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return nil, fmt.Errorf("get Infinite Flight session flights: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode != 0 {
		return nil, fmt.Errorf("get Infinite Flight session flights: errorCode %d", response.JSON200.ErrorCode)
	}

	flights := make([]Flight, 0, len(response.JSON200.Result))
	for _, item := range response.JSON200.Result {
		flights = append(flights, Flight{
			Username: item.Username, Callsign: item.Callsign, Latitude: item.Latitude,
			Longitude: item.Longitude, Altitude: item.Altitude, Speed: item.Speed,
			VerticalSpeed: item.VerticalSpeed, Track: item.Track,
			LastReport: item.LastReport, FlightID: item.FlightId, UserID: item.UserId,
			AircraftID: item.AircraftId, LiveryID: item.LiveryId,
			VirtualOrganization: item.VirtualOrganization, PilotState: item.PilotState,
			IsConnected: item.IsConnected,
		})
	}
	return flights, nil
}

func (c *Client) GetFlightPlan(ctx context.Context, sessionID, flightID string) (FlightPlan, error) {
	response, err := c.generated.GetFlightPlanWithResponse(ctx, sessionID, flightID)
	if err != nil {
		return FlightPlan{}, fmt.Errorf("get Infinite Flight flight plan: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return FlightPlan{}, fmt.Errorf("get Infinite Flight flight plan: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return FlightPlan{}, fmt.Errorf("get Infinite Flight flight plan: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode != 0 {
		return FlightPlan{}, fmt.Errorf("get Infinite Flight flight plan: errorCode %d", response.JSON200.ErrorCode)
	}
	return flightPlanFromGenerated(response.JSON200.Result), nil
}

func flightPlanFromGenerated(plan infiniteflightapi.FlightPlanInfo) FlightPlan {
	items := make([]FlightPlanItem, 0, len(plan.FlightPlanItems))
	for _, item := range plan.FlightPlanItems {
		items = append(items, flightPlanItemFromGenerated(item))
	}
	return FlightPlan{
		FlightPlanID:   plan.FlightPlanId,
		FlightID:       plan.FlightId,
		Waypoints:      plan.Waypoints,
		LastUpdate:     plan.LastUpdate,
		Items:          items,
		FlightPlanType: plan.FlightPlanType,
	}
}

func flightPlanItemFromGenerated(item infiniteflightapi.FlightPlanItem) FlightPlanItem {
	out := FlightPlanItem{
		Name:       item.Name,
		Type:       item.Type,
		Identifier: item.Identifier,
		Altitude:   item.Altitude,
		Location: Coordinate{
			Latitude:  item.Location.Latitude,
			Longitude: item.Location.Longitude,
			Altitude:  item.Location.Altitude,
		},
	}
	if item.Children != nil {
		out.Children = make([]FlightPlanItem, 0, len(*item.Children))
		for _, child := range *item.Children {
			out.Children = append(out.Children, flightPlanItemFromGenerated(child))
		}
	}
	return out
}

func (c *Client) GetAircraftLiveries(ctx context.Context) ([]Livery, error) {
	response, err := c.generated.GetAircraftLiveriesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get Infinite Flight aircraft liveries: %w", err)
	}
	if response.JSON200 == nil {
		if response.JSONDefault != nil {
			return nil, fmt.Errorf("get Infinite Flight aircraft liveries: HTTP %d, errorCode %d", response.StatusCode(), response.JSONDefault.ErrorCode)
		}
		return nil, fmt.Errorf("get Infinite Flight aircraft liveries: unexpected HTTP %d", response.StatusCode())
	}
	if response.JSON200.ErrorCode != 0 {
		return nil, fmt.Errorf("get Infinite Flight aircraft liveries: errorCode %d", response.JSON200.ErrorCode)
	}

	liveries := make([]Livery, 0, len(response.JSON200.Result))
	for _, item := range response.JSON200.Result {
		liveries = append(liveries, Livery{
			ID: item.Id, AircraftID: item.AircraftID, AircraftName: item.AircraftName, LiveryName: item.LiveryName,
		})
	}
	return liveries, nil
}
