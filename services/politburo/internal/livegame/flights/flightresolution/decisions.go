package flightresolution

// EffectDecision is one composable resolution step with separate criteria and action.
type EffectDecision struct {
	Name    string
	Applies func(env *resolveEnv) bool
	Apply   func(env *resolveEnv)
}

var flightEffects = []EffectDecision{
	syncFromUpstreamEffect,
	liveryEnrichEffect,
	pathSyncUpdateEffect,
	fplSyncEnqueueEffect,
}
