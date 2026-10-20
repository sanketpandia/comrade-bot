package virtualairlines

import (
	"sort"
	"strings"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
)

// VirtualAirlineMatch is a VA whose callsign rule matched a live flight on a session.
type VirtualAirlineMatch struct {
	VAID   string `json:"vaId"`
	VACode string `json:"vaCode"`
}

// CallsignAffix is one VA's callsign matching rule from BASIC config.
type CallsignAffix struct {
	VAID   string
	VACode string
	Prefix string
	Suffix string
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

// BuildLiveVACatalog assembles a catalog from active VAs' BASIC config. VAs without
// a prefix or suffix are skipped; VAs without preferred servers match on every session.
func BuildLiveVACatalog(rows []vaconfig.ActiveBasic) LiveVACatalog {
	catalog := LiveVACatalog{Servers: map[string][]string{}}
	unscopedSet := map[string]bool{}

	for _, row := range rows {
		callsign := row.Basic.GameCallsign
		if callsign.Prefix == "" && callsign.Suffix == "" {
			continue
		}
		catalog.Rules = append(catalog.Rules, CallsignAffix{
			VAID:   row.VAID,
			VACode: row.VACode,
			Prefix: callsign.Prefix,
			Suffix: callsign.Suffix,
		})

		if len(row.Basic.PreferredServers) == 0 {
			unscopedSet[row.VAID] = true
			continue
		}
		for _, server := range row.Basic.PreferredServers {
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

// Match returns VAs whose affix rules match a flight on the given session. callsign is the
// upstream callsign; normalizedCallsign has game trailers (for example " Heavy") removed so
// suffix rules can match either form.
func (c LiveVACatalog) Match(callsign, normalizedCallsign, sessionNormalizedName string) []VirtualAirlineMatch {
	callsign = strings.TrimSpace(callsign)
	normalizedCallsign = strings.TrimSpace(normalizedCallsign)
	candidates := candidateVAIDs(c, normalizeSessionName(sessionNormalizedName))

	var matches []VirtualAirlineMatch
	for _, rule := range c.Rules {
		if !candidates[rule.VAID] || !rule.matches(callsign, normalizedCallsign) {
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

func (r CallsignAffix) matches(callsign, normalizedCallsign string) bool {
	if r.Prefix == "" && r.Suffix == "" {
		return false
	}
	if r.Prefix != "" && !strings.HasPrefix(callsign, r.Prefix) {
		return false
	}
	if r.Suffix != "" && !strings.HasSuffix(callsign, r.Suffix) && !strings.HasSuffix(normalizedCallsign, r.Suffix) {
		return false
	}
	return true
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
