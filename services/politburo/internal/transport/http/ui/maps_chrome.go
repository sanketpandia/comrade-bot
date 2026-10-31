package ui

import (
	"net/url"
	"strings"
)

// MapsChrome describes first-paint map page chrome from the query string.
type MapsChrome struct {
	Embed              bool
	ShowHeader         bool
	FiltersOpen        bool
	DetailRailEnabled  bool
	ShowMapOnlyButton  bool
	ShellClass         string
	BodyExtraClass     string
}

func ParseMapsChrome(values url.Values) MapsChrome {
	mode := strings.TrimSpace(values.Get("mode"))
	embed := mode == "embed"

	showHeader := true
	if embed {
		showHeader = false
	}
	if v := chromeFlag(values.Get("header")); v != nil {
		showHeader = *v
	}

	filtersOpen := true
	if v := chromeFlag(values.Get("filters")); v != nil {
		filtersOpen = *v
	}

	detailRailEnabled := true
	if v := chromeFlag(values.Get("detail")); v != nil {
		detailRailEnabled = *v
	}

	showMapOnly := !embed

	var shellParts []string
	if !filtersOpen {
		shellParts = append(shellParts, "maps-shell--filters-collapsed")
	}
	if !detailRailEnabled {
		shellParts = append(shellParts, "maps-shell--detail-disabled")
	}
	if embed {
		shellParts = append(shellParts, "maps-shell--embed")
	}

	var bodyParts []string
	if !showHeader {
		bodyParts = append(bodyParts, "maps-page--no-header")
	}

	return MapsChrome{
		Embed:             embed,
		ShowHeader:        showHeader,
		FiltersOpen:       filtersOpen,
		DetailRailEnabled: detailRailEnabled,
		ShowMapOnlyButton: showMapOnly,
		ShellClass:        strings.Join(shellParts, " "),
		BodyExtraClass:    strings.Join(bodyParts, " "),
	}
}

// chromeFlag interprets chrome query values. Only "1" (true) and "0" (false) apply.
func chromeFlag(raw string) *bool {
	switch strings.TrimSpace(raw) {
	case "1":
		v := true
		return &v
	case "0":
		v := false
		return &v
	default:
		return nil
	}
}

func mapsChromeHiddenFields(chrome MapsChrome, values url.Values) map[string]string {
	out := map[string]string{}
	if chrome.Embed {
		out["mode"] = "embed"
	}
	if v := strings.TrimSpace(values.Get("header")); v == "0" || v == "1" {
		out["header"] = v
	} else if !chrome.ShowHeader && !chrome.Embed {
		out["header"] = "0"
	}
	if v := strings.TrimSpace(values.Get("filters")); v == "0" || v == "1" {
		out["filters"] = v
	} else if !chrome.FiltersOpen {
		out["filters"] = "0"
	}
	if v := strings.TrimSpace(values.Get("detail")); v == "0" || v == "1" {
		out["detail"] = v
	}
	return out
}
