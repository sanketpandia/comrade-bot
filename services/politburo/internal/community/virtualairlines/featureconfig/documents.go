package featureconfig

import (
	"encoding/json"
	"fmt"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
)

const (
	documentVersion = 1
	FeatureStats    = "stats"
	FeaturePIREP    = "pirep"
)

// StatsDocument is persisted under MAP_STATS.
type StatsDocument struct {
	Version  int           `json:"version"`
	Revision int           `json:"revision"`
	Enabled  bool          `json:"enabled"`
	Pilot    *StatsSection `json:"pilot,omitempty"`
	Career   *StatsSection `json:"career,omitempty"`
}

type StatsSection struct {
	MatchBy string   `json:"matchBy"`
	Fields  []string `json:"fields"`
}

// PIREPDocument is persisted under MAP_PIREP.
type PIREPDocument struct {
	Version  int      `json:"version"`
	Revision int      `json:"revision"`
	Enabled  bool     `json:"enabled"`
	Fields   []string `json:"fields"`
}

func ConfigTypeForFeature(feature string) (string, error) {
	switch feature {
	case FeatureStats:
		return vaconfig.TypeMapStats, nil
	case FeaturePIREP:
		return vaconfig.TypeMapPIREP, nil
	default:
		return "", ErrInvalidFeature
	}
}

func ParseStatsDocument(raw json.RawMessage) (StatsDocument, error) {
	if len(raw) == 0 {
		return StatsDocument{Version: documentVersion, Revision: 0, Enabled: false}, nil
	}
	var doc StatsDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return StatsDocument{}, fmt.Errorf("decode stats document: %w", err)
	}
	if doc.Version != documentVersion {
		return StatsDocument{}, fmt.Errorf("unsupported stats document version %d", doc.Version)
	}
	return doc, nil
}

func (d StatsDocument) Marshal() (json.RawMessage, error) {
	d.Version = documentVersion
	out, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func ParsePIREPDocument(raw json.RawMessage) (PIREPDocument, error) {
	if len(raw) == 0 {
		return PIREPDocument{Version: documentVersion, Revision: 0, Enabled: false}, nil
	}
	var doc PIREPDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return PIREPDocument{}, fmt.Errorf("decode pirep document: %w", err)
	}
	if doc.Version != documentVersion {
		return PIREPDocument{}, fmt.Errorf("unsupported pirep document version %d", doc.Version)
	}
	return doc, nil
}

func (d PIREPDocument) Marshal() (json.RawMessage, error) {
	d.Version = documentVersion
	out, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}