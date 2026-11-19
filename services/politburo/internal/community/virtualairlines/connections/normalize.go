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

func NormalizeRecord(entity string, mapping Mapping, schemaByID map[string]FieldMeta, rawFields map[string]any) (MappedRecord, error) {
	out := map[string]any{}
	for logicalKey, fieldID := range mapping.Fields {
		cf, ok := catalogField(entity, logicalKey)
		if !ok {
			continue
		}
		meta, ok := schemaByID[fieldID]
		if !ok {
			return MappedRecord{}, fmt.Errorf("%w: unknown field id %s", ErrSchemaDrift, fieldID)
		}
		cell, present := rawFields[meta.Name]
		if !present || cell == nil {
			continue
		}
		val, err := normalizeCell(cf, cell)
		if err != nil {
			return MappedRecord{}, fmt.Errorf("%w: field %s: %v", ErrSchemaDrift, cf.Key, err)
		}
		if val == nil {
			continue
		}
		out[cf.Key] = val
	}
	return MappedRecord{Fields: out}, nil
}

func normalizeCell(cf CatalogField, cell any) (any, error) {
	switch cf.Kind {
	case KindText:
		return normalizeText(cell)
	case KindNumber:
		return normalizeNumber(cell)
	case KindDuration:
		return normalizeDuration(cell)
	case KindDate, KindInstant:
		return normalizeDate(cell)
	case KindLinks:
		return normalizeLinks(cell)
	default:
		return normalizeText(cell)
	}
}

func normalizeText(cell any) (any, error) {
	switch v := cell.(type) {
	case string:
		if v == "" {
			return nil, nil
		}
		return v, nil
	case map[string]any:
		if name, ok := v["name"].(string); ok && name != "" {
			return name, nil
		}
	}
	return nil, fmt.Errorf("unsupported text cell")
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
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	}
	return nil, fmt.Errorf("unsupported duration cell")
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
		return t.UTC().Format(time.RFC3339), nil
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