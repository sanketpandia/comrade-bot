package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestApplyFlightUsersSeedsFirstUser(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	links := flights.Links{}
	applyFlightUsers(EffectInput{
		Raw: flights.Observation{UserID: userA},
	}, nil, nil, &links, nil)
	if len(links.Users) != 1 || links.Users[0] != userA {
		t.Fatalf("users = %#v", links.Users)
	}
}

func TestApplyFlightUsersAppendsOnHandoff(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	userB := "0d85b360-92b5-4e62-ac64-41bd1c829772"
	links := flights.Links{Users: []string{userA}}
	prior := &flights.Record{Raw: flights.Observation{UserID: userA}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userB},
		Prior: prior,
	}, nil, nil, &links, nil)
	if len(links.Users) != 2 || links.Users[1] != userB {
		t.Fatalf("users = %#v", links.Users)
	}
}

func TestApplyFlightUsersSkipsDuplicateOnResume(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	userB := "0d85b360-92b5-4e62-ac64-41bd1c829772"
	links := flights.Links{Users: []string{userA, userB}}
	prior := &flights.Record{Raw: flights.Observation{UserID: userB}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userA},
		Prior: prior,
	}, nil, nil, &links, nil)
	if len(links.Users) != 2 {
		t.Fatalf("users = %#v", links.Users)
	}
}

func TestApplyFlightUsersBackfillsLegacyEmptyList(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	links := flights.Links{}
	prior := &flights.Record{Raw: flights.Observation{UserID: userA}}
	applyFlightUsers(EffectInput{
		Raw:   flights.Observation{UserID: userA},
		Prior: prior,
	}, nil, nil, &links, nil)
	if len(links.Users) != 1 || links.Users[0] != userA {
		t.Fatalf("users = %#v", links.Users)
	}
}
