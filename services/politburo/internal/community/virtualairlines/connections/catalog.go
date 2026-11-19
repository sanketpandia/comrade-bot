package connections

// Entity names for mappings and reads.
const (
	EntityPilots = "pilots"
	EntityRoutes = "routes"
	EntityPIREPs = "pireps"
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

// CatalogField defines a logical mapping target.
type CatalogField struct {
	Key           string    `json:"key"`
	Label         string    `json:"label"`
	Kind          FieldKind `json:"kind"`
	AcceptedTypes []string  `json:"acceptedTypes"`
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
		{Key: "callsign", Label: "Callsign", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText", "email", "url", "phoneNumber"}},
		{Key: "ifcUsername", Label: "IFC username", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText", "email", "url"}},
		{Key: "displayName", Label: "Display name", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText"}},
	},
	EntityRoutes: {
		{Key: "origin", Label: "Origin", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "destination", Label: "Destination", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText"}},
		{Key: "durationSeconds", Label: "Duration (seconds)", Kind: KindDuration, AcceptedTypes: []string{"number", "duration"}},
		{Key: "aircraft", Label: "Aircraft", Kind: KindText, AcceptedTypes: []string{"singleLineText", "singleSelect", "multipleSelects"}},
	},
	EntityPIREPs: {
		{Key: "pilotRefs", Label: "Pilot links", Kind: KindLinks, AcceptedTypes: []string{"multipleRecordLinks", "singleRecordLink"}},
		{Key: "routeRefs", Label: "Route links", Kind: KindLinks, AcceptedTypes: []string{"multipleRecordLinks", "singleRecordLink"}},
		{Key: "ifcUsername", Label: "IFC username", Kind: KindText, AcceptedTypes: []string{"singleLineText"}},
		{Key: "completedOn", Label: "Completed on", Kind: KindDate, AcceptedTypes: []string{"date", "dateTime"}},
		{Key: "flightTimeSeconds", Label: "Flight time (seconds)", Kind: KindDuration, AcceptedTypes: []string{"number", "duration"}},
		{Key: "aircraft", Label: "Aircraft", Kind: KindText, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "airline", Label: "Airline", Kind: KindText, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "flightMode", Label: "Flight mode", Kind: KindText, AcceptedTypes: []string{"singleLineText", "singleSelect"}},
		{Key: "remarks", Label: "Remarks", Kind: KindText, AcceptedTypes: []string{"singleLineText", "multilineText", "richText"}},
		{Key: "fuelKg", Label: "Fuel (kg)", Kind: KindNumber, AcceptedTypes: []string{"number"}},
		{Key: "cargoKg", Label: "Cargo (kg)", Kind: KindNumber, AcceptedTypes: []string{"number"}},
		{Key: "passengerCount", Label: "Passengers", Kind: KindNumber, AcceptedTypes: []string{"number"}},
	},
}

var catalogMinimumRules = map[string]string{
	EntityPilots: "callsign or ifcUsername",
	EntityRoutes: "origin and destination",
	EntityPIREPs: "at least one supported field",
}

func CatalogEntities() []CatalogEntity {
	out := []CatalogEntity{
		{Entity: EntityPilots, Label: "Pilots", Fields: catalog[EntityPilots], MinimumRule: catalogMinimumRules[EntityPilots]},
		{Entity: EntityRoutes, Label: "Routes", Fields: catalog[EntityRoutes], MinimumRule: catalogMinimumRules[EntityRoutes]},
		{Entity: EntityPIREPs, Label: "PIREPs", Fields: catalog[EntityPIREPs], MinimumRule: catalogMinimumRules[EntityPIREPs]},
	}
	return out
}

func catalogField(entity, key string) (CatalogField, bool) {
	for _, f := range catalog[entity] {
		if f.Key == key {
			return f, true
		}
	}
	return CatalogField{}, false
}

func MappingMeetsMinimum(entity string, fields map[string]string) bool {
	switch entity {
	case EntityPilots:
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