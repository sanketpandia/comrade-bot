package pirepvaluemappings

import gameliveries "infinite-experiment/politburo/internal/livegame/liveries"

type SourceKind string

const (
	SourceExplicit       SourceKind = "explicit"
	SourceDefault        SourceKind = "default"
	SourceNeedsValue     SourceKind = "needs_value"
	SourceCatalogMismatch SourceKind = "catalog_mismatch"
)

type ResolvedField struct {
	RawInput string     `json:"rawInput,omitempty"`
	Value    string     `json:"value,omitempty"`
	Source   SourceKind `json:"source"`
}

type ResolveInput struct {
	AircraftID string
	LiveryID   string
}

type ResolveResult struct {
	Aircraft        ResolvedField `json:"aircraft"`
	Airline         ResolvedField `json:"airline"`
	CatalogMismatch bool          `json:"catalogMismatch,omitempty"`
}

type Config struct {
	AircraftMappings map[string]string
	AirlineMappings  map[string]string
	DefaultAircraft  *string
	DefaultAirline   *string
}

func ConfigFromRow(row *Row) Config {
	if row == nil {
		return Config{
			AircraftMappings: map[string]string{},
			AirlineMappings:  map[string]string{},
		}
	}
	return Config{
		AircraftMappings: copyMap(row.AircraftMappings),
		AirlineMappings:  copyMap(row.AirlineMappings),
		DefaultAircraft:  row.DefaultAircraft,
		DefaultAirline:   row.DefaultAirline,
	}
}

func Resolve(cfg Config, livery gameliveries.AircraftLivery, input ResolveInput) ResolveResult {
	var out ResolveResult
	catalogMismatch := false

	aircraftName := livery.AircraftName
	liveryName := livery.LiveryName
	liveryKnown := livery.LiveryID != ""
	if liveryKnown && input.AircraftID != "" && livery.AircraftID != "" && livery.AircraftID != input.AircraftID {
		catalogMismatch = true
	}

	out.CatalogMismatch = catalogMismatch

	if aircraftName != "" {
		out.Aircraft = resolveMapping(aircraftName, cfg.AircraftMappings, cfg.DefaultAircraft)
	} else if input.AircraftID != "" || input.LiveryID != "" {
		out.Aircraft = ResolvedField{Source: SourceNeedsValue}
	}

	if catalogMismatch {
		out.Airline = ResolvedField{RawInput: liveryName, Source: SourceCatalogMismatch}
	} else if liveryName != "" {
		out.Airline = resolveMapping(liveryName, cfg.AirlineMappings, cfg.DefaultAirline)
	} else if liveryName == "" && (input.LiveryID != "" || input.AircraftID != "") {
		out.Airline = ResolvedField{Source: SourceNeedsValue}
	}

	return out
}

func resolveMapping(raw string, rules map[string]string, fallback *string) ResolvedField {
	if v, ok := rules[raw]; ok {
		return ResolvedField{RawInput: raw, Value: v, Source: SourceExplicit}
	}
	if fallback != nil {
		return ResolvedField{RawInput: raw, Value: *fallback, Source: SourceDefault}
	}
	return ResolvedField{RawInput: raw, Source: SourceNeedsValue}
}