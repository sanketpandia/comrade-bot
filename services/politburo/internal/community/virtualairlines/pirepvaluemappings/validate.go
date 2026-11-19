package pirepvaluemappings

import (
	"encoding/json"
	"fmt"
	"strings"
)

var allowedTopLevel = map[string]struct{}{
	"version":          {},
	"defaults":           {},
	"AircraftMappings":   {},
	"AirlineMappings":    {},
}

func ValidateDocument(doc Document) error {
	if doc.Version != DocumentVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalid, doc.Version)
	}
	if doc.AircraftMappings == nil || doc.AirlineMappings == nil {
		return fmt.Errorf("%w: mappings must be objects", ErrInvalid)
	}
	if err := validateDefaults(doc.Defaults); err != nil {
		return err
	}
	if err := validateRules("AircraftMappings", doc.AircraftMappings); err != nil {
		return err
	}
	if err := validateRules("AirlineMappings", doc.AirlineMappings); err != nil {
		return err
	}
	return nil
}

func validateDefaults(d Defaults) error {
	if d.Aircraft != nil && strings.TrimSpace(*d.Aircraft) == "" {
		return fmt.Errorf("%w: defaults.aircraft must be nonblank or null", ErrInvalid)
	}
	if d.Airline != nil && strings.TrimSpace(*d.Airline) == "" {
		return fmt.Errorf("%w: defaults.airline must be nonblank or null", ErrInvalid)
	}
	return nil
}

func validateRules(path string, rules map[string]string) error {
	for k, v := range rules {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("%w: %s contains blank key", ErrInvalid, path)
		}
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%w: %s.%q has blank value", ErrInvalid, path, k)
		}
	}
	return nil
}

func rejectDuplicateKeys(raw []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	for k := range top {
		if _, ok := allowedTopLevel[k]; !ok {
			return fmt.Errorf("%w: unknown field %q", ErrInvalid, k)
		}
	}
	if err := checkObjectDuplicates(raw, top["AircraftMappings"]); err != nil {
		return fmt.Errorf("%w: AircraftMappings: %v", ErrInvalid, err)
	}
	if err := checkObjectDuplicates(raw, top["AirlineMappings"]); err != nil {
		return fmt.Errorf("%w: AirlineMappings: %v", ErrInvalid, err)
	}
	return nil
}

func checkObjectDuplicates(_ []byte, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("expected object")
	}
	seen := map[string]struct{}{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("invalid key type")
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate key %q", key)
		}
		seen[key] = struct{}{}
		var skip any
		if err := dec.Decode(&skip); err != nil {
			return err
		}
	}
	return nil
}