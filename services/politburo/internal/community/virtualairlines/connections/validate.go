package connections

import (
	"fmt"
	"strings"
)

func ValidateDocument(doc Document) error {
	ids := map[string]struct{}{}
	baseIDs := map[string]string{} // base setup id -> connection id
	for i, conn := range doc.Connections {
		if conn.ID == "" {
			return fmt.Errorf("%w: connection %d missing id", ErrInvalidMapping, i)
		}
		if _, dup := ids[conn.ID]; dup {
			return fmt.Errorf("%w: duplicate connection id", ErrInvalidMapping)
		}
		ids[conn.ID] = struct{}{}
		if strings.TrimSpace(conn.Name) == "" {
			return fmt.Errorf("%w: connection name required", ErrInvalidMapping)
		}
		if conn.Provider != ProviderAirtable {
			return fmt.Errorf("%w: unsupported provider %q", ErrInvalidMapping, conn.Provider)
		}
		if conn.Airtable == nil {
			return fmt.Errorf("%w: airtable setup required", ErrInvalidMapping)
		}
		baseSeen := map[string]struct{}{}
		for _, b := range conn.Airtable.Bases {
			if b.ID == "" || b.BaseID == "" {
				return fmt.Errorf("%w: base id fields required", ErrInvalidMapping)
			}
			if _, dup := baseSeen[b.ID]; dup {
				return fmt.Errorf("%w: duplicate base setup id", ErrInvalidMapping)
			}
			baseSeen[b.ID] = struct{}{}
			baseIDs[b.ID] = conn.ID
		}
	}
	for entity, mapping := range doc.Mappings {
		if entity != EntityPilots && entity != EntityCareerPilots && entity != EntityRoutes && entity != EntityPIREPs {
			return fmt.Errorf("%w: unknown entity %q", ErrInvalidMapping, entity)
		}
		if mapping.ConnectionID == "" || mapping.BaseSetupID == "" || mapping.TableID == "" {
			return fmt.Errorf("%w: mapping %s incomplete", ErrInvalidMapping, entity)
		}
		if connID, ok := baseIDs[mapping.BaseSetupID]; !ok || connID != mapping.ConnectionID {
			return fmt.Errorf("%w: mapping %s references unknown connection or base", ErrInvalidMapping, entity)
		}
		if !MappingMeetsMinimum(entity, mapping.Fields) {
			return fmt.Errorf("%w: mapping %s below minimum fields", ErrInvalidMapping, entity)
		}
		for key, fieldID := range mapping.Fields {
			if _, ok := catalogField(entity, key); !ok {
				return fmt.Errorf("%w: unknown logical field %s", ErrInvalidMapping, key)
			}
			if strings.TrimSpace(fieldID) == "" {
				return fmt.Errorf("%w: empty field id for %s", ErrInvalidMapping, key)
			}
		}
	}
	return nil
}

func connectionByID(doc Document, id string) (Connection, int, bool) {
	for i, c := range doc.Connections {
		if c.ID == id {
			return c, i, true
		}
	}
	return Connection{}, 0, false
}

func baseByID(conn Connection, baseSetupID string) (BaseSetup, bool) {
	if conn.Airtable == nil {
		return BaseSetup{}, false
	}
	for _, b := range conn.Airtable.Bases {
		if b.ID == baseSetupID {
			return b, true
		}
	}
	return BaseSetup{}, false
}

func mappingReferencesConnection(doc Document, connectionID string) bool {
	for _, m := range doc.Mappings {
		if m.ConnectionID == connectionID {
			return true
		}
	}
	return false
}

func mappingReferencesBase(doc Document, baseSetupID string) bool {
	for _, m := range doc.Mappings {
		if m.BaseSetupID == baseSetupID {
			return true
		}
	}
	return false
}