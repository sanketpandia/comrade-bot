package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestApplyFlightUsersSeedsFirstUser(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	out := flights.Derived{}
	applyFlightUsers(EffectInput{
		Raw: flights.Observation{UserID: userA},
	}, &out, nil)
	if len(out.Users) != 1 || out.Users[0] != userA {
		t.Fatalf("users = %#v", out.Users)
	}
}

func TestApplyFlightUsersAppendsOnHandoff(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	userB := "0d85b360-92b5-4e62-ac64-41bd1c829772"
	out := flights.Derived{Users: []string{userA}}
	prior := &flights.Record{Raw: flights.Observation{UserID: userA}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userB},
		Prior: prior,
	}, &out, nil)
	if len(out.Users) != 2 || out.Users[1] != userB {
		t.Fatalf("users = %#v", out.Users)
	}
}

func TestApplyFlightUsersSkipsDuplicateOnResume(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	userB := "0d85b360-92b5-4e62-ac64-41bd1c829772"
	out := flights.Derived{Users: []string{userA, userB}}
	prior := &flights.Record{Raw: flights.Observation{UserID: userB}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userA},
		Prior: prior,
	}, &out, nil)
	if len(out.Users) != 2 {
		t.Fatalf("users = %#v", out.Users)
	}
}

func TestApplyFlightUsersBackfillsLegacyEmptyList(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	out := flights.Derived{}
	prior := &flights.Record{Raw: flights.Observation{UserID: userA}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userA},
		Prior: prior,
	}, &out, nil)
	if len(out.Users) != 1 || out.Users[0] != userA {
		t.Fatalf("users = %#v", out.Users)
	}
}
