package gameflights

import (
	"errors"
	"net/http"
	"strings"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

var errServerNameRequired = errors.New("normalizedServerName is required")

// ListActiveTracksInput builds a domain list request from the HTTP request and OpenAPI query params.
func ListActiveTracksInput(r *http.Request, normalizedServerName string, params politburoapi.GetActiveFlightsParams) (domainflights.ListActiveTracksInput, error) {
	if normalizedServerName == "" {
		return domainflights.ListActiveTracksInput{}, errServerNameRequired
	}
	pageNumber, pageLength, err := domainflights.ParsePaginationQuery(r.URL.Query())
	if err != nil {
		return domainflights.ListActiveTracksInput{}, err
	}
	filterQuery, err := filterQueryFromParams(params)
	if err != nil {
		return domainflights.ListActiveTracksInput{}, err
	}
	return domainflights.ListActiveTracksInput{
		Server:     normalizedServerName,
		PageNumber: pageNumber,
		PageLength: pageLength,
		Filters:    filterQuery,
	}, nil
}

func filterQueryFromParams(params politburoapi.GetActiveFlightsParams) (domainflights.FilterQuery, error) {
	var pilotNames []string
	if params.PilotState != nil {
		pilotNames = make([]string, 0, len(*params.PilotState))
		for _, state := range *params.PilotState {
			pilotNames = append(pilotNames, string(state))
		}
	}
	states, err := domainflights.ParseFilterPilotStates(pilotNames)
	if err != nil {
		return domainflights.FilterQuery{}, err
	}
	userName := ""
	if params.UserName != nil {
		userName = *params.UserName
	}
	callSign := ""
	if params.CallSign != nil {
		callSign = *params.CallSign
	}
	vaCode := ""
	if params.VaCode != nil {
		vaCode = strings.TrimSpace(*params.VaCode)
	}
	userName, callSign = domainflights.NormalizeFilterStrings(userName, callSign)
	return domainflights.FilterQuery{
		PilotStates: states,
		UserName:    userName,
		CallSign:    callSign,
		VACode:      vaCode,
	}, nil
}

func isInvalidListRequest(err error) bool {
	return errors.Is(err, errServerNameRequired) ||
		errors.Is(err, domainflights.ErrInvalidFilter) ||
		errors.Is(err, domainflights.ErrUnknownVACode)
}
