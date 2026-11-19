package pirepvaluemappings

import (
	"testing"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

func TestResolveExplicitAndDefault(t *testing.T) {
	other := "Other"
	generic := "Generic"
	cfg := Config{
		AircraftMappings: map[string]string{"Airbus A318": "A318-100"},
		AirlineMappings:  map[string]string{"KLM Cityhopper": "KLM"},
		DefaultAircraft:  &other,
		DefaultAirline:   &generic,
	}
	livery := gameliveries.AircraftLivery{
		LiveryID:     "l1",
		AircraftID:   "a1",
		AircraftName: "Airbus A318",
		LiveryName:   "KLM Cityhopper",
	}
	got := Resolve(cfg, livery, ResolveInput{AircraftID: "a1", LiveryID: "l1"})
	if got.Aircraft.Value != "A318-100" || got.Aircraft.Source != SourceExplicit {
		t.Fatalf("aircraft = %+v", got.Aircraft)
	}
	if got.Airline.Value != "KLM" || got.Airline.Source != SourceExplicit {
		t.Fatalf("airline = %+v", got.Airline)
	}

	livery2 := gameliveries.AircraftLivery{
		LiveryID:     "l2",
		AircraftID:   "a2",
		AircraftName: "Unknown Type",
		LiveryName:   "Some Livery",
	}
	got2 := Resolve(cfg, livery2, ResolveInput{AircraftID: "a2", LiveryID: "l2"})
	if got2.Aircraft.Value != "Other" || got2.Aircraft.Source != SourceDefault {
		t.Fatalf("aircraft default = %+v", got2.Aircraft)
	}
	if got2.Airline.Value != "Generic" || got2.Airline.Source != SourceDefault {
		t.Fatalf("airline default = %+v", got2.Airline)
	}
}

func TestResolveCatalogMismatch(t *testing.T) {
	cfg := Config{
		AirlineMappings: map[string]string{"Air France": "Air France"},
	}
	livery := gameliveries.AircraftLivery{
		LiveryID:     "l1",
		AircraftID:   "a1",
		AircraftName: "Airbus A320",
		LiveryName:   "Air France",
	}
	got := Resolve(cfg, livery, ResolveInput{AircraftID: "wrong", LiveryID: "l1"})
	if !got.CatalogMismatch {
		t.Fatal("expected catalog mismatch")
	}
	if got.Airline.Source != SourceCatalogMismatch {
		t.Fatalf("airline = %+v", got.Airline)
	}
}

func TestTwoVAsDifferentOutput(t *testing.T) {
	cfgA := Config{AircraftMappings: map[string]string{"Type X": "VAL-A"}}
	cfgB := Config{AircraftMappings: map[string]string{"Type X": "VAL-B"}}
	livery := gameliveries.AircraftLivery{AircraftName: "Type X", LiveryName: "Generic"}
	a := Resolve(cfgA, livery, ResolveInput{})
	b := Resolve(cfgB, livery, ResolveInput{})
	if a.Aircraft.Value != "VAL-A" || b.Aircraft.Value != "VAL-B" {
		t.Fatalf("got %q and %q", a.Aircraft.Value, b.Aircraft.Value)
	}
}