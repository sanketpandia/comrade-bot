package config

import (
	"encoding/json"
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

func TestValidateUnknownType(t *testing.T) {
	raw := json.RawMessage(`{}`)
	if err := Validate("FUTURE", raw); err != ErrUnknownType {
		t.Fatalf("err = %v, want ErrUnknownType", err)
	}
}
