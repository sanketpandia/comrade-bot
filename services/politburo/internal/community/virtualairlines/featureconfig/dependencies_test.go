package featureconfig

import (
	"context"
	"encoding/json"
	"testing"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

type memRepo struct {
	blocks map[string]*vaconfig.Block
}

func (m *memRepo) Get(_ context.Context, vaID, configType string) (*vaconfig.Block, error) {
	b, ok := m.blocks[vaID+":"+configType]
	if !ok {
		return nil, vaconfig.ErrNotFound
	}
	return b, nil
}

func TestGuardBlocksClearPilotMapping(t *testing.T) {
	raw, _ := json.Marshal(StatsDocument{Pilot: &StatsSection{MatchBy: "callsign", Fields: []string{"displayName"}}})
	repo := &memRepo{blocks: map[string]*vaconfig.Block{
		"va-1:" + vaconfig.TypeMapStats: {Config: raw},
	}}
	g := NewGuard(repo)
	if err := g.BeforeClearMapping(context.Background(), "va-1", connections.EntityPilots); err == nil {
		t.Fatal("expected dependency error")
	}
}