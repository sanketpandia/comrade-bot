package flights

import (
	"strings"
)

// FilterQuery holds validated active-flight list filters.
type FilterQuery struct {
	PilotStates []string
	UserName    string
	CallSign    string
}

// AvailableFilter describes one supported query filter for cache-backed list endpoints.
type AvailableFilter struct {
	Name    string
	Type    string
	Desc    string
	Current any
	Default any
	Options []string
}

// ParseFilterPilotStates deduplicates and validates pilotState query values.
func ParseFilterPilotStates(names []string) ([]string, error) {
	selected := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		parsed, ok := ParsePilotStateName(strings.TrimSpace(name))
		if !ok {
			return nil, ErrInvalidFilter
		}
		if _, exists := seen[parsed]; exists {
			continue
		}
		seen[parsed] = struct{}{}
		selected = append(selected, parsed)
	}
	return selected, nil
}

func NormalizeFilterStrings(userName, callSign string) (string, string) {
	return strings.TrimSpace(userName), strings.TrimSpace(callSign)
}

// MatchesFilters reports whether a flight record passes the filter query.
func MatchesFilters(flight Flight, query FilterQuery, pilotStateSeen map[string]struct{}) bool {
	if len(query.PilotStates) > 0 {
		if _, exists := pilotStateSeen[flight.Normalized.PilotState]; !exists {
			return false
		}
	}
	if !ContainsFold(Username(flight), query.UserName) {
		return false
	}
	if !ContainsFold(flight.Callsign, query.CallSign) {
		return false
	}
	return true
}

func BuildAvailableFilters(query FilterQuery) []AvailableFilter {
	return []AvailableFilter{
		{
			Name:    "pilotState",
			Type:    "multi",
			Desc:    "Restrict results to one or more pilot states",
			Current: query.PilotStates,
			Default: []string{},
			Options: PilotStateNames(),
		},
		{
			Name:    "userName",
			Type:    "string",
			Desc:    "Restrict results to flights whose username contains this value",
			Current: query.UserName,
			Default: "",
		},
		{
			Name:    "callSign",
			Type:    "string",
			Desc:    "Restrict results to flights whose callsign contains this value",
			Current: query.CallSign,
			Default: "",
		},
	}
}

func pilotStateSet(states []string) map[string]struct{} {
	seen := make(map[string]struct{}, len(states))
	for _, state := range states {
		seen[state] = struct{}{}
	}
	return seen
}
