package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

func applyCallsignAffixFields(env *resolveEnv) {
	if !env.hasFlight && env.prior != nil {
		env.flight = *env.prior
		env.hasFlight = true
	}
	if !env.hasFlight {
		return
	}

	callsign := env.upstream.Callsign
	normalized := flights.NormalizedCallsignForAffix(callsign)
	matches := catalogMatches(env, callsign)

	changed := env.flight.NormalizedCallsign != normalized ||
		!virtualAirlineMatchesEqual(env.flight.MatchedVirtualAirlines, matches)
	if !changed {
		return
	}

	env.flight.NormalizedCallsign = normalized
	env.flight.MatchedVirtualAirlines = matches
	env.recordMutated = true
}

func catalogMatches(env *resolveEnv, callsign string) []flights.VirtualAirlineMatch {
	raw := env.ctx.LiveVACatalog.Match(callsign, env.ctx.Session.NormalizedName)
	if len(raw) == 0 {
		return nil
	}
	out := make([]flights.VirtualAirlineMatch, len(raw))
	for i, m := range raw {
		out[i] = flights.VirtualAirlineMatch{VAID: m.VAID, VACode: m.VACode}
	}
	return out
}

func callsignAffixFieldsDiffer(env *resolveEnv) bool {
	callsign := env.upstream.Callsign
	normalized := flights.NormalizedCallsignForAffix(callsign)
	matches := catalogMatches(env, callsign)
	if env.prior == nil {
		return normalized != "" || len(matches) > 0
	}
	return env.prior.NormalizedCallsign != normalized ||
		!virtualAirlineMatchesEqual(env.prior.MatchedVirtualAirlines, matches)
}

func virtualAirlineMatchesEqual(a, b []flights.VirtualAirlineMatch) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].VAID != b[i].VAID || a[i].VACode != b[i].VACode {
			return false
		}
	}
	return true
}
