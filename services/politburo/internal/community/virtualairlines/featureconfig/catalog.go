package featureconfig

import "infinite-experiment/politburo/internal/community/virtualairlines/connections"

// CatalogFieldForFeature re-exports catalog entries for API responses.
type CatalogFieldForFeature struct {
	Key           string                      `json:"key"`
	Label         string                      `json:"label"`
	Kind          connections.FieldKind       `json:"kind"`
	Usage         connections.FieldUsage      `json:"usage,omitempty"`
	Cardinality   connections.FieldCardinality `json:"cardinality,omitempty"`
	Aliases       []string                    `json:"aliases,omitempty"`
	AcceptedTypes []string                    `json:"acceptedTypes"`
	Derived       bool                        `json:"derived,omitempty"`
}

func StatsCatalogFields() []CatalogFieldForFeature {
	return append(entityFields(connections.EntityPilots), append(entityFields(connections.EntityCareerPilots), derivedStatsFields()...)...)
}

func derivedStatsFields() []CatalogFieldForFeature {
	return []CatalogFieldForFeature{{
		Key: "lastFlownRoute", Label: "Last flown route",
		Kind: connections.KindText, Usage: connections.UsageRead, Cardinality: connections.CardinalityScalar,
		Derived: true,
	}}
}

func PIREPCatalogFields() []CatalogFieldForFeature {
	return entityFields(connections.EntityPIREPs)
}

func entityFields(entity string) []CatalogFieldForFeature {
	out := make([]CatalogFieldForFeature, 0)
	for _, e := range connections.CatalogEntities() {
		if e.Entity != entity {
			continue
		}
		for _, f := range e.Fields {
			out = append(out, CatalogFieldForFeature{
				Key: f.Key, Label: f.Label, Kind: f.Kind, Usage: f.Usage,
				Cardinality: f.Cardinality, Aliases: f.Aliases, AcceptedTypes: f.AcceptedTypes,
			})
		}
		break
	}
	return out
}