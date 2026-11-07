package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var flightUsersEffect = EffectDecision{
	Name:    "FlightUsers",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyFlightUsers,
}

func applyFlightUsers(in EffectInput, _ *flights.Analytics, _ *flights.Actions, links *flights.Links, _ *Outcome) {
	userID := in.Raw.UserID
	if userID == "" {
		return
	}

	if len(links.Users) == 0 {
		links.Users = flights.AppendUserIDIfNew(links.Users, userID)
		return
	}

	if in.Prior == nil || userID == in.Prior.Raw.UserID {
		return
	}
	links.Users = flights.AppendUserIDIfNew(links.Users, userID)
}
