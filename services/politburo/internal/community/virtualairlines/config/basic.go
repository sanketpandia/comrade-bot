package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Basic is the typed shape of a BASIC config block. Unknown keys are allowed and ignored.
type Basic struct {
	GameCallsign     GameCallsign `json:"game_callsign"`
	PreferredServers []string     `json:"preferred_servers"`
}

// GameCallsign holds the case-sensitive callsign affix rule for live flight matching.
type GameCallsign struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// ParseBasic decodes and validates a BASIC block. Affixes are trimmed; preferred servers
// are normalized to lowercase session names, deduplicated, and sorted.
func ParseBasic(raw json.RawMessage) (Basic, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Basic{}, fmt.Errorf("%w: empty body", ErrInvalidConfig)
	}
	var basic Basic
	if err := json.Unmarshal(raw, &basic); err != nil {
		return Basic{}, fmt.Errorf("%w: %s", ErrInvalidConfig, describeDecodeError(err))
	}
	basic.GameCallsign.Prefix = strings.TrimSpace(basic.GameCallsign.Prefix)
	basic.GameCallsign.Suffix = strings.TrimSpace(basic.GameCallsign.Suffix)

	seen := map[string]bool{}
	servers := make([]string, 0, len(basic.PreferredServers))
	for i, server := range basic.PreferredServers {
		name := strings.ToLower(strings.TrimSpace(server))
		if name == "" {
			return Basic{}, fmt.Errorf("%w: preferred_servers[%d] must not be empty", ErrInvalidConfig, i)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		servers = append(servers, name)
	}
	sort.Strings(servers)
	basic.PreferredServers = nil
	if len(servers) > 0 {
		basic.PreferredServers = servers
	}
	return basic, nil
}

// ValidateBasic reports whether raw is a valid BASIC block.
func ValidateBasic(raw json.RawMessage) error {
	_, err := ParseBasic(raw)
	return err
}

func describeDecodeError(err error) string {
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		return "body must be a JSON object"
	}
	field := typeErr.Field
	if field == "" {
		field = "body"
	}
	switch typeErr.Type.Kind() {
	case reflect.String:
		return field + " must be a string"
	case reflect.Slice:
		return field + " must be an array"
	case reflect.Struct:
		return field + " must be an object"
	default:
		return field + " has an invalid type"
	}
}
