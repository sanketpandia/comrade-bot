package flightresolution

var vaMatchEffect = EffectDecision{
	Name:    "VAMatch",
	Applies: vaMatchApplies,
	Apply:   applyVAMatch,
}

func vaMatchApplies(env *resolveEnv) bool {
	if env.syncedUpstream {
		return false
	}
	return callsignAffixFieldsDiffer(env)
}

func applyVAMatch(env *resolveEnv) {
	applyCallsignAffixFields(env)
}
