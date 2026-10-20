package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	valid := `{
		"active_flights": {
			"url": "https://discord.com/api/webhooks/123456789012345678/token",
			"frequency": "30m",
			"filters": {"pilotState": ["active", "away_in_flight"], "vaCode": "TEST"}
		}
	}`
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{name: "empty object", raw: `{}`, ok: true},
		{name: "active flights", raw: valid, ok: true},
		{name: "frequency 60m", raw: activeFlights(`"60m"`, `{}`), ok: true},
		{name: "frequency 120m", raw: activeFlights(`"120m"`, `{}`), ok: true},
		{name: "frequency 15m", raw: activeFlights(`"15m"`, `{}`), ok: false},
		{name: "http url", raw: `{
			"active_flights": {
				"url": "http://discord.com/api/webhooks/1/token",
				"frequency": "30m",
				"filters": {}
			}
		}`, ok: false},
		{name: "wrong host", raw: `{
			"active_flights": {
				"url": "https://example.com/api/webhooks/1/token",
				"frequency": "30m",
				"filters": {}
			}
		}`, ok: false},
		{name: "missing token", raw: `{
			"active_flights": {
				"url": "https://discord.com/api/webhooks/123",
				"frequency": "30m",
				"filters": {}
			}
		}`, ok: false},
		{name: "bad pilot state", raw: activeFlights(`"30m"`, `{"pilotState":["flying"]}`), ok: false},
		{name: "duplicate pilot states", raw: activeFlights(`"30m"`, `{"pilotState":["active","active"]}`), ok: true},
		{name: "empty va code", raw: activeFlights(`"30m"`, `{"vaCode":" "}`), ok: false},
		{name: "missing filters", raw: `{
			"active_flights": {
				"url": "https://discord.com/api/webhooks/123/token",
				"frequency": "30m"
			}
		}`, ok: false},
		{name: "filters array", raw: activeFlights(`"30m"`, `["active"]`), ok: false},
		{name: "unknown feature", raw: `{"future_feature":{"anything":true}}`, ok: true},
		{name: "unknown feature not object", raw: `{"future_feature":[]}`, ok: false},
		{name: "array body", raw: `[]`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(json.RawMessage(tt.raw))
			if tt.ok && err != nil {
				t.Fatalf("Validate() = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDue(t *testing.T) {
	hour14 := time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC)
	hour14Half := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	hour15 := time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)
	ist := time.FixedZone("IST", 5*3600+30*60)
	fromIST := time.Date(2026, 10, 20, 19, 30, 0, 0, ist) // 14:00 UTC

	if !Due(Frequency30m, hour14Half) || !Due(Frequency30m, hour15) {
		t.Fatal("30m should be due on every tick")
	}
	if Due(Frequency60m, hour14Half) {
		t.Fatal("60m should skip minute 30")
	}
	if !Due(Frequency60m, hour14) || !Due(Frequency60m, hour15) {
		t.Fatal("60m should run at minute 0")
	}
	if !Due(Frequency120m, hour14) || !Due(Frequency120m, fromIST) {
		t.Fatal("120m should run at minute 0 of an even UTC hour")
	}
	if Due(Frequency120m, hour15) || Due(Frequency120m, hour14Half) {
		t.Fatal("120m should skip odd hours and minute 30")
	}
	if Due("15m", hour14) {
		t.Fatal("unknown frequency should not be due")
	}
}

func activeFlights(frequency, filters string) string {
	return `{
		"active_flights": {
			"url": "https://discord.com/api/webhooks/123/token",
			"frequency": ` + frequency + `,
			"filters": ` + filters + `
		}
	}`
}
