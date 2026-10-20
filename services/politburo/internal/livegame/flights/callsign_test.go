package flights

import "testing"

func TestNormalizedCallsignForAffix(t *testing.T) {
	if got := NormalizedCallsignForAffix("Delta 095AK Heavy"); got != "Delta 095AK" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizedCallsignForAffix("Airline 001VA Flight of 5"); got != "Airline 001VA" {
		t.Fatalf("got %q", got)
	}
}

func TestCallsignAffixMatchesSuffixAfterHeavy(t *testing.T) {
	if !CallsignAffixMatches("Delta 095AK Heavy", "", "AK") {
		t.Fatal("expected AK suffix match")
	}
}

func TestCallsignAffixMatchesLiteralHeavySuffix(t *testing.T) {
	if !CallsignAffixMatches("SW39 Heavy", "SW", "Heavy") {
		t.Fatal("expected prefix+Heavy suffix on full callsign")
	}
}
