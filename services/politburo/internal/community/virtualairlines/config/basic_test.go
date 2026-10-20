package config

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateBasicAcceptsGameCallsign(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":"IE","suffix":"Heavy"}}`)
	if err := ValidateBasic(raw); err != nil {
		t.Fatalf("ValidateBasic() = %v", err)
	}
}

func TestValidateBasicAllowsExtraKeys(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":"IE"},"future_flag":true}`)
	if err := ValidateBasic(raw); err != nil {
		t.Fatalf("ValidateBasic() = %v", err)
	}
}

func TestValidateBasicRejectsNonStringPrefix(t *testing.T) {
	raw := json.RawMessage(`{"game_callsign":{"prefix":1}}`)
	err := ValidateBasic(raw)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
	if err.Error() != "invalid config: game_callsign.prefix must be a string" {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestValidateBasicAcceptsPreferredServers(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":["expert","casual"]}`)
	if err := ValidateBasic(raw); err != nil {
		t.Fatalf("ValidateBasic() = %v", err)
	}
}

func TestValidateBasicRejectsPreferredServersNotArray(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":"expert"}`)
	if err := ValidateBasic(raw); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
}

func TestValidateBasicRejectsEmptyPreferredServerEntry(t *testing.T) {
	raw := json.RawMessage(`{"preferred_servers":["expert",""]}`)
	if err := ValidateBasic(raw); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
}

func TestValidateBasicRejectsNonObjectBody(t *testing.T) {
	if err := ValidateBasic(json.RawMessage(`[]`)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
}

func TestParseBasicNormalizes(t *testing.T) {
	basic, err := ParseBasic(json.RawMessage(`{
		"game_callsign":{"prefix":" IE ","suffix":""},
		"preferred_servers":[" Expert ","expert","casual"]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if basic.GameCallsign.Prefix != "IE" {
		t.Fatalf("prefix = %q", basic.GameCallsign.Prefix)
	}
	if len(basic.PreferredServers) != 2 || basic.PreferredServers[0] != "casual" || basic.PreferredServers[1] != "expert" {
		t.Fatalf("preferred servers = %#v", basic.PreferredServers)
	}
}
