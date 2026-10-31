package ui

import (
	"net/http"
	"net/url"
	"strings"

	"infinite-experiment/politburo/internal/routes"
)

// mapsActivePageURL builds the shareable map page URL from an HTMX partial request query.
func mapsActivePageURL(r *http.Request) string {
	q := r.URL.Query()
	params := url.Values{}

	if v := strings.TrimSpace(q.Get("serverId")); v != "" {
		params.Set("serverId", v)
	}
	for _, state := range q["pilotState"] {
		if s := strings.TrimSpace(state); s != "" {
			params.Add("pilotState", s)
		}
	}
	if v := strings.TrimSpace(q.Get("callSign")); v != "" {
		params.Set("callSign", v)
	}
	if v := strings.TrimSpace(q.Get("userName")); v != "" {
		params.Set("userName", v)
	}
	if v := strings.TrimSpace(q.Get("vaCode")); v != "" {
		params.Set("vaCode", v)
	}
	if strings.TrimSpace(q.Get("showCallsigns")) == "0" {
		params.Set("showCallsigns", "0")
	}
	if v := strings.TrimSpace(q.Get("flightId")); v != "" {
		params.Set("flightId", v)
	}
	if strings.TrimSpace(q.Get("mode")) == "embed" {
		params.Set("mode", "embed")
	}

	for _, key := range []string{"header", "filters", "detail"} {
		v := strings.TrimSpace(q.Get(key))
		if v == "0" || v == "1" {
			params.Set(key, v)
		}
	}

	encoded := params.Encode()
	if encoded == "" {
		return routes.MapsFlightsActive
	}
	return routes.MapsFlightsActive + "?" + encoded
}
