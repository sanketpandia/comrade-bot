package featureconfig

import (
	"encoding/json"
	"testing"
)

func TestParseStatsDocumentFixture(t *testing.T) {
	raw := json.RawMessage(`{
		"version": 1,
		"revision": 3,
		"enabled": true,
		"pilot": {
			"matchBy": "callsign",
			"fields": ["displayName", "flightTimeSeconds", "category", "region", "rank"]
		},
		"career": {
			"matchBy": "callsign",
			"fields": ["flightTimeSeconds", "lastFlownRoute"]
		}
	}`)
	doc, err := ParseStatsDocument(raw)
	if err != nil {
		t.Fatalf("ParseStatsDocument() = %v", err)
	}
	if !doc.Enabled || doc.Revision != 3 || doc.Pilot == nil {
		t.Fatalf("doc = %+v", doc)
	}
}

func TestParsePIREPDocumentFixture(t *testing.T) {
	raw := json.RawMessage(`{
		"version": 1,
		"revision": 1,
		"enabled": true,
		"fields": ["pilotRefs", "routeRefs", "flightTimeSeconds", "remarks"]
	}`)
	doc, err := ParsePIREPDocument(raw)
	if err != nil {
		t.Fatalf("ParsePIREPDocument() = %v", err)
	}
	if !doc.Enabled || len(doc.Fields) != 4 {
		t.Fatalf("doc = %+v", doc)
	}
}