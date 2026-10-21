package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var flightUsersEffect = EffectDecision{
	Name:    "FlightUsers",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyFlightUsers,
}

func applyFlightUsers(in EffectInput, out *flights.Derived, _ *Outcome) {
	userID := in.Raw.UserID
	if userID == "" {
		return
	}

	if len(out.Users) == 0 {
		out.Users = flights.AppendUserIDIfNew(out.Users, userID)
		return
	}

	if in.Prior == nil || userID == in.Prior.Raw.UserID {
		return
	}
	out.Users = flights.AppendUserIDIfNew(out.Users, userID)
}
