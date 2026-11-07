package valive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"infinite-experiment/politburo/internal/community/registration/proof"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

var (
	ErrConnectionNotFound   = errors.New("live connection not found")
	ErrRegistrationNotFound = errors.New("registration not in organization fleet")
	ErrScheduleNotFound     = errors.New("no open schedule matches registration and route")
	ErrLiveClientDisabled   = errors.New("infinite flight live client disabled")
)

// ActionAssignScheduleCrew is the live_api_v3_action_history action_name for crew self-assign.
const ActionAssignScheduleCrew = "assign_va_live_schedule_crew"

// ScheduleCrewLiveClient lists fleet data and assigns schedule crew upstream.
type ScheduleCrewLiveClient interface {
	ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error)
	ListOrganizationSchedules(context.Context, string, string, int, int, []string) (infiniteflight.LiveSchedulePage, error)
	AssignScheduleCrew(context.Context, string, string, string, json.RawMessage) error
}

// AssignCaptainService matches an open schedule and assigns the caller as captain.
type AssignCaptainService struct {
	connections *ConnectionsRepository
	tokens      *TokenProvider
	live        ScheduleCrewLiveClient
}

func NewAssignCaptainService(
	connections *ConnectionsRepository,
	tokens *TokenProvider,
	live ScheduleCrewLiveClient,
) *AssignCaptainService {
	return &AssignCaptainService{connections: connections, tokens: tokens, live: live}
}

// AssignCaptainInput carries primitive ids only (no HTTP claims).
type AssignCaptainInput struct {
	VAID             string
	OrganizationID   string
	ActorUserID      *string
	IFUserID         string
	Registration     string
	Route            string
}

// AssignCaptainResult is returned after a successful crew assignment.
type AssignCaptainResult struct {
	ScheduleID      string
	OrganizationID  string
	Callsign          string
	Registration      string
	OriginIcao        string
	DestinationIcao   string
}

// ResolveAircraftIDs returns persistent fleet row ids whose registration matches.
func (s *AssignCaptainService) ResolveAircraftIDs(ctx context.Context, vaID, orgID, registration string) ([]string, map[string]string, error) {
	vaID = strings.TrimSpace(vaID)
	orgID = strings.TrimSpace(orgID)
	wantReg := strings.TrimSpace(registration)
	if vaID == "" || orgID == "" {
		return nil, nil, fmt.Errorf("va id and organization id are required")
	}
	if wantReg == "" {
		return nil, nil, fmt.Errorf("registration is required")
	}
	if s.connections == nil || s.tokens == nil {
		return nil, nil, fmt.Errorf("live assign captain is not configured")
	}
	if s.live == nil {
		return nil, nil, ErrLiveClientDisabled
	}
	conn, err := s.connections.GetByVAAndOrg(ctx, vaID, orgID)
	if err != nil {
		return nil, nil, err
	}
	if conn == nil {
		return nil, nil, ErrConnectionNotFound
	}
	accessToken, err := s.tokens.AccessToken(ctx, vaID)
	if err != nil {
		return nil, nil, err
	}
	aircraftRows, err := s.live.ListOrganizationAircraft(ctx, accessToken, orgID)
	if err != nil {
		return nil, nil, err
	}
	aircraftIDs, regByAircraftID := matchAircraftIDsFromRows(aircraftRows, wantReg)
	if len(aircraftIDs) == 0 {
		return nil, nil, ErrRegistrationNotFound
	}
	return aircraftIDs, regByAircraftID, nil
}

func matchAircraftIDsFromRows(rows []infiniteflight.LiveAircraft, wantReg string) ([]string, map[string]string) {
	aircraftIDs := make([]string, 0)
	regByAircraftID := make(map[string]string)
	for _, row := range rows {
		if !strings.EqualFold(strings.TrimSpace(row.Registration), wantReg) {
			continue
		}
		aircraftIDs = append(aircraftIDs, row.ID)
		regByAircraftID[row.ID] = strings.TrimSpace(row.Registration)
	}
	return aircraftIDs, regByAircraftID
}

func (s *AssignCaptainService) AssignCaptain(ctx context.Context, input AssignCaptainInput) (AssignCaptainResult, error) {
	var empty AssignCaptainResult
	vaID := strings.TrimSpace(input.VAID)
	orgID := strings.TrimSpace(input.OrganizationID)
	if vaID == "" || orgID == "" {
		return empty, fmt.Errorf("va id and organization id are required")
	}
	wantReg := strings.TrimSpace(input.Registration)
	if wantReg == "" {
		return empty, fmt.Errorf("registration is required")
	}
	origin, dest, err := proof.ParseRouteProof(input.Route)
	if err != nil {
		return empty, err
	}
	if s.connections == nil || s.tokens == nil {
		return empty, fmt.Errorf("live assign captain is not configured")
	}
	if s.live == nil {
		return empty, ErrLiveClientDisabled
	}

	aircraftIDs, regByAircraftID, err := s.ResolveAircraftIDs(ctx, vaID, orgID, wantReg)
	if err != nil {
		return empty, err
	}

	accessToken, err := s.tokens.AccessToken(ctx, vaID)
	if err != nil {
		return empty, err
	}

	match, err := findFirstMatchingSchedule(ctx, s.live, accessToken, orgID, aircraftIDs, origin, dest)
	if err != nil {
		return empty, err
	}
	if match == nil {
		return empty, ErrScheduleNotFound
	}

	registration := regByAircraftID[match.AircraftID]
	if registration == "" {
		registration = strings.ToUpper(wantReg)
	}

	roleCode, err := infiniteflight.ParsePersistentFlightCrewMemberRole("Captain")
	if err != nil {
		return empty, err
	}
	body, err := json.Marshal(map[string]int{"role": roleCode})
	if err != nil {
		return empty, err
	}

	scheduleID := match.ID
	orgCopy := orgID
	meta := map[string]any{
		"registration":      registration,
		"route":             strings.TrimSpace(input.Route),
		"scheduleId":        scheduleID,
		"callsign":          strings.TrimSpace(match.Callsign),
		"originIcao":        strings.TrimSpace(match.OriginIcao),
		"destinationIcao":   strings.TrimSpace(match.DestinationIcao),
	}
	if err := s.tokens.Write(ctx, vaID, input.ActorUserID, &orgCopy, ActionAssignScheduleCrew, meta, func(token string) error {
		return s.live.AssignScheduleCrew(ctx, token, scheduleID, strings.TrimSpace(input.IFUserID), body)
	}); err != nil {
		return empty, err
	}

	return AssignCaptainResult{
		ScheduleID:      scheduleID,
		OrganizationID:  orgID,
		Callsign:          strings.TrimSpace(match.Callsign),
		Registration:      registration,
		OriginIcao:        strings.TrimSpace(match.OriginIcao),
		DestinationIcao:   strings.TrimSpace(match.DestinationIcao),
	}, nil
}

func findFirstMatchingSchedule(
	ctx context.Context,
	live ScheduleCrewLiveClient,
	accessToken, orgID string,
	aircraftIDs []string,
	origin, destination string,
) (*infiniteflight.LiveSchedule, error) {
	wantOrigin := strings.ToUpper(strings.TrimSpace(origin))
	wantDest := strings.ToUpper(strings.TrimSpace(destination))
	const pageSize = 100
	for page := 1; ; page++ {
		result, err := live.ListOrganizationSchedules(ctx, accessToken, orgID, page, pageSize, aircraftIDs)
		if err != nil {
			return nil, err
		}
		for i := range result.Items {
			item := result.Items[i]
			if strings.ToUpper(strings.TrimSpace(item.OriginIcao)) == wantOrigin &&
				strings.ToUpper(strings.TrimSpace(item.DestinationIcao)) == wantDest {
				return &item, nil
			}
		}
		if page >= result.TotalPages || result.TotalPages < 1 {
			break
		}
	}
	return nil, nil
}
