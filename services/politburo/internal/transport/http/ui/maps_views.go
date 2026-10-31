package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

type mapsPageView struct {
	ServerID         string
	Chrome           MapsChrome
	ChromeHidden     map[string]string
	CallSign         string
	UserName         string
	VACode           string
	PilotStates      []string
	ShowCallsigns    bool
	DetailFlightID   string
}

type mapsFlightsPartialView struct {
	Count         int
	CacheRelative string
	CacheISO      string
	Error         string
	TracksJSON    template.JS
}

type mapsFlightDetailView struct {
	FlightID    string
	Tab         string
	NotFound    bool
	GoneMessage string
	Record      *mapFlightRecordView
}

type mapFlightRecordView struct {
	Callsign         string
	Username         string
	AircraftName     string
	LiveryName       string
	Server           string
	Speed            string
	Altitude         string
	VerticalSpeed    string
	VSpeedTrend      string
	VSpeedTrendClass string
	Track            string
	PilotState       string
	PilotStateClass  string
	Position         string
	Latitude         string
	Longitude        string
	Analytics        *mapFlightAnalyticsView
	DebugKeys        []mapDebugKeyView
}

type mapFlightAnalyticsView struct {
	Phase              string
	PhaseSince         string
	PhaseSinceISO      string
	PhaseSinceRelative string
	PreviousPhase      string
	ConfidenceLevel    string
	ConfidencePercent  string
	Activity           string
	APPlusCandidate    bool
	APPlusScore        string
	APPlusSince        string
	APPlusReasons      []string
}

// mapDebugKeyView is one copyable identifier. Source is "IF" for Infinite Flight
// identifiers and "CB" for Comrade Bot database identifiers.
type mapDebugKeyView struct {
	Source string
	Label  string
	Value  string
}

func mapsPageViewFromRequest(r *http.Request) mapsPageView {
	q := r.URL.Query()
	states := append([]string(nil), q["pilotState"]...)
	showCallsigns := true
	if v := strings.TrimSpace(q.Get("showCallsigns")); v == "0" {
		showCallsigns = false
	}
	chrome := ParseMapsChrome(q)
	return mapsPageView{
		ServerID:       strings.TrimSpace(q.Get("serverId")),
		Chrome:         chrome,
		ChromeHidden:   mapsChromeHiddenFields(chrome, q),
		CallSign:       strings.TrimSpace(q.Get("callSign")),
		UserName:       strings.TrimSpace(q.Get("userName")),
		VACode:         strings.TrimSpace(q.Get("vaCode")),
		PilotStates:    states,
		ShowCallsigns:  showCallsigns,
		DetailFlightID: strings.TrimSpace(q.Get("flightId")),
	}
}

func mapFiltersFromRequest(r *http.Request) (domainflights.FilterQuery, string, bool, error) {
	q := r.URL.Query()
	server := strings.TrimSpace(q.Get("serverId"))
	states, err := domainflights.ParseFilterPilotStates(q["pilotState"])
	if err != nil {
		return domainflights.FilterQuery{}, "", true, err
	}
	userName, callSign := domainflights.NormalizeFilterStrings(
		strings.TrimSpace(q.Get("userName")),
		strings.TrimSpace(q.Get("callSign")),
	)
	vaCode := strings.ToUpper(strings.TrimSpace(q.Get("vaCode")))
	showCallsigns := true
	if strings.TrimSpace(q.Get("showCallsigns")) == "0" {
		showCallsigns = false
	}
	return domainflights.FilterQuery{
		PilotStates: states,
		UserName:    userName,
		CallSign:    callSign,
		VACode:      vaCode,
	}, server, showCallsigns, nil
}

func listAllTracks(r *http.Request, reader *domainflights.Reader, server string, filters domainflights.FilterQuery) (domainflights.ListActiveTracksResult, []domainflights.Track, error) {
	if reader == nil || server == "" {
		return domainflights.ListActiveTracksResult{}, nil, nil
	}
	const pageLength = domainflights.MaxPageLength
	all := make([]domainflights.Track, 0)
	var merged domainflights.ListActiveTracksResult
	pageNumber := 1
	for {
		result, err := reader.ListActiveTracks(r.Context(), domainflights.ListActiveTracksInput{
			Server:     server,
			PageNumber: pageNumber,
			PageLength: pageLength,
			Filters:    filters,
		})
		if err != nil {
			return domainflights.ListActiveTracksResult{}, nil, err
		}
		if pageNumber == 1 {
			merged = result
		}
		all = append(all, result.Tracks...)
		if len(all) >= result.TotalLength || len(result.Tracks) == 0 {
			merged.TotalLength = result.TotalLength
			break
		}
		pageNumber++
	}
	return merged, all, nil
}

func flightsPartialView(tracks []domainflights.Track, meta time.Time, metaValid bool, total int, errMsg string) mapsFlightsPartialView {
	type trackJSON struct {
		FlightID  string  `json:"flightId"`
		Callsign  string  `json:"callsign"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Speed     int     `json:"speed"`
		Track     float64 `json:"track,omitempty"`
	}
	out := make([]trackJSON, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, trackJSON{
			FlightID:  t.FlightID,
			Callsign:  t.Callsign,
			Latitude:  t.Latitude,
			Longitude: t.Longitude,
			Speed:     t.Speed,
			Track:     t.Track,
		})
	}
	raw, _ := json.Marshal(out)
	view := mapsFlightsPartialView{
		Count:      total,
		TracksJSON: template.JS(raw),
	}
	if errMsg != "" {
		view.Error = errMsg
		return view
	}
	if metaValid && !meta.IsZero() {
		view.CacheISO = meta.UTC().Format(time.RFC3339)
		view.CacheRelative = relativeTime(meta)
	}
	return view
}

func relativeTime(at time.Time) string {
	d := time.Since(at.UTC())
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d / time.Minute)
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	default:
		h := int(d / time.Hour)
		if h == 1 {
			return "1 hr ago"
		}
		return fmt.Sprintf("%d hr ago", h)
	}
}

func recordViewFromDomain(record domainflights.Record) mapFlightRecordView {
	raw := record.Raw
	derived := record.Derived
	vsTrend, vsClass := verticalTrend(raw.VerticalSpeed)
	pilotLabel := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(derived.Normalized.PilotState)), "_", " ")
	if pilotLabel == "" {
		pilotLabel = "UNKNOWN"
	}
	speed := derived.Normalized.Speed
	if speed == "" {
		speed = fmt.Sprintf("%d kts", domainflights.RoundedSpeed(raw.Speed))
	}
	vspeed := derived.Normalized.VerticalSpeed
	if vspeed == "" {
		vspeed = fmt.Sprintf("%.0f ft/min", raw.VerticalSpeed)
	}
	lat := fmt.Sprintf("%.4f", domainflights.RoundedLatitude(raw.Latitude))
	lon := fmt.Sprintf("%.4f", domainflights.RoundedLongitude(raw.Longitude))
	view := mapFlightRecordView{
		Callsign:         raw.Callsign,
		Username:         stringOrEmpty(raw.Username),
		AircraftName:     derived.AircraftName,
		LiveryName:       derived.LiveryName,
		Server:           serverLabel(raw.NormalizedName),
		Speed:            speed,
		Altitude:         fmt.Sprintf("%d ft", domainflights.RoundedAltitude(raw.Altitude)),
		VerticalSpeed:    vspeed,
		VSpeedTrend:      vsTrend,
		VSpeedTrendClass: vsClass,
		Track:            fmt.Sprintf("%.0f°", domainflights.RoundedTrack(raw.Track)),
		PilotState:       pilotLabel,
		PilotStateClass:  pilotStateBadgeClass(derived.Normalized.PilotState),
		Position:         lat + ", " + lon,
		Latitude:         lat,
		Longitude:        lon,
		DebugKeys:        debugKeysFromRecord(record),
	}
	if derived.FlightState != nil {
		view.Analytics = analyticsFromSummary(*derived.FlightState)
	}
	return view
}

func analyticsFromSummary(state flightstate.Summary) *mapFlightAnalyticsView {
	out := &mapFlightAnalyticsView{
		Phase:             humanizeToken(string(state.Phase)),
		PreviousPhase:     humanizeToken(string(state.PreviousPhase)),
		ConfidenceLevel:   confidenceLevel(state.Confidence),
		ConfidencePercent: fmt.Sprintf("%.0f%%", state.Confidence*100),
		Activity:          humanizeToken(string(state.Activity)),
		APPlusCandidate:   state.APPlus.Candidate,
	}
	if !state.PhaseSince.IsZero() {
		out.PhaseSince = state.PhaseSince.UTC().Format("15:04:05 UTC")
		out.PhaseSinceISO = state.PhaseSince.UTC().Format(time.RFC3339)
		out.PhaseSinceRelative = relativeTime(state.PhaseSince)
	}
	if state.APPlus.Candidate {
		out.APPlusScore = fmt.Sprintf("%.0f%%", state.APPlus.Score*100)
		if !state.APPlus.Since.IsZero() {
			out.APPlusSince = state.APPlus.Since.UTC().Format("15:04:05 UTC")
		}
		for _, reason := range state.APPlus.Reasons {
			out.APPlusReasons = append(out.APPlusReasons, humanizeToken(reason))
		}
	}
	return out
}

func confidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.7:
		return "high"
	case confidence >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

func debugKeysFromRecord(record domainflights.Record) []mapDebugKeyView {
	raw := record.Raw
	keys := []mapDebugKeyView{
		{Source: "IF", Label: "Flight ID", Value: raw.FlightID},
		{Source: "IF", Label: "User ID", Value: raw.UserID},
		{Source: "IF", Label: "Session ID", Value: raw.SessionID},
	}
	for _, userID := range record.Derived.Users {
		if userID == "" || userID == raw.UserID {
			continue
		}
		keys = append(keys, mapDebugKeyView{Source: "IF", Label: "Previous user ID", Value: userID})
	}
	for _, match := range record.Derived.MatchedVirtualAirlines {
		label := "VA ID"
		if match.VACode != "" {
			label = "VA ID · " + match.VACode
		}
		keys = append(keys, mapDebugKeyView{Source: "CB", Label: label, Value: match.VAID})
	}
	keys = append(keys, mapDebugKeyView{Source: "IF", Label: "Pilot state code", Value: fmt.Sprintf("%d", raw.PilotState)})
	out := keys[:0]
	for _, key := range keys {
		if strings.TrimSpace(key.Value) != "" {
			out = append(out, key)
		}
	}
	return out
}

// humanizeToken turns snake_case enum values into sentence case labels.
func humanizeToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	words := strings.Split(strings.ReplaceAll(token, "_", " "), " ")
	for i := 0; i+1 < len(words); i++ {
		if words[i] == "ap" && words[i+1] == "plus" {
			words[i] = "AP+"
			words = append(words[:i+1], words[i+2:]...)
		}
	}
	label := strings.Join(words, " ")
	return strings.ToUpper(label[:1]) + label[1:]
}

func verticalTrend(speed float64) (label, className string) {
	switch {
	case speed > 50:
		return "CLIMBING", "badge--success"
	case speed < -50:
		return "DESCENDING", "badge--error"
	default:
		return "LEVEL", "badge--muted"
	}
}

func pilotStateBadgeClass(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return "badge--pilot-active"
	case "away_in_flight":
		return "badge--pilot-away-flight"
	case "away_parked":
		return "badge--pilot-away-parked"
	case "in_background":
		return "badge--pilot-background"
	default:
		return "badge--pilot-unknown"
	}
}

func formatUTC(at time.Time) string {
	if at.IsZero() {
		return "—"
	}
	return at.UTC().Format("2006-01-02 15:04:05 UTC")
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
