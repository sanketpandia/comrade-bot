// Package config validates and parses VA WEBHOOKS documents.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
)

const (
	FeatureActiveFlights = "active_flights"

	Frequency30m  = "30m"
	Frequency60m  = "60m"
	Frequency120m = "120m"
)

// Entry is one feature in a WEBHOOKS document.
type Entry struct {
	Key       string
	URL       string
	Frequency string
	Filters   json.RawMessage
}

// Validate checks a WEBHOOKS JSON document. Unknown feature keys must be objects
// but are otherwise accepted so new features can be stored before this build knows them.
func Validate(raw json.RawMessage) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return errors.New("empty body")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return errors.New("body must be an object")
	}
	for key, value := range top {
		if !jsonObject(value) {
			return fmt.Errorf("%s must be an object", key)
		}
		if key != FeatureActiveFlights {
			continue
		}
		if err := validateActiveFlights(value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return nil
}

// Parse decodes a stored WEBHOOKS document into feature entries sorted by key.
func Parse(raw json.RawMessage) ([]Entry, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("empty body")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, errors.New("body must be an object")
	}
	keys := make([]string, 0, len(top))
	for key := range top {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]Entry, 0, len(keys))
	for _, key := range keys {
		var probe struct {
			URL       string          `json:"url"`
			Frequency string          `json:"frequency"`
			Filters   json.RawMessage `json:"filters"`
		}
		if err := json.Unmarshal(top[key], &probe); err != nil {
			return nil, fmt.Errorf("feature %s: %w", key, err)
		}
		filters := probe.Filters
		if len(bytes.TrimSpace(filters)) == 0 {
			filters = json.RawMessage(`{}`)
		}
		out = append(out, Entry{
			Key:       key,
			URL:       strings.TrimSpace(probe.URL),
			Frequency: strings.TrimSpace(probe.Frequency),
			Filters:   filters,
		})
	}
	return out, nil
}

// Due reports whether a feature frequency should run at the given instant.
// The clock is converted to UTC. 30m is every tick, 60m only at minute 0,
// and 120m only at minute 0 of an even hour.
func Due(frequency string, at time.Time) bool {
	at = at.UTC()
	switch strings.TrimSpace(frequency) {
	case Frequency30m:
		return true
	case Frequency60m:
		return at.Minute() == 0
	case Frequency120m:
		return at.Minute() == 0 && at.Hour()%2 == 0
	default:
		return false
	}
}

func validateActiveFlights(raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return errors.New("must be an object")
	}
	urlRaw, ok := fields["url"]
	if !ok {
		return errors.New("url is required")
	}
	var webhookURL string
	if err := json.Unmarshal(urlRaw, &webhookURL); err != nil {
		return errors.New("url must be a string")
	}
	if !ValidDiscordWebhookURL(webhookURL) {
		return errors.New("url must be an https://discord.com/api/webhooks/{id}/{token} URL")
	}

	freqRaw, ok := fields["frequency"]
	if !ok {
		return errors.New("frequency is required")
	}
	var frequency string
	if err := json.Unmarshal(freqRaw, &frequency); err != nil {
		return errors.New("frequency must be a string")
	}
	switch strings.TrimSpace(frequency) {
	case Frequency30m, Frequency60m, Frequency120m:
	default:
		return errors.New("frequency must be 30m, 60m, or 120m")
	}

	filtersRaw, ok := fields["filters"]
	if !ok {
		return errors.New("filters is required")
	}
	return validateActiveFlightFilters(filtersRaw)
}

func validateActiveFlightFilters(raw json.RawMessage) error {
	if !jsonObject(raw) {
		return errors.New("filters must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return errors.New("filters must be an object")
	}
	if pilotRaw, ok := fields["pilotState"]; ok {
		var names []string
		if err := json.Unmarshal(pilotRaw, &names); err != nil {
			return errors.New("filters.pilotState must be an array of strings")
		}
		if _, err := flights.ParseFilterPilotStates(names); err != nil {
			return errors.New("filters.pilotState has an unknown pilot state")
		}
	}
	if codeRaw, ok := fields["vaCode"]; ok {
		var code string
		if err := json.Unmarshal(codeRaw, &code); err != nil {
			return errors.New("filters.vaCode must be a string")
		}
		if strings.TrimSpace(code) == "" {
			return errors.New("filters.vaCode must not be empty")
		}
	}
	return nil
}

// ValidDiscordWebhookURL reports whether raw is an HTTPS Discord webhook URL.
func ValidDiscordWebhookURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" || u.Port() != "" {
		return false
	}
	if !strings.EqualFold(u.Hostname(), "discord.com") {
		return false
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	parts := strings.Split(path, "/")
	return len(parts) == 5 && parts[1] == "api" && parts[2] == "webhooks" && parts[3] != "" && parts[4] != ""
}

func jsonObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
