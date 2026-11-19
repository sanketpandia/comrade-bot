package pirepvaluemappings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

const DocumentVersion = 1

const maxDocumentBytes = 1 << 20

// Document is the canonical v1 interchange format (export/import).
type Document struct {
	Version          int               `json:"version"`
	Defaults         Defaults          `json:"defaults"`
	AircraftMappings map[string]string `json:"AircraftMappings"`
	AirlineMappings  map[string]string `json:"AirlineMappings"`
}

type Defaults struct {
	Aircraft *string `json:"aircraft"`
	Airline  *string `json:"airline"`
}

func EmptyDocument() Document {
	return Document{
		Version:          DocumentVersion,
		AircraftMappings: map[string]string{},
		AirlineMappings:  map[string]string{},
	}
}

func (d Document) Normalize() Document {
	out := d
	if out.Version == 0 {
		out.Version = DocumentVersion
	}
	if out.AircraftMappings == nil {
		out.AircraftMappings = map[string]string{}
	}
	if out.AirlineMappings == nil {
		out.AirlineMappings = map[string]string{}
	}
	return out
}

func DocumentFromRow(row Row) Document {
	doc := EmptyDocument()
	if len(row.AircraftMappings) > 0 {
		doc.AircraftMappings = copyMap(row.AircraftMappings)
	}
	if len(row.AirlineMappings) > 0 {
		doc.AirlineMappings = copyMap(row.AirlineMappings)
	}
	doc.Defaults.Aircraft = row.DefaultAircraft
	doc.Defaults.Airline = row.DefaultAirline
	return doc
}

func RowFromDocument(doc Document) (Row, error) {
	doc = doc.Normalize()
	if err := ValidateDocument(doc); err != nil {
		return Row{}, err
	}
	return Row{
		AircraftMappings: copyMap(doc.AircraftMappings),
		AirlineMappings:  copyMap(doc.AirlineMappings),
		DefaultAircraft:  doc.Defaults.Aircraft,
		DefaultAirline:   doc.Defaults.Airline,
	}, nil
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ParseDocument(raw []byte) (Document, error) {
	if len(raw) > maxDocumentBytes {
		return Document{}, ErrPayloadLarge
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return Document{}, err
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Document{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	doc = doc.Normalize()
	if err := ValidateDocument(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// MarshalExport returns stable UTF-8 JSON with sorted keys.
func MarshalExport(doc Document) ([]byte, error) {
	doc = doc.Normalize()
	if err := ValidateDocument(doc); err != nil {
		return nil, err
	}
	ordered := exportShape{
		Version: DocumentVersion,
		Defaults: exportDefaults{
			Aircraft: doc.Defaults.Aircraft,
			Airline:  doc.Defaults.Airline,
		},
		AircraftMappings: sortedMap(doc.AircraftMappings),
		AirlineMappings:  sortedMap(doc.AirlineMappings),
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ordered); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type exportShape struct {
	Version          int               `json:"version"`
	Defaults         exportDefaults    `json:"defaults"`
	AircraftMappings map[string]string `json:"AircraftMappings"`
	AirlineMappings  map[string]string `json:"AirlineMappings"`
}

type exportDefaults struct {
	Aircraft *string `json:"aircraft"`
	Airline  *string `json:"airline"`
}

func sortedMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

func DocumentsEqual(a, b Document) bool {
	a = a.Normalize()
	b = b.Normalize()
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}