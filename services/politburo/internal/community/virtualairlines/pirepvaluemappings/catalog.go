package pirepvaluemappings

import (
	"sort"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type RuleStatus string

const (
	StatusExplicit         RuleStatus = "explicit"
	StatusUsesDefault      RuleStatus = "uses_default"
	StatusNeedsValue       RuleStatus = "needs_value"
	StatusAbsentFromCatalog RuleStatus = "absent_from_catalog"
)

type NameGroup struct {
	Name        string   `json:"name"`
	RowCount    int      `json:"rowCount"`
	AircraftIDs []string `json:"aircraftIds,omitempty"`
	Status      RuleStatus `json:"status"`
	FilingValue string   `json:"filingValue,omitempty"`
}

func BuildCatalogNames(rows []gameliveries.AircraftLivery) CatalogNames {
	out := CatalogNames{
		Aircraft: map[string]struct{}{},
		Airline:  map[string]struct{}{},
	}
	for _, row := range rows {
		if row.AircraftName != "" {
			out.Aircraft[row.AircraftName] = struct{}{}
		}
		if row.LiveryName != "" {
			out.Airline[row.LiveryName] = struct{}{}
		}
	}
	return out
}

func AircraftNameGroups(rows []gameliveries.AircraftLivery, cfg Config) []NameGroup {
	byName := map[string]*NameGroup{}
	for _, row := range rows {
		if row.AircraftName == "" {
			continue
		}
		g, ok := byName[row.AircraftName]
		if !ok {
			g = &NameGroup{Name: row.AircraftName}
			byName[row.AircraftName] = g
		}
		g.RowCount++
	}
	return finalizeGroups(byName, cfg.AircraftMappings, cfg.DefaultAircraft, true)
}

func AirlineNameGroups(rows []gameliveries.AircraftLivery, cfg Config) []NameGroup {
	byName := map[string]*NameGroup{}
	seenAircraft := map[string]map[string]struct{}{}
	for _, row := range rows {
		if row.LiveryName == "" {
			continue
		}
		g, ok := byName[row.LiveryName]
		if !ok {
			g = &NameGroup{Name: row.LiveryName}
			byName[row.LiveryName] = g
			seenAircraft[row.LiveryName] = map[string]struct{}{}
		}
		g.RowCount++
		if row.AircraftID != "" {
			seenAircraft[row.LiveryName][row.AircraftID] = struct{}{}
		}
	}
	for name, g := range byName {
		for id := range seenAircraft[name] {
			g.AircraftIDs = append(g.AircraftIDs, id)
		}
		sort.Strings(g.AircraftIDs)
	}
	return finalizeGroups(byName, cfg.AirlineMappings, cfg.DefaultAirline, false)
}

func finalizeGroups(byName map[string]*NameGroup, rules map[string]string, fallback *string, _ bool) []NameGroup {
	out := make([]NameGroup, 0, len(byName))
	for _, g := range byName {
		if v, ok := rules[g.Name]; ok {
			g.Status = StatusExplicit
			g.FilingValue = v
		} else if fallback != nil {
			g.Status = StatusUsesDefault
			g.FilingValue = *fallback
		} else {
			g.Status = StatusNeedsValue
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func RuleStatusForKey(name string, rules map[string]string, fallback *string, inCatalog bool) RuleStatus {
	if _, ok := rules[name]; ok {
		if !inCatalog {
			return StatusAbsentFromCatalog
		}
		return StatusExplicit
	}
	if fallback != nil {
		return StatusUsesDefault
	}
	return StatusNeedsValue
}