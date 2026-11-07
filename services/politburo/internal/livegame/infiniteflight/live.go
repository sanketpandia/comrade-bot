package infiniteflight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LiveOrganization is a row from GET /live/organizations.
type LiveOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role int    `json:"role"`
}

// LiveAircraft is a row from GET /live/organizations/{organizationId}/aircraft.
type LiveAircraft struct {
	ID                string     `json:"id"`
	AircraftID        string     `json:"aircraftId"`
	OrganizationID    string     `json:"organizationId"`
	Registration      string     `json:"registration"`
	Status            int        `json:"status"`
	Visibility        int        `json:"visibility"`
	FleetPriority     *int       `json:"fleetPriority"`
	FleetRank         int        `json:"fleetRank"`
	IsFleetActiveSlot bool       `json:"isFleetActiveSlot"`
	CreatedAt         time.Time  `json:"createdAt"`
}

// LiveScheduleCrewMember is an assigned crew row on a Live schedule.
type LiveScheduleCrewMember struct {
	UserID   string  `json:"userId"`
	Username *string `json:"username"`
	Role     int     `json:"role"`
}

// LiveSchedule is a row from GET /live/organizations/{organizationId}/schedules.
type LiveSchedule struct {
	ID                    string                 `json:"id"`
	Status                int                    `json:"status"`
	Callsign              string                 `json:"callsign"`
	OrganizationID        string                 `json:"organizationId"`
	AircraftID            string                 `json:"aircraftId"`
	FlightType            int                    `json:"flightType"`
	OriginIcao            string                 `json:"originIcao"`
	DestinationIcao       string                 `json:"destinationIcao"`
	IsTimeless            bool                   `json:"isTimeless"`
	ScheduledDepartureUtc time.Time              `json:"scheduledDepartureUtc"`
	ScheduledArrivalUtc   time.Time              `json:"scheduledArrivalUtc"`
	Briefing              *string                `json:"briefing"`
	Debriefing            *string                `json:"debriefing"`
	FlightPlan            *string                `json:"flightPlan"`
	Crew                  []LiveScheduleCrewMember `json:"crew"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
	Sequence              int                    `json:"sequence"`
}

// LiveSchedulePage is the paged result for organization schedules.
type LiveSchedulePage struct {
	Items      []LiveSchedule `json:"items"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
}

// LiveAircraftFlightCrewMember is a pilot on a flight log entry.
type LiveAircraftFlightCrewMember struct {
	UserID   string  `json:"userId"`
	Username *string `json:"username"`
}

// LivePilotFlightTime is one control period on a flight (decoded; UI may ignore).
type LivePilotFlightTime struct {
	UserID            string   `json:"userId"`
	Username          *string  `json:"username"`
	FlightTimeSeconds float64  `json:"flightTimeSeconds"`
	StartSeconds      float64  `json:"startSeconds"`
	EndSeconds        *float64 `json:"endSeconds"`
}

// LiveAircraftFlight is a row from GET /live/aircraft/{aircraftId}/flights.
type LiveAircraftFlight struct {
	ID                  string                         `json:"id"`
	AircraftID          string                         `json:"aircraftId"`
	OrganizationID      string                         `json:"organizationId"`
	Callsign            string                         `json:"callsign"`
	OriginIcao          *string                        `json:"originIcao"`
	DestinationIcao     *string                        `json:"destinationIcao"`
	ActualDepartureUtc  *time.Time                     `json:"actualDepartureUtc"`
	ActualArrivalUtc    *time.Time                     `json:"actualArrivalUtc"`
	Status              int                            `json:"status"`
	FlightTimeMinutes   float64                        `json:"flightTimeMinutes"`
	AirframeTimeMinutes float64                        `json:"airframeTimeMinutes"`
	Crew                []LiveAircraftFlightCrewMember `json:"crew"`
	PilotFlightTimes    []LivePilotFlightTime          `json:"pilotFlightTimes"`
}

// LiveAircraftFlightPage is the paged result for aircraft flights.
type LiveAircraftFlightPage struct {
	Items      []LiveAircraftFlight `json:"items"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalCount int                  `json:"totalCount"`
	TotalPages int                  `json:"totalPages"`
}

// LiveClient calls Infinite Flight Public API v3 with a user OAuth access token.
type LiveClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLiveClient builds a client for the Live API base URL (typically /public/v3).
func NewLiveClient(baseURL string, timeout time.Duration) (*LiveClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("Live API base URL is empty")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &LiveClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

type liveEnvelope[T any] struct {
	ErrorCode int `json:"errorCode"`
	Result    T   `json:"result"`
}

const defaultLiveResponseLimit = 1 << 20

func (c *LiveClient) getLive(ctx context.Context, accessToken, path, action string, notFound error) ([]byte, error) {
	return c.requestLive(ctx, http.MethodGet, accessToken, path, action, notFound, defaultLiveResponseLimit, nil)
}

func (c *LiveClient) getLiveWithLimit(ctx context.Context, accessToken, path, action string, notFound error, maxBody int64) ([]byte, error) {
	return c.requestLive(ctx, http.MethodGet, accessToken, path, action, notFound, maxBody, nil)
}

func (c *LiveClient) requestLive(ctx context.Context, method, accessToken, path, action string, notFound error, maxBody int64, body []byte) ([]byte, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if maxBody <= 0 {
		maxBody = defaultLiveResponseLimit
	}
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", action, err)
	}
	if resp.StatusCode == http.StatusNotFound && notFound != nil {
		return nil, notFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", action, resp.StatusCode)
	}
	return payload, nil
}

func decodeLiveVoid(payload []byte, action string) error {
	var parsed liveEnvelope[json.RawMessage]
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("decode %s response: %w", action, err)
	}
	if parsed.ErrorCode != upstreamErrorCodeOK {
		return fmt.Errorf("%s: errorCode %d", action, parsed.ErrorCode)
	}
	return nil
}

func decodeLive[T any](payload []byte, action string) (T, error) {
	var zero T
	var parsed liveEnvelope[T]
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return zero, fmt.Errorf("decode %s response: %w", action, err)
	}
	if parsed.ErrorCode != upstreamErrorCodeOK {
		return zero, fmt.Errorf("%s: errorCode %d", action, parsed.ErrorCode)
	}
	return parsed.Result, nil
}

const listOrganizationsAction = "list Infinite Flight organizations"
const getOrganizationAction = "get Infinite Flight organization"
const listOrganizationAircraftAction = "list Infinite Flight organization aircraft"
const listOrganizationSchedulesAction = "list Infinite Flight organization schedules"
const listAircraftSchedulesAction = "list Infinite Flight aircraft schedules"
const listAircraftFlightsAction = "list Infinite Flight aircraft flights"
const createAircraftScheduleAction = "create Infinite Flight aircraft schedule"
const updateScheduleAction = "update Infinite Flight schedule"
const updateScheduleFlightPlanAction = "update Infinite Flight schedule flight plan"
const assignScheduleCrewAction = "assign Infinite Flight schedule crew"

const scheduleResponseLimit = 4 << 20

// ListOrganizations returns organizations for the bearer access token.
func (c *LiveClient) ListOrganizations(ctx context.Context, accessToken string) ([]LiveOrganization, error) {
	payload, err := c.getLive(ctx, accessToken, "/live/organizations", listOrganizationsAction, nil)
	if err != nil {
		return nil, err
	}
	return decodeLive[[]LiveOrganization](payload, listOrganizationsAction)
}

// GetOrganization returns one organization for the bearer access token.
func (c *LiveClient) GetOrganization(ctx context.Context, accessToken, organizationID string) (LiveOrganization, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return LiveOrganization{}, fmt.Errorf("organization id is required")
	}
	payload, err := c.getLive(ctx, accessToken, "/live/organizations/"+organizationID, getOrganizationAction, ErrLiveOrganizationNotFound)
	if err != nil {
		return LiveOrganization{}, err
	}
	return decodeLive[LiveOrganization](payload, getOrganizationAction)
}

// ListOrganizationAircraft returns active aircraft for an organization.
func (c *LiveClient) ListOrganizationAircraft(ctx context.Context, accessToken, organizationID string) ([]LiveAircraft, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	payload, err := c.getLive(ctx, accessToken, "/live/organizations/"+organizationID+"/aircraft", listOrganizationAircraftAction, ErrLiveOrganizationNotFound)
	if err != nil {
		return nil, err
	}
	return decodeLive[[]LiveAircraft](payload, listOrganizationAircraftAction)
}

// ListOrganizationSchedules returns one page of open schedules for an organization.
func (c *LiveClient) ListOrganizationSchedules(ctx context.Context, accessToken, organizationID string, page, pageSize int, aircraftIDs []string) (LiveSchedulePage, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return LiveSchedulePage{}, fmt.Errorf("organization id is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))
	for _, id := range aircraftIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			query.Add("aircraftId", id)
		}
	}
	path := "/live/organizations/" + organizationID + "/schedules"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	payload, err := c.getLiveWithLimit(ctx, accessToken, path, listOrganizationSchedulesAction, ErrLiveOrganizationNotFound, scheduleResponseLimit)
	if err != nil {
		return LiveSchedulePage{}, err
	}
	return decodeLive[LiveSchedulePage](payload, listOrganizationSchedulesAction)
}

// ListAircraftSchedules returns schedules for one organization aircraft (persistent id).
func (c *LiveClient) ListAircraftSchedules(ctx context.Context, accessToken, aircraftID string) ([]LiveSchedule, error) {
	aircraftID = strings.TrimSpace(aircraftID)
	if aircraftID == "" {
		return nil, fmt.Errorf("aircraft id is required")
	}
	path := "/live/aircraft/" + aircraftID + "/schedules"
	payload, err := c.getLiveWithLimit(ctx, accessToken, path, listAircraftSchedulesAction, ErrLiveOrganizationNotFound, scheduleResponseLimit)
	if err != nil {
		return nil, err
	}
	return decodeLive[[]LiveSchedule](payload, listAircraftSchedulesAction)
}

// ListAircraftFlights returns one page of flight log entries for an organization aircraft (persistent id).
func (c *LiveClient) ListAircraftFlights(ctx context.Context, accessToken, aircraftID string, page, pageSize int) (LiveAircraftFlightPage, error) {
	aircraftID = strings.TrimSpace(aircraftID)
	if aircraftID == "" {
		return LiveAircraftFlightPage{}, fmt.Errorf("aircraft id is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))
	path := "/live/aircraft/" + aircraftID + "/flights?" + query.Encode()
	payload, err := c.getLiveWithLimit(ctx, accessToken, path, listAircraftFlightsAction, ErrLiveOrganizationNotFound, scheduleResponseLimit)
	if err != nil {
		return LiveAircraftFlightPage{}, err
	}
	return decodeLive[LiveAircraftFlightPage](payload, listAircraftFlightsAction)
}

// CreateAircraftSchedule proxies POST /live/aircraft/{aircraftId}/schedules.
func (c *LiveClient) CreateAircraftSchedule(ctx context.Context, accessToken, aircraftID string, body json.RawMessage) error {
	aircraftID = strings.TrimSpace(aircraftID)
	if aircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}
	path := "/live/aircraft/" + aircraftID + "/schedules"
	payload, err := c.requestLive(ctx, http.MethodPost, accessToken, path, createAircraftScheduleAction, ErrLiveOrganizationNotFound, defaultLiveResponseLimit, body)
	if err != nil {
		return err
	}
	return decodeLiveVoid(payload, createAircraftScheduleAction)
}

// UpdateSchedule proxies PUT /live/schedules/{scheduleId}.
func (c *LiveClient) UpdateSchedule(ctx context.Context, accessToken, scheduleID string, body json.RawMessage) error {
	scheduleID = strings.TrimSpace(scheduleID)
	if scheduleID == "" {
		return fmt.Errorf("schedule id is required")
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}
	path := "/live/schedules/" + scheduleID
	payload, err := c.requestLive(ctx, http.MethodPut, accessToken, path, updateScheduleAction, ErrLiveOrganizationNotFound, defaultLiveResponseLimit, body)
	if err != nil {
		return err
	}
	return decodeLiveVoid(payload, updateScheduleAction)
}

// UpdateScheduleFlightPlan proxies PUT /live/schedules/{scheduleId}/flightplan.
func (c *LiveClient) UpdateScheduleFlightPlan(ctx context.Context, accessToken, scheduleID string, body json.RawMessage) error {
	scheduleID = strings.TrimSpace(scheduleID)
	if scheduleID == "" {
		return fmt.Errorf("schedule id is required")
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}
	path := "/live/schedules/" + scheduleID + "/flightplan"
	payload, err := c.requestLive(ctx, http.MethodPut, accessToken, path, updateScheduleFlightPlanAction, ErrLiveOrganizationNotFound, defaultLiveResponseLimit, body)
	if err != nil {
		return err
	}
	return decodeLiveVoid(payload, updateScheduleFlightPlanAction)
}

// AssignScheduleCrew proxies PUT /live/schedules/{scheduleId}/crew/{userId}.
func (c *LiveClient) AssignScheduleCrew(ctx context.Context, accessToken, scheduleID, userID string, body json.RawMessage) error {
	scheduleID = strings.TrimSpace(scheduleID)
	userID = strings.TrimSpace(userID)
	if scheduleID == "" {
		return fmt.Errorf("schedule id is required")
	}
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}
	path := "/live/schedules/" + scheduleID + "/crew/" + userID
	payload, err := c.requestLive(ctx, http.MethodPut, accessToken, path, assignScheduleCrewAction, ErrLiveOrganizationNotFound, defaultLiveResponseLimit, body)
	if err != nil {
		return err
	}
	return decodeLiveVoid(payload, assignScheduleCrewAction)
}
