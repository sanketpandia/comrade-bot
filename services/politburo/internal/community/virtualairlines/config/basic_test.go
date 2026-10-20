package config

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateBasicAcceptsGameCallsign(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":"IE","suffix":"Heavy"}}`)
	if err := Validate(TypeBasic, raw); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidateBasicAllowsExtraKeys(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":"IE"},"future_flag":true}`)
	if err := Validate(TypeBasic, raw); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidateBasicRejectsNonStringPrefix(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":1}}`)
	if err := Validate(TypeBasic, raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateBasicAcceptsPreferredServers(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":["expert","casual"]}`)
	if err := Validate(TypeBasic, raw); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidateBasicRejectsPreferredServersNotArray(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":"expert"}`)
	if err := Validate(TypeBasic, raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateBasicRejectsEmptyPreferredServerEntry(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":["expert",""]}`)
	if err := Validate(TypeBasic, raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateWebhooksDelegates(t *testing.T) {
	raw := json.RawMessage(`{"active_flights":{"url":"https://discord.com/api/webhooks/123/token","frequency":"30m","filters":{"pilotState":["active"]}}}`)
	if err := Validate(TypeWebhooks, raw); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	bad := json.RawMessage(`{"active_flights":{"url":"http://discord.com/api/webhooks/123/token","frequency":"30m","filters":{}}}`)
	if err := Validate(TypeWebhooks, bad); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateUnknownType(t *testing.T) {
	raw := json.RawMessage(`{}`)
	if err := Validate("FUTURE", raw); err != ErrUnknownType {
		t.Fatalf("err = %v, want ErrUnknownType", err)
	}
}
