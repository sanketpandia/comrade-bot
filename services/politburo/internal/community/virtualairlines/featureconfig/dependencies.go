package featureconfig

import (
	"context"
	"errors"
	"fmt"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

type featureBlocks interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
}

// Guard validates connection edits against saved feature mappings.
type Guard struct {
	repo featureBlocks
}

func NewGuard(repo featureBlocks) *Guard {
	return &Guard{repo: repo}
}

func (g *Guard) BeforeClearMapping(ctx context.Context, vaID, entity string) error {
	return g.checkEntity(ctx, vaID, entity)
}

func (g *Guard) BeforeDeleteConnection(ctx context.Context, vaID, connectionID string) error {
	stats, pirep, err := g.load(ctx, vaID)
	if err != nil {
		return err
	}
	if statsUsesConnection(stats, connectionID) || pirepUsesConnection(pirep, connectionID) {
		return fmt.Errorf("%w: stats or pirep still use this connection", connections.ErrDependency)
	}
	return nil
}

func (g *Guard) BeforeDeleteBase(ctx context.Context, vaID, baseSetupID string) error {
	_ = vaID
	_ = baseSetupID
	return nil
}

func (g *Guard) checkEntity(ctx context.Context, vaID, entity string) error {
	stats, _, err := g.load(ctx, vaID)
	if err != nil {
		return err
	}
	if stats == nil {
		return nil
	}
	switch entity {
	case connections.EntityPilots:
		if stats.Pilot != nil {
			return fmt.Errorf("%w: stats pilot section uses pilots mapping", connections.ErrDependency)
		}
	case connections.EntityCareerPilots:
		if stats.Career != nil {
			return fmt.Errorf("%w: stats career section uses careerPilots mapping", connections.ErrDependency)
		}
	case connections.EntityPIREPs:
		if block, _ := g.repo.Get(ctx, vaID, vaconfig.TypeMapPIREP); block != nil {
			doc, err := ParsePIREPDocument(block.Config)
			if err == nil && len(doc.Fields) > 0 {
				return fmt.Errorf("%w: pirep mapping configured", connections.ErrDependency)
			}
		}
	}
	return nil
}

func (g *Guard) load(ctx context.Context, vaID string) (*StatsDocument, *PIREPDocument, error) {
	block, err := g.repo.Get(ctx, vaID, vaconfig.TypeMapStats)
	if err != nil && !errors.Is(err, vaconfig.ErrNotFound) {
		return nil, nil, err
	}
	var stats *StatsDocument
	if block != nil {
		doc, err := ParseStatsDocument(block.Config)
		if err != nil {
			return nil, nil, err
		}
		stats = &doc
	}
	block, err = g.repo.Get(ctx, vaID, vaconfig.TypeMapPIREP)
	if err != nil && !errors.Is(err, vaconfig.ErrNotFound) {
		return stats, nil, err
	}
	var pirep *PIREPDocument
	if block != nil {
		doc, err := ParsePIREPDocument(block.Config)
		if err != nil {
			return stats, nil, err
		}
		pirep = &doc
	}
	return stats, pirep, nil
}

func statsUsesConnection(stats *StatsDocument, connectionID string) bool {
	if stats == nil {
		return false
	}
	return stats.Pilot != nil || stats.Career != nil
}

func pirepUsesConnection(pirep *PIREPDocument, _ string) bool {
	return pirep != nil && len(pirep.Fields) > 0
}