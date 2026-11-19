package connections

import (
	"encoding/json"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

type airtableFieldOptions struct {
	LinkedTableID string `json:"linkedTableId"`
	Result        *struct {
		Type string `json:"type"`
	} `json:"result"`
	// Lookup / rollup element kinds when present.
	Options *struct {
		Result *struct {
			Type string `json:"type"`
		} `json:"result"`
	} `json:"options"`
}

// effectiveAirtableType returns the type used for catalog compatibility and normalization.
func effectiveAirtableType(f airtable.Field) string {
	t := strings.TrimSpace(f.Type)
	switch t {
	case "formula", "lookup", "rollup", "multipleLookupValues", "count":
		_, result, _ := airtableFieldMeta(f)
		if result != "" {
			return result
		}
	case "lastModifiedTime", "createdTime":
		_, result, _ := airtableFieldMeta(f)
		if result != "" {
			return result
		}
		return "dateTime"
	}
	return t
}

func airtableFieldMeta(f airtable.Field) (linkedTableID, resultType, elementType string) {
	if len(f.Options) == 0 {
		return "", "", ""
	}
	var opts airtableFieldOptions
	if err := json.Unmarshal(f.Options, &opts); err != nil {
		return "", "", ""
	}
	if opts.Result != nil {
		resultType = opts.Result.Type
	}
	if opts.Options != nil && opts.Options.Result != nil && resultType == "" {
		resultType = opts.Options.Result.Type
		elementType = resultType
	}
	return opts.LinkedTableID, resultType, elementType
}