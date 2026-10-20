package flightresolution

import (
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
)

func applyCallsignAffixFields(env *resolveEnv) {
	if !env.hasFlight && env.prior != nil {
		env.flight = *env.prior
		env.hasFlight = true
	}
	if !env.hasFlight {
		return
	}

	normalized, matches := callsignAffixFields(env)
	changed := env.flight.NormalizedCallsign != normalized ||
		!virtualAirlineMatchesEqual(env.flight.MatchedVirtualAirlines, matches)
	if !changed {
		return
	}

	env.flight.NormalizedCallsign = normalized
	env.flight.MatchedVirtualAirlines = matches
	env.recordMutated = true
}

func callsignAffixFields(env *resolveEnv) (string, []virtualairlines.VirtualAirlineMatch) {
	callsign := env.upstream.Callsign
	normalized := flights.NormalizedCallsignForAffix(callsign)
	return normalized, env.ctx.LiveVACatalog.Match(callsign, normalized, env.ctx.Session.NormalizedName)
}

func callsignAffixFieldsDiffer(env *resolveEnv) bool {
	normalized, matches := callsignAffixFields(env)
	if env.prior == nil {
		return normalized != "" || len(matches) > 0
	}
	return env.prior.NormalizedCallsign != normalized ||
		!virtualAirlineMatchesEqual(env.prior.MatchedVirtualAirlines, matches)
}

func virtualAirlineMatchesEqual(a, b []virtualairlines.VirtualAirlineMatch) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
