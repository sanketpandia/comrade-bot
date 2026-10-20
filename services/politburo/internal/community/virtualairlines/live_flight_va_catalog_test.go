package virtualairlines

import (
	"testing"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
)

func basicRow(vaID, vaCode, prefix, suffix string, servers ...string) vaconfig.ActiveBasic {
	return vaconfig.ActiveBasic{
		VAID:   vaID,
		VACode: vaCode,
		Basic: vaconfig.Basic{
			GameCallsign:     vaconfig.GameCallsign{Prefix: prefix, Suffix: suffix},
			PreferredServers: servers,
		},
	}
}

func TestBuildLiveVACatalogServerIndexAndUnscoped(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{
		basicRow("va-1", "AAA", "IE", "", "casual", "expert"),
		basicRow("va-2", "BBB", "SW", ""),
	})

	if len(catalog.Servers["expert"]) != 1 || catalog.Servers["expert"][0] != "va-1" {
		t.Fatalf("expert servers = %v", catalog.Servers["expert"])
	}
	if len(catalog.Servers["casual"]) != 1 || catalog.Servers["casual"][0] != "va-1" {
		t.Fatalf("casual servers = %v", catalog.Servers["casual"])
	}
	if len(catalog.UnscopedVAIDs) != 1 || catalog.UnscopedVAIDs[0] != "va-2" {
		t.Fatalf("unscoped = %v", catalog.UnscopedVAIDs)
	}
}

func TestBuildLiveVACatalogSkipsVAWithoutAffix(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "AAA", "", "", "expert")})
	if len(catalog.Rules) != 0 || len(catalog.Servers) != 0 {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestLiveVACatalogMatchServerScoped(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "IE", "IE", "", "expert")})
	matches := catalog.Match("IE123", "IE123", "expert")
	if len(matches) != 1 || matches[0].VAID != "va-1" {
		t.Fatalf("expert match = %#v", matches)
	}
	if len(catalog.Match("IE123", "IE123", "casual")) != 0 {
		t.Fatal("expected no match on casual")
	}
}

func TestLiveVACatalogMatchUnscopedAllSessions(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "IE", "IE", "")})
	for _, session := range []string{"casual", "expert", "training"} {
		if len(catalog.Match("IE123", "IE123", session)) != 1 {
			t.Fatalf("session %s: no match", session)
		}
	}
}

func TestLiveVACatalogMatchIsCaseSensitive(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "IE", "ie", "")})
	if len(catalog.Match("IE123", "IE123", "expert")) != 0 {
		t.Fatal("expected case-sensitive prefix miss")
	}
}

func TestLiveVACatalogMatchLiteralTrailerSuffix(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "SW", "SW", "Heavy", "expert")})
	if len(catalog.Match("SW39 Heavy", "SW39", "expert")) != 1 {
		t.Fatal("expected prefix+suffix match on the upstream callsign")
	}
	if len(catalog.Match("SW39", "SW39", "expert")) != 0 {
		t.Fatal("expected suffix miss")
	}
}

func TestLiveVACatalogMatchSuffixOnNormalizedCallsign(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{basicRow("va-1", "DAL", "", "AK", "expert")})
	matches := catalog.Match("Delta 095AK Heavy", "Delta 095AK", "expert")
	if len(matches) != 1 || matches[0].VAID != "va-1" {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestLiveVACatalogMatchMultipleVAs(t *testing.T) {
	catalog := BuildLiveVACatalog([]vaconfig.ActiveBasic{
		basicRow("va-1", "AA", "DAL", "", "expert"),
		basicRow("va-2", "BB", "", "Heavy", "expert"),
	})
	matches := catalog.Match("DAL123 Heavy", "DAL123", "expert")
	if len(matches) != 2 || matches[0].VACode != "AA" || matches[1].VACode != "BB" {
		t.Fatalf("matches = %#v", matches)
	}
}
