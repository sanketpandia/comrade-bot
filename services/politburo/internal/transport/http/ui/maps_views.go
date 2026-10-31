package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	domainflights "infinite-experiment/politburo/internal/livegame/flights"
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
	Callsign            string
	Username            string
	VirtualOrganization string
	AircraftName        string
	LiveryName          string
	SessionName         string
	Speed               string
	Altitude            string
	VerticalSpeed       string
	VSpeedTrend         string
	VSpeedTrendClass    string
	Track               string
	PilotState          string
	PilotStateClass     string
	Connection          string
	ConnectionClass     string
	FPLSyncRequired     bool
	LastReport          string
	FlightID            string
	UserID              string
	Position            string
	PilotStateCode      string
	SessionID           string
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
	lastReport := raw.LastReport
	if !raw.LastReportValid {
		lastReport = raw.ObservedAt
	}
	vsTrend, vsClass := verticalTrend(raw.VerticalSpeed)
	pilotLabel := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(derived.Normalized.PilotState)), "_", " ")
	if pilotLabel == "" {
		pilotLabel = "UNKNOWN"
	}
	connText := "DISCONNECTED"
	connClass := "maps-detail-badge--disconnected"
	if strings.EqualFold(strings.TrimSpace(derived.Normalized.IsConnected), "connected") {
		connText = "CONNECTED"
		connClass = "maps-detail-badge--connected"
	}
	speed := derived.Normalized.Speed
	if speed == "" {
		speed = fmt.Sprintf("%d kts", domainflights.RoundedSpeed(raw.Speed))
	}
	vspeed := derived.Normalized.VerticalSpeed
	if vspeed == "" {
		vspeed = fmt.Sprintf("%.0f ft/min", raw.VerticalSpeed)
	}
	aircraft := derived.AircraftName
	livery := derived.LiveryName
	return mapFlightRecordView{
		Callsign:            raw.Callsign,
		Username:            stringOrEmpty(raw.Username),
		VirtualOrganization: stringOrEmpty(raw.VirtualOrganization),
		AircraftName:        aircraft,
		LiveryName:          livery,
		SessionName:         raw.NormalizedName,
		Speed:               speed,
		Altitude:            fmt.Sprintf("%d ft", domainflights.RoundedAltitude(raw.Altitude)),
		VerticalSpeed:       vspeed,
		VSpeedTrend:         vsTrend,
		VSpeedTrendClass:    vsClass,
		Track:               fmt.Sprintf("%.0f°", domainflights.RoundedTrack(raw.Track)),
		PilotState:          pilotLabel,
		PilotStateClass:     pilotStateBadgeClass(derived.Normalized.PilotState),
		Connection:          connText,
		ConnectionClass:     connClass,
		FPLSyncRequired:     derived.PathSync.FPLSyncRequired,
		LastReport:          formatUTC(lastReport),
		FlightID:            raw.FlightID,
		UserID:              raw.UserID,
		Position: fmt.Sprintf("%s, %s",
			fmt.Sprintf("%.4f", domainflights.RoundedLatitude(raw.Latitude)),
			fmt.Sprintf("%.4f", domainflights.RoundedLongitude(raw.Longitude)),
		),
		PilotStateCode: fmt.Sprintf("%d", raw.PilotState),
		SessionID:      raw.SessionID,
	}
}

func verticalTrend(speed float64) (label, className string) {
	switch {
	case speed > 50:
		return "CLIMBING", "maps-detail-badge--climb"
	case speed < -50:
		return "DESCENDING", "maps-detail-badge--descend"
	default:
		return "LEVEL", "maps-detail-badge--level"
	}
}

func pilotStateBadgeClass(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return "maps-detail-badge--pilot-active"
	case "away_in_flight":
		return "maps-detail-badge--pilot-away-flight"
	case "away_parked":
		return "maps-detail-badge--pilot-away-parked"
	case "in_background":
		return "maps-detail-badge--pilot-background"
	default:
		return "maps-detail-badge--pilot-unknown"
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
