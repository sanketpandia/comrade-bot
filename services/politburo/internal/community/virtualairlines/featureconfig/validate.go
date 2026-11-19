package featureconfig

import (
	"fmt"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

func ValidateStatsDraft(doc StatsDocument, mappings map[string]connections.Mapping) error {
	if doc.Pilot == nil && doc.Career == nil {
		return fmt.Errorf("%w: at least one section required", ErrInvalidDraft)
	}
	if doc.Pilot != nil {
		if err := validateSection(doc.Pilot, connections.EntityPilots, mappings); err != nil {
			return err
		}
	}
	if doc.Career != nil {
		if err := validateSection(doc.Career, connections.EntityCareerPilots, mappings); err != nil {
			return err
		}
		for _, key := range doc.Career.Fields {
			if key == "lastFlownRoute" {
				if err := validateLastFlownRoute(mappings); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateSection(sec *StatsSection, entity string, mappings map[string]connections.Mapping) error {
	if sec == nil {
		return nil
	}
	match := strings.TrimSpace(sec.MatchBy)
	if match != "callsign" && match != "ifcUsername" {
		return fmt.Errorf("%w: invalid matchBy %q", ErrInvalidDraft, sec.MatchBy)
	}
	mapping, ok := mappings[entity]
	if !ok || mapping.TableID == "" {
		return fmt.Errorf("%w: missing mapping for %s", ErrInvalidDraft, entity)
	}
	if _, bound := mapping.Fields[match]; !bound {
		return fmt.Errorf("%w: matchBy %s not bound on %s", ErrInvalidDraft, match, entity)
	}
	for _, key := range sec.Fields {
		if _, derived := connections.DerivedStatsKeys[key]; derived {
			continue
		}
		if _, bound := mapping.Fields[key]; !bound {
			return fmt.Errorf("%w: field %s not bound on %s", ErrInvalidDraft, key, entity)
		}
	}
	return nil
}

func validateLastFlownRoute(mappings map[string]connections.Mapping) error {
	career, ok := mappings[connections.EntityCareerPilots]
	if !ok || career.Fields["lastLogRefs"] == "" {
		return fmt.Errorf("%w: lastFlownRoute requires careerPilots.lastLogRefs", ErrInvalidDraft)
	}
	pirep, ok := mappings[connections.EntityPIREPs]
	if !ok {
		return fmt.Errorf("%w: lastFlownRoute requires pireps mapping", ErrInvalidDraft)
	}
	if pirep.Fields["routeLabel"] == "" && pirep.Fields["routeRefs"] == "" {
		return fmt.Errorf("%w: lastFlownRoute requires pireps.routeLabel or routeRefs", ErrInvalidDraft)
	}
	if pirep.Fields["routeRefs"] != "" {
		if routes, ok := mappings[connections.EntityRoutes]; !ok || routes.TableID == "" {
			return fmt.Errorf("%w: lastFlownRoute route link requires routes mapping", ErrInvalidDraft)
		}
	}
	return nil
}

func ValidatePIREPDraft(doc PIREPDocument, mappings map[string]connections.Mapping) error {
	if len(doc.Fields) == 0 {
		return fmt.Errorf("%w: at least one field required", ErrInvalidDraft)
	}
	pirep, ok := mappings[connections.EntityPIREPs]
	if !ok || pirep.TableID == "" {
		return fmt.Errorf("%w: pireps mapping required", ErrInvalidDraft)
	}
	hasPilot := false
	other := 0
	for _, key := range doc.Fields {
		if key == "pilotRefs" || key == "ifcUsername" {
			hasPilot = true
		}
		if key != "pilotRefs" && key != "ifcUsername" && key != "routeLabel" {
			other++
		}
		cf, ok := connections.CatalogFieldByKey(connections.EntityPIREPs, key)
		if !ok {
			return fmt.Errorf("%w: unknown pirep field %s", ErrInvalidDraft, key)
		}
		if cf.Usage == connections.UsageRead {
			continue
		}
		if _, bound := pirep.Fields[key]; !bound {
			return fmt.Errorf("%w: field %s not bound", ErrInvalidDraft, key)
		}
	}
	if !hasPilot {
		return fmt.Errorf("%w: pilotRefs or ifcUsername required", ErrInvalidDraft)
	}
	if other == 0 {
		return fmt.Errorf("%w: at least one flight detail field required", ErrInvalidDraft)
	}
	return nil
}