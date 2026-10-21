// Package activeflights builds and sends the active_flights Discord webhook.
package activeflights

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/discord"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
)

const FeatureKey = webhookconfig.FeatureActiveFlights

type flightLister interface {
	ListActiveTracks(ctx context.Context, input flights.ListActiveTracksInput) (flights.ListActiveTracksResult, error)
	GetActiveFlight(ctx context.Context, flightID string) (flights.Record, error)
}

type sessionLister interface {
	ListActive(ctx context.Context) (sessions.ListActiveResult, error)
}

type poster interface {
	PostEmbeds(ctx context.Context, webhookType, webhookURL string, embeds []discord.Embed) error
}

type collectedFlight struct {
	flight flights.Record
}

// Handler queries live flights for one VA and posts packed Discord messages.
type Handler struct {
	flights  flightLister
	sessions sessionLister
	discord  poster
	metrics  *metrics.Registry
	now      func() time.Time
}

func New(flights flightLister, sessions sessionLister, discord poster, reg *metrics.Registry) *Handler {
	return &Handler{
		flights:  flights,
		sessions: sessions,
		discord:  discord,
		metrics:  reg,
		now:      time.Now,
	}
}

func (h *Handler) Handle(ctx context.Context, item dispatch.Item) dispatch.Result {
	if len(item.PreferredServers) == 0 {
		slog.Warn("skip active_flights webhook; no preferred servers", "va_id", item.VAID, "webhook_type", FeatureKey)
		return dispatch.Result{Outcome: dispatch.OutcomeSkipped, SkipReason: dispatch.SkipNoPreferredServers}
	}

	active, err := h.sessions.ListActive(ctx)
	if err != nil {
		return errorResult(err)
	}
	servers := onlinePreferred(item.PreferredServers, active.Sessions)

	query, err := filterQuery(item.Filters)
	if err != nil {
		return errorResult(err)
	}

	matched, loaded, err := h.collect(ctx, servers, query)
	if err != nil {
		return errorResult(err)
	}
	reported := len(loaded)
	capped := matched > maxReportedFlights
	h.metrics.ObserveActiveFlightsDispatch(FeatureKey, len(servers), matched, reported, capped)

	at := h.now().UTC()
	flightList := make([]flights.Record, len(loaded))
	for i, flight := range loaded {
		flightList[i] = flight.flight
	}
	embeds := BuildPosts(SnapshotInput{
		At:        at,
		Servers:   servers,
		Matched:   matched,
		Reported:  reported,
		Capped:    capped,
		Frequency: item.Frequency,
		Filters:   item.Filters,
		Flights:   flightList,
	})

	sent := 0
	for _, embed := range embeds {
		if err := h.discord.PostEmbeds(ctx, FeatureKey, item.URL, []discord.Embed{embed}); err != nil {
			slog.Warn("discord webhook post failed", "va_id", item.VAID, "webhook_type", FeatureKey, "error", err)
			return dispatch.Result{
				Outcome:         dispatch.OutcomeError,
				DiscordMessages: sent,
				ErrorMessage:    err.Error(),
			}
		}
		sent++
	}
	return dispatch.Result{Outcome: dispatch.OutcomeSuccess, DiscordMessages: sent}
}

func filterQuery(raw json.RawMessage) (flights.FilterQuery, error) {
	var filters struct {
		PilotState []string `json:"pilotState"`
		VACode     string   `json:"vaCode"`
	}
	if len(strings.TrimSpace(string(raw))) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &filters); err != nil {
			return flights.FilterQuery{}, err
		}
	}
	states, err := flights.ParseFilterPilotStates(filters.PilotState)
	if err != nil {
		return flights.FilterQuery{}, err
	}
	return flights.FilterQuery{
		PilotStates: states,
		VACode:      strings.ToUpper(strings.TrimSpace(filters.VACode)),
	}, nil
}

func (h *Handler) collect(ctx context.Context, servers []string, query flights.FilterQuery) (int, []collectedFlight, error) {
	matched := 0
	var loaded []collectedFlight
	for _, server := range servers {
		result, err := h.flights.ListActiveTracks(ctx, flights.ListActiveTracksInput{
			Server:     server,
			PageNumber: flights.DefaultPageNumber,
			PageLength: flights.MaxPageLength,
			Filters:    query,
		})
		if err != nil {
			return 0, nil, err
		}
		matched += result.TotalLength
		if len(loaded) >= maxReportedFlights {
			continue
		}
		for _, track := range result.Tracks {
			if len(loaded) >= maxReportedFlights {
				break
			}
			flight, err := h.flights.GetActiveFlight(ctx, track.FlightID)
			if errors.Is(err, flights.ErrCacheMiss) {
				continue
			}
			if err != nil {
				return 0, nil, err
			}
			loaded = append(loaded, collectedFlight{flight: flight})
		}
	}
	return matched, loaded, nil
}

func onlinePreferred(preferred []string, active []sessions.ActiveSession) []string {
	online := make(map[string]struct{}, len(active))
	for _, session := range active {
		name := strings.ToLower(strings.TrimSpace(session.NormalizedName))
		if name != "" {
			online[name] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, server := range preferred {
		name := strings.ToLower(strings.TrimSpace(server))
		if name == "" {
			continue
		}
		if _, ok := online[name]; !ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func errorResult(err error) dispatch.Result {
	return dispatch.Result{Outcome: dispatch.OutcomeError, ErrorMessage: err.Error()}
}
