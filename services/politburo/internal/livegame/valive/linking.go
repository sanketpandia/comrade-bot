package valive

import (
	"context"
	"fmt"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const (
	roleAdmin = 1
	roleOwner = 2
)

// LinkAdminOrganizationsResult counts orgs linked vs skipped because another VA owns them.
type LinkAdminOrganizationsResult struct {
	Linked           int
	SkippedOccupied  int
}

// LinkAdminOrganizations upserts connections for orgs the user administers.
func LinkAdminOrganizations(
	ctx context.Context,
	connections *ConnectionsRepository,
	vaID, actorUserID string,
	orgs []infiniteflight.LiveOrganization,
) (LinkAdminOrganizationsResult, error) {
	var result LinkAdminOrganizationsResult
	for _, org := range orgs {
		if org.Role != roleAdmin && org.Role != roleOwner {
			continue
		}
		existing, err := connections.GetByOrgID(ctx, org.ID)
		if err != nil {
			return result, err
		}
		if existing != nil && existing.VAID != vaID {
			if err := connections.RecordSkippedOccupied(ctx, vaID, org.ID, actorUserID, org.Name); err != nil {
				return result, err
			}
			result.SkippedOccupied++
			continue
		}
		if err := connections.Upsert(ctx, vaID, org.ID, org.Name, org.Role, actorUserID); err != nil {
			return result, fmt.Errorf("link organization %s: %w", org.ID, err)
		}
		result.Linked++
	}
	return result, nil
}
