package virtualairlines

import "testing"

func TestBuildLiveVACatalogServerIndexAndUnscoped(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "AAA", Prefix: "IE", PreferredServers: []string{"expert", "casual", "expert"}},
		{VAID: "va-2", VACode: "BBB", Prefix: "SW", PreferredServers: nil},
	})

	if catalog.Servers["expert"] == nil || catalog.Servers["expert"][0] != "va-1" || len(catalog.Servers["expert"]) != 1 {
		t.Fatalf("expert servers = %v", catalog.Servers["expert"])
	}
	if catalog.Servers["casual"] == nil || catalog.Servers["casual"][0] != "va-1" {
		t.Fatalf("casual servers = %v", catalog.Servers["casual"])
	}
	if len(catalog.UnscopedVAIDs) != 1 || catalog.UnscopedVAIDs[0] != "va-2" {
		t.Fatalf("unscoped = %v", catalog.UnscopedVAIDs)
	}
}

func TestLiveVACatalogMatchServerScoped(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "IE", Prefix: "IE", PreferredServers: []string{"expert"}},
	})
	matches := catalog.Match("IE123", "expert")
	if len(matches) != 1 || matches[0].VAID != "va-1" {
		t.Fatalf("expert match = %#v", matches)
	}
	if len(catalog.Match("IE123", "casual")) != 0 {
		t.Fatal("expected no match on casual")
	}
}

func TestLiveVACatalogMatchUnscopedAllSessions(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "IE", Prefix: "IE", PreferredServers: nil},
	})
	for _, session := range []string{"casual", "expert", "training"} {
		if len(catalog.Match("IE123", session)) != 1 {
			t.Fatalf("session %s: %#v", session, catalog.Match("IE123", session))
		}
	}
}

func TestLiveVACatalogMatchPrefixAndSuffix(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "SW", Prefix: "SW", Suffix: "Heavy", PreferredServers: []string{"expert"}},
	})
	if len(catalog.Match("SW39 Heavy", "expert")) != 1 {
		t.Fatal("expected prefix+suffix match")
	}
	if len(catalog.Match("SW39", "expert")) != 0 {
		t.Fatal("expected suffix miss")
	}
}

func TestLiveVACatalogMatchMultipleVAs(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "AA", Prefix: "DAL", PreferredServers: []string{"expert"}},
		{VAID: "va-2", VACode: "BB", Suffix: "Heavy", PreferredServers: []string{"expert"}},
	})
	matches := catalog.Match("DAL123 Heavy", "expert")
	if len(matches) != 2 {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestSuffixMatchesAfterFlightOfTrailer(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "TST", Suffix: "VA", PreferredServers: []string{"expert"}},
	})
	matches := catalog.Match("Airline 001VA Flight of 5", "expert")
	if len(matches) != 1 {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestLiveVACatalogMatchSuffixBeforeWeightClass(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "DAL", Suffix: "AK", PreferredServers: []string{"expert"}},
	})
	matches := catalog.Match("Delta 095AK Heavy", "expert")
	if len(matches) != 1 || matches[0].VAID != "va-1" {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestAffixMissReason(t *testing.T) {
	catalog := BuildLiveVACatalog([]basicConfigRow{
		{VAID: "va-1", VACode: "DAL", Suffix: "AK", PreferredServers: []string{"expert"}},
	})
	if catalog.AffixMissReason("Delta 095AK Heavy", "casual") != "no_eligible_vas_for_session" {
		t.Fatalf("casual reason = %q", catalog.AffixMissReason("Delta 095AK Heavy", "casual"))
	}
	if catalog.AffixMissReason("Delta 095AK Heavy", "expert") != "" {
		t.Fatalf("expert reason = %q", catalog.AffixMissReason("Delta 095AK Heavy", "expert"))
	}
}

func TestParseBasicConfigRow(t *testing.T) {
	row, err := parseBasicConfigRow("id-1", "TST", []byte(`{
		"game_callsign":{"prefix":" IE ","suffix":""},
		"preferred_servers":[" Expert ","expert"]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if row.Prefix != "IE" || len(row.PreferredServers) != 1 || row.PreferredServers[0] != "expert" {
		t.Fatalf("row = %#v", row)
	}
}
