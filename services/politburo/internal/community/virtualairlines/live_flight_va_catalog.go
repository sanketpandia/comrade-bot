package virtualairlines

import (
	"encoding/json"
	"sort"
	"strings"

	"infinite-experiment/politburo/internal/livegame/flights"
)

// VirtualAirlineMatch is a VA that matched a live flight callsign on a session.
type VirtualAirlineMatch struct {
	VAID   string
	VACode string
}

// CallsignAffix is one VA's callsign matching rule from BASIC config.
type CallsignAffix struct {
	VAID             string
	VACode           string
	Prefix           string
	Suffix           string
	PreferredServers []string // normalized; empty means server-unscoped
}

// LiveVACatalog holds affix rules and server eligibility for live flight classification.
type LiveVACatalog struct {
	Rules         []CallsignAffix
	Servers       map[string][]string // normalized session name -> va ids
	UnscopedVAIDs []string
}

func normalizeSessionName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// BuildLiveVACatalog assembles a catalog from parsed BASIC config rows.
func BuildLiveVACatalog(rows []basicConfigRow) LiveVACatalog {
	catalog := LiveVACatalog{Servers: map[string][]string{}}
	unscopedSet := map[string]bool{}

	for _, row := range rows {
		if row.Prefix == "" && row.Suffix == "" {
			continue
		}
		rule := CallsignAffix{
			VAID:             row.VAID,
			VACode:           row.VACode,
			Prefix:           row.Prefix,
			Suffix:           row.Suffix,
			PreferredServers: row.PreferredServers,
		}
		catalog.Rules = append(catalog.Rules, rule)

		if len(row.PreferredServers) == 0 {
			unscopedSet[row.VAID] = true
			continue
		}
		for _, server := range row.PreferredServers {
			appendUniqueVAID(catalog.Servers, server, row.VAID)
		}
	}

	for id := range unscopedSet {
		catalog.UnscopedVAIDs = append(catalog.UnscopedVAIDs, id)
	}
	sort.Strings(catalog.UnscopedVAIDs)
	for server := range catalog.Servers {
		sort.Strings(catalog.Servers[server])
	}
	sort.Slice(catalog.Rules, func(i, j int) bool {
		if catalog.Rules[i].VACode != catalog.Rules[j].VACode {
			return catalog.Rules[i].VACode < catalog.Rules[j].VACode
		}
		return catalog.Rules[i].VAID < catalog.Rules[j].VAID
	})
	return catalog
}

func appendUniqueVAID(servers map[string][]string, server, vaID string) {
	list := servers[server]
	for _, existing := range list {
		if existing == vaID {
			return
		}
	}
	servers[server] = append(list, vaID)
}

// Match returns VAs whose affix rules match callsign on the given session.
func (c LiveVACatalog) Match(callsign, sessionNormalizedName string) []VirtualAirlineMatch {
	callsign = strings.TrimSpace(callsign)
	session := normalizeSessionName(sessionNormalizedName)
	candidates := candidateVAIDs(c, session)

	var matches []VirtualAirlineMatch
	for _, rule := range c.Rules {
		if !candidates[rule.VAID] {
			continue
		}
		if !flights.CallsignAffixMatches(callsign, rule.Prefix, rule.Suffix) {
			continue
		}
		matches = append(matches, VirtualAirlineMatch{VAID: rule.VAID, VACode: rule.VACode})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].VACode != matches[j].VACode {
			return matches[i].VACode < matches[j].VACode
		}
		return matches[i].VAID < matches[j].VAID
	})
	return matches
}

// AffixMissReason explains why Match returned no VAs (empty string when rules matched or catalog is empty).
func (c LiveVACatalog) AffixMissReason(callsign, sessionNormalizedName string) string {
	if len(c.Rules) == 0 {
		return "empty_catalog"
	}
	session := normalizeSessionName(sessionNormalizedName)
	candidates := candidateVAIDs(c, session)
	if len(candidates) == 0 {
		return "no_eligible_vas_for_session"
	}
	callsign = strings.TrimSpace(callsign)
	eligibleRules := 0
	for _, rule := range c.Rules {
		if !candidates[rule.VAID] {
			continue
		}
		eligibleRules++
		if flights.CallsignAffixMatches(callsign, rule.Prefix, rule.Suffix) {
			return ""
		}
	}
	if eligibleRules == 0 {
		return "no_eligible_vas_for_session"
	}
	return "affix_mismatch"
}

func candidateVAIDs(c LiveVACatalog, session string) map[string]bool {
	out := map[string]bool{}
	for _, id := range c.Servers[session] {
		out[id] = true
	}
	for _, id := range c.UnscopedVAIDs {
		out[id] = true
	}
	return out
}

type basicConfigRow struct {
	VAID             string
	VACode           string
	Prefix           string
	Suffix           string
	PreferredServers []string
}

func parseBasicConfigRow(vaID, vaCode string, config []byte) (basicConfigRow, error) {
	var top struct {
		GameCallsign struct {
			Prefix string `json:"prefix"`
			Suffix string `json:"suffix"`
		} `json:"game_callsign"`
		PreferredServers []string `json:"preferred_servers"`
	}
	if err := json.Unmarshal(config, &top); err != nil {
		return basicConfigRow{}, err
	}
	servers := normalizePreferredServers(top.PreferredServers)
	return basicConfigRow{
		VAID:             vaID,
		VACode:           vaCode,
		Prefix:           strings.TrimSpace(top.GameCallsign.Prefix),
		Suffix:           strings.TrimSpace(top.GameCallsign.Suffix),
		PreferredServers: servers,
	}, nil
}

func normalizePreferredServers(servers []string) []string {
	if len(servers) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range servers {
		n := normalizeSessionName(s)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
