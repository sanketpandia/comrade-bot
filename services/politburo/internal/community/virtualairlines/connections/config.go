package connections

import (
	"encoding/json"
	"fmt"

	"infinite-experiment/politburo/internal/secretstore"
)

const (
	documentVersion  = 1
	ProviderAirtable = "airtable"
)

// Document is the persisted DATA_CONNECTIONS block.
type Document struct {
	Version     int                `json:"version"`
	Revision    int                `json:"revision"`
	Connections []Connection       `json:"connections"`
	Mappings    map[string]Mapping `json:"mappings"`
}

type Connection struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	Airtable *AirtableSetup `json:"airtable,omitempty"`
}

type AirtableSetup struct {
	Credential secretstore.Envelope `json:"credential"`
	Bases      []BaseSetup          `json:"bases"`
}

type BaseSetup struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	BaseID string `json:"baseId"`
}

// DurationUnit is the stored unit for plain numeric duration columns.
type DurationUnit string

const (
	UnitSeconds DurationUnit = "seconds"
	UnitMinutes DurationUnit = "minutes"
	UnitHours   DurationUnit = "hours"
)

// Mapping binds one logical entity to a provider table.
type Mapping struct {
	ConnectionID string                       `json:"connectionId"`
	BaseSetupID  string                       `json:"baseSetupId"`
	TableID      string                       `json:"tableId"`
	Fields       map[string]string            `json:"fields"`
	FieldUnits   map[string]DurationUnit      `json:"fieldUnits,omitempty"`
}

// EmptyDocument returns revision-zero state with no connections.
func EmptyDocument() Document {
	return Document{
		Version:     documentVersion,
		Revision:    0,
		Connections: []Connection{},
		Mappings:    map[string]Mapping{},
	}
}

func ParseDocument(raw json.RawMessage) (Document, error) {
	if len(raw) == 0 {
		return EmptyDocument(), nil
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Document{}, fmt.Errorf("%w: decode document: %v", ErrCorruptConfig, err)
	}
	if doc.Version != documentVersion {
		return Document{}, fmt.Errorf("%w: unsupported document version %d", ErrInvalidMapping, doc.Version)
	}
	if doc.Connections == nil {
		doc.Connections = []Connection{}
	}
	if doc.Mappings == nil {
		doc.Mappings = map[string]Mapping{}
	}
	return doc, nil
}

func (d Document) Marshal() (json.RawMessage, error) {
	out, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

// AdminView is the redacted API projection.
type AdminView struct {
	Revision    int                `json:"revision"`
	Connections []ConnectionView   `json:"connections"`
	Mappings    map[string]Mapping `json:"mappings"`
	Catalog     []CatalogEntity    `json:"catalog"`
}

type ConnectionView struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Provider      string      `json:"provider"`
	HasCredential bool        `json:"hasCredential"`
	Bases         []BaseSetup `json:"bases"`
}

func Redact(doc Document) AdminView {
	views := make([]ConnectionView, 0, len(doc.Connections))
	for _, c := range doc.Connections {
		hasCred := false
		var bases []BaseSetup
		if c.Airtable != nil {
			hasCred = len(c.Airtable.Credential.Sealed) > 0
			bases = append(bases, c.Airtable.Bases...)
		}
		views = append(views, ConnectionView{
			ID:            c.ID,
			Name:          c.Name,
			Provider:      c.Provider,
			HasCredential: hasCred,
			Bases:         bases,
		})
	}
	mappings := doc.Mappings
	if mappings == nil {
		mappings = map[string]Mapping{}
	}
	return AdminView{
		Revision:    doc.Revision,
		Connections: views,
		Mappings:    mappings,
		Catalog:     CatalogEntities(),
	}
}
