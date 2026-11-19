package connections

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// MappedRecord is a logical projection from a provider row.
type MappedRecord struct {
	SourceID string         `json:"sourceId"`
	Fields   map[string]any `json:"fields"`
}

// FieldMeta is minimal schema metadata for normalization.
type FieldMeta struct {
	ID   string
	Name string
	Type string
}

// FieldIssue describes a non-fatal normalization problem for one logical key.
type FieldIssue struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

func NormalizeRecord(entity string, mapping Mapping, schemaByID map[string]FieldMeta, rawFields map[string]any) (MappedRecord, []FieldIssue, error) {
	out := map[string]any{}
	var issues []FieldIssue
	for logicalKey, fieldID := range mapping.Fields {
		cf, ok := catalogField(entity, logicalKey)
		if !ok {
			continue
		}
		meta, ok := schemaByID[fieldID]
		if !ok {
			return MappedRecord{}, nil, fmt.Errorf("%w: unknown field id %s", ErrSchemaDrift, fieldID)
		}
		cell, present := rawFields[meta.Name]
		if !present || cell == nil {
			continue
		}
		val, warn, err := normalizeCell(cf, cell, meta.Type, mapping.FieldUnits[logicalKey])
		if err != nil {
			issues = append(issues, FieldIssue{Key: cf.Key, Message: err.Error()})
			continue
		}
		if warn != "" {
			issues = append(issues, FieldIssue{Key: cf.Key, Message: warn})
		}
		if val == nil {
			continue
		}
		out[cf.Key] = val
	}
	return MappedRecord{Fields: out}, issues, nil
}

func normalizeCell(cf CatalogField, cell any, providerType string, unit DurationUnit) (any, string, error) {
	effective := providerType
	switch cf.Kind {
	case KindText:
		return normalizeTextCell(cf, cell, effective)
	case KindNumber:
		v, err := normalizeNumber(cell)
		return v, "", err
	case KindDuration:
		if effective == "number" || providerType == "number" {
			v, err := normalizeDurationWithUnit(cell, unit)
			return v, "", err
		}
		v, err := normalizeDuration(cell)
		return v, "", err
	case KindDate, KindInstant:
		v, err := normalizeDate(cell)
		return v, "", err
	case KindLinks:
		v, err := normalizeLinks(cell)
		return v, "", err
	default:
		return normalizeTextCell(cf, cell, effective)
	}
}

func normalizeTextCell(cf CatalogField, cell any, _ string) (any, string, error) {
	if cf.Cardinality == CardinalityList {
		return normalizeTextList(cell)
	}
	switch v := cell.(type) {
	case string:
		if v == "" {
			return nil, "", nil
		}
		return v, "", nil
	case map[string]any:
		if name, ok := v["name"].(string); ok && name != "" {
			return name, "", nil
		}
	case []any:
		strs, err := stringsFromAnySlice(v)
		if err != nil {
			return nil, "", err
		}
		if len(strs) == 0 {
			return nil, "", nil
		}
		if len(strs) > 1 {
			return strs[0], fmt.Sprintf("multiple values; using first of %d", len(strs)), nil
		}
		return strs[0], "", nil
	}
	return nil, "", fmt.Errorf("unsupported text cell")
}

func normalizeTextList(cell any) (any, string, error) {
	switch v := cell.(type) {
	case []any:
		strs, err := stringsFromAnySlice(v)
		if err != nil {
			return nil, "", err
		}
		if len(strs) == 0 {
			return nil, "", nil
		}
		return strs, "", nil
	case string:
		if v == "" {
			return nil, "", nil
		}
		return []string{v}, "", nil
	case map[string]any:
		if name, ok := v["name"].(string); ok && name != "" {
			return []string{name}, "", nil
		}
	}
	return nil, "", fmt.Errorf("unsupported list text cell")
}

func stringsFromAnySlice(items []any) ([]string, error) {
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch t := item.(type) {
		case string:
			if t != "" {
				out = append(out, t)
			}
		case map[string]any:
			if name, ok := t["name"].(string); ok && name != "" {
				out = append(out, name)
			}
		default:
			return nil, fmt.Errorf("unsupported list element")
		}
	}
	return out, nil
}

func normalizeNumber(cell any) (any, error) {
	switch v := cell.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil
		}
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	}
	return nil, fmt.Errorf("unsupported number cell")
}

func normalizeDuration(cell any) (any, error) {
	switch v := cell.(type) {
	case float64:
		return int64(math.Round(v)), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	}
	return nil, fmt.Errorf("unsupported duration cell")
}

func normalizeDurationWithUnit(cell any, unit DurationUnit) (any, error) {
	if unit == "" {
		unit = UnitSeconds
	}
	var n float64
	switch v := cell.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	default:
		return nil, fmt.Errorf("unsupported duration cell")
	}
	var seconds float64
	switch unit {
	case UnitMinutes:
		seconds = n * 60
	case UnitHours:
		seconds = n * 3600
	default:
		seconds = n
	}
	return int64(math.Round(seconds)), nil
}

func normalizeDate(cell any) (any, error) {
	s, ok := cell.(string)
	if !ok || s == "" {
		return nil, fmt.Errorf("unsupported date cell")
	}
	if strings.Contains(s, "T") {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
		return t.UTC(), nil
	}
	return s, nil
}

func normalizeLinks(cell any) (any, error) {
	switch v := cell.(type) {
	case []any:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
		if len(ids) == 0 {
			return nil, nil
		}
		return ids, nil
	case string:
		if v == "" {
			return nil, nil
		}
		return []string{v}, nil
	}
	return nil, fmt.Errorf("unsupported link cell")
}

func filterLiteralValue(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(t), nil
	default:
		return "", ErrInvalidFilter
	}
}