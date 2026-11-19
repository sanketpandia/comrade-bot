package connections

// Entity names for mappings and reads.
const (
	EntityPilots       = "pilots"
	EntityCareerPilots = "careerPilots"
	EntityRoutes       = "routes"
	EntityPIREPs       = "pireps"
)

// FieldKind describes normalized output types.
type FieldKind string

const (
	KindText     FieldKind = "text"
	KindNumber   FieldKind = "number"
	KindDuration FieldKind = "duration"
	KindDate     FieldKind = "date"
	KindLinks    FieldKind = "links"
	KindInstant  FieldKind = "instant"
)

// FieldUsage is read, write, or both for filing vs display.
type FieldUsage string

const (
	UsageRead  FieldUsage = "read"
	UsageWrite FieldUsage = "write"
	UsageBoth  FieldUsage = "both"
)

// FieldCardinality is scalar or list after normalization.
type FieldCardinality string

const (
	CardinalityScalar FieldCardinality = "scalar"
	CardinalityList   FieldCardinality = "list"
)

// CatalogField defines a logical mapping target.
type CatalogField struct {
	Key           string           `json:"key"`
	Label         string           `json:"label"`
	Kind          FieldKind        `json:"kind"`
	Usage         FieldUsage       `json:"usage,omitempty"`
	Cardinality   FieldCardinality `json:"cardinality,omitempty"`
	Aliases       []string         `json:"aliases,omitempty"`
	AcceptedTypes []string         `json:"acceptedTypes"`
}

// CatalogEntity groups fields for one logical entity.
type CatalogEntity struct {
	Entity      string         `json:"entity"`
	Label       string         `json:"label"`
	Fields      []CatalogField `json:"fields"`
	MinimumRule string         `json:"minimumRule"`
}

var catalog = map[string][]CatalogField{
	EntityPilots: {
		{Key: "callsign", Label: "Callsign", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, Aliases: []string{"identifier", "pilot id"}, AcceptedTypes: []string{"singleLineText", "multilineText", "email", "url", "phoneNumber"}},
		{Key: "ifcUsername", Label: "IFC username", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, Aliases: []string{"ifc", "username"}, AcceptedTypes: []string{"singleLineText", "multilineText", "email", "url"}},
		{Key: "displayName", Label: "Display name", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"name", "pilot name"}, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "flightTimeSeconds", Label: "Flight time", Kind: KindDuration, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"block time", "flight time", "hours"}, AcceptedTypes: []string{"number", "duration"}},
		{Key: "category", Label: "Category", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"fleet", "type"}, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleSelects"}},
		{Key: "region", Label: "Region", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"division", "base"}, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleSelects"}},
		{Key: "rank", Label: "Rank", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"grade", "title"}, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleSelects"}},
		{Key: "status", Label: "Status", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"active", "membership"}, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
	},
	EntityCareerPilots: {
		{Key: "callsign", Label: "Callsign", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "ifcUsername", Label: "IFC username", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText", "email", "url"}},
		{Key: "flightTimeSeconds", Label: "Total flight time", Kind: KindDuration, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"total hours", "cm hours"}, AcceptedTypes: []string{"number", "duration"}},
		{Key: "timeToNextAircraftSeconds", Label: "Time to next aircraft", Kind: KindDuration, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"time remaining", "hours to next"}, AcceptedTypes: []string{"number", "duration"}},
		{Key: "lastActivity", Label: "Last activity", Kind: KindDate, Usage: UsageRead, Cardinality: CardinalityScalar, Aliases: []string{"last flight", "last active"}, AcceptedTypes: []string{"date", "dateTime"}},
		{Key: "assignedRoutes", Label: "Assigned routes", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityList, Aliases: []string{"rotation", "routes"}, AcceptedTypes: []string{"singleLineText", "multilineText", "multipleSelects", "multipleRecordLinks"}},
		{Key: "aircraft", Label: "Aircraft", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleRecordLinks"}},
		{Key: "airline", Label: "Airline", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "lastLogRefs", Label: "Latest report", Kind: KindLinks, Usage: UsageRead, Cardinality: CardinalityList, Aliases: []string{"latest report", "last pirep"}, AcceptedTypes: []string{"multipleRecordLinks"}},
	},
	EntityRoutes: {
		{Key: "origin", Label: "Origin", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "destination", Label: "Destination", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "durationSeconds", Label: "Duration (seconds)", Kind: KindDuration, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"number", "duration"}},
		{Key: "aircraft", Label: "Aircraft", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleSelects"}},
		{Key: "routeLabel", Label: "Route label", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText", "formula"}},
	},
	EntityPIREPs: {
		{Key: "pilotRefs", Label: "Pilot links", Kind: KindLinks, Usage: UsageBoth, Cardinality: CardinalityList, AcceptedTypes: []string{"multipleRecordLinks"}},
		{Key: "routeRefs", Label: "Route links", Kind: KindLinks, Usage: UsageBoth, Cardinality: CardinalityList, AcceptedTypes: []string{"multipleRecordLinks"}},
		{Key: "ifcUsername", Label: "IFC username", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText"}},
		{Key: "completedOn", Label: "Completed on", Kind: KindDate, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"date", "dateTime"}},
		{Key: "flightTimeSeconds", Label: "Flight time", Kind: KindDuration, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"number", "duration"}},
		{Key: "aircraft", Label: "Aircraft", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "airline", Label: "Airline", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "flightMode", Label: "Flight mode", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "remarks", Label: "Remarks", Kind: KindText, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText", "richText"}},
		{Key: "fuelKg", Label: "Fuel (kg)", Kind: KindNumber, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"number"}},
		{Key: "cargoKg", Label: "Cargo (kg)", Kind: KindNumber, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"number"}},
		{Key: "passengerCount", Label: "Passengers", Kind: KindNumber, Usage: UsageBoth, Cardinality: CardinalityScalar, AcceptedTypes: []string{"number"}},
		{Key: "routeLabel", Label: "Route label", Kind: KindText, Usage: UsageRead, Cardinality: CardinalityScalar, AcceptedTypes: []string{"singleLineText", "multilineText", "formula"}},
	},
}

var catalogMinimumRules = map[string]string{
	EntityPilots:       "callsign or ifcUsername",
	EntityCareerPilots: "callsign or ifcUsername",
	EntityRoutes:       "origin and destination",
	EntityPIREPs:       "at least one supported field",
}

// DerivedStatsKeys are feature-only keys resolved at preview time.
var DerivedStatsKeys = map[string]struct{}{
	"lastFlownRoute": {},
}

// WritableComputedAirtableTypes cannot receive filing values.
var WritableComputedAirtableTypes = map[string]struct{}{
	"formula": {}, "rollup": {}, "count": {}, "lookup": {},
	"createdTime": {}, "lastModifiedTime": {}, "autoNumber": {},
}

func CatalogEntities() []CatalogEntity {
	order := []string{EntityPilots, EntityCareerPilots, EntityRoutes, EntityPIREPs}
	out := make([]CatalogEntity, 0, len(order))
	for _, entity := range order {
		out = append(out, CatalogEntity{
			Entity:      entity,
			Label:       catalogLabel(entity),
			Fields:      catalog[entity],
			MinimumRule: catalogMinimumRules[entity],
		})
	}
	return out
}

func catalogLabel(entity string) string {
	switch entity {
	case EntityPilots:
		return "Pilots"
	case EntityCareerPilots:
		return "Career pilots"
	case EntityRoutes:
		return "Routes"
	case EntityPIREPs:
		return "PIREPs"
	default:
		return entity
	}
}

func catalogField(entity, key string) (CatalogField, bool) {
	for _, f := range catalog[entity] {
		if f.Key == key {
			return f, true
		}
	}
	return CatalogField{}, false
}

// CatalogFieldByKey resolves a catalog entry for validation and APIs.
func CatalogFieldByKey(entity, key string) (CatalogField, bool) {
	return catalogField(entity, key)
}

func MappingMeetsMinimum(entity string, fields map[string]string) bool {
	switch entity {
	case EntityPilots, EntityCareerPilots:
		_, cs := fields["callsign"]
		_, ifc := fields["ifcUsername"]
		return cs || ifc
	case EntityRoutes:
		_, o := fields["origin"]
		_, d := fields["destination"]
		return o && d
	case EntityPIREPs:
		return len(fields) > 0
	default:
		return false
	}
}