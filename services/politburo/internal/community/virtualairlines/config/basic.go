package config

import (
	"encoding/json"
	"fmt"
)

func validateBasic(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("%w: empty body", ErrInvalidConfig)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if cs, ok := top["game_callsign"]; ok {
		if err := validateGameCallsign(cs); err != nil {
			return err
		}
	}
	return nil
}

func validateGameCallsign(raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("%w: game_callsign must be an object", ErrInvalidConfig)
	}
	for key, value := range fields {
		switch key {
		case "prefix", "suffix":
			if err := requireJSONString(value, "game_callsign."+key); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireJSONString(raw json.RawMessage, field string) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("%w: %s must be a string", ErrInvalidConfig, field)
	}
	return nil
}
