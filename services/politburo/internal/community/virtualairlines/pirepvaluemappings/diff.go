package pirepvaluemappings

import "sort"

type DiffEntry struct {
	Key      string `json:"key"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
}

type Diff struct {
	Defaults           DefaultsDiff `json:"defaults"`
	AircraftAdditions  []DiffEntry  `json:"aircraftAdditions"`
	AircraftChanges    []DiffEntry  `json:"aircraftChanges"`
	AircraftRemovals   []DiffEntry  `json:"aircraftRemovals"`
	AirlineAdditions   []DiffEntry  `json:"airlineAdditions"`
	AirlineChanges     []DiffEntry  `json:"airlineChanges"`
	AirlineRemovals    []DiffEntry  `json:"airlineRemovals"`
	UnmatchedAliases   []string     `json:"unmatchedAliases,omitempty"`
}

type DefaultsDiff struct {
	Aircraft *DiffEntry `json:"aircraft,omitempty"`
	Airline  *DiffEntry `json:"airline,omitempty"`
}

func DiffDocuments(current, proposed Document, catalogNames CatalogNames) Diff {
	current = current.Normalize()
	proposed = proposed.Normalize()
	out := Diff{}
	out.Defaults = diffDefaults(current.Defaults, proposed.Defaults)
	out.AircraftAdditions, out.AircraftChanges, out.AircraftRemovals = diffRules(current.AircraftMappings, proposed.AircraftMappings)
	out.AirlineAdditions, out.AirlineChanges, out.AirlineRemovals = diffRules(current.AirlineMappings, proposed.AirlineMappings)
	out.UnmatchedAliases = unmatchedRuleKeys(proposed, catalogNames)
	return out
}

func diffDefaults(old, new Defaults) DefaultsDiff {
	var d DefaultsDiff
	if ptrString(old.Aircraft) != ptrString(new.Aircraft) {
		d.Aircraft = &DiffEntry{
			OldValue: ptrString(old.Aircraft),
			NewValue: ptrString(new.Aircraft),
		}
	}
	if ptrString(old.Airline) != ptrString(new.Airline) {
		d.Airline = &DiffEntry{
			OldValue: ptrString(old.Airline),
			NewValue: ptrString(new.Airline),
		}
	}
	return d
}

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func diffRules(oldMap, newMap map[string]string) (additions, changes, removals []DiffEntry) {
	for k, v := range newMap {
		prev, ok := oldMap[k]
		if !ok {
			additions = append(additions, DiffEntry{Key: k, NewValue: v})
			continue
		}
		if prev != v {
			changes = append(changes, DiffEntry{Key: k, OldValue: prev, NewValue: v})
		}
	}
	for k, v := range oldMap {
		if _, ok := newMap[k]; !ok {
			removals = append(removals, DiffEntry{Key: k, OldValue: v})
		}
	}
	sortDiffEntries(additions)
	sortDiffEntries(changes)
	sortDiffEntries(removals)
	return additions, changes, removals
}

func sortDiffEntries(in []DiffEntry) {
	sort.Slice(in, func(i, j int) bool { return in[i].Key < in[j].Key })
}

type CatalogNames struct {
	Aircraft map[string]struct{}
	Airline  map[string]struct{}
}

func unmatchedRuleKeys(doc Document, catalog CatalogNames) []string {
	var out []string
	for k := range doc.AircraftMappings {
		if _, ok := catalog.Aircraft[k]; !ok {
			out = append(out, "aircraft:"+k)
		}
	}
	for k := range doc.AirlineMappings {
		if _, ok := catalog.Airline[k]; !ok {
			out = append(out, "airline:"+k)
		}
	}
	return out
}