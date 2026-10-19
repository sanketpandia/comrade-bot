package flightresolution

var fplSyncEnqueueEffect = EffectDecision{
	Name:    "FPLSyncEnqueue",
	Applies: fplSyncEnqueueApplies,
	Apply:   applyFPLSyncEnqueue,
}

func fplSyncEnqueueApplies(env *resolveEnv) bool {
	return env.fplDue
}

func applyFPLSyncEnqueue(env *resolveEnv) {
	if !env.hasFlight {
		return
	}
	env.outcome.FPLSyncScheduled = true
	RunFPLSync(env.ctx.Context, env.flight)
}
