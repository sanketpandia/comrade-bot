package virtualairlines

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/routes"
)

type vaByIDLookup interface {
	GetByID(ctx context.Context, id string) (*VirtualAirline, error)
}

type vaConfigGetter interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
}

// MapLink builds browser paths for VA-scoped live map views.
type MapLink struct {
	VA     vaByIDLookup
	Config vaConfigGetter
}

// ActiveFlightsMapPath returns a relative path to the active flights map with vaCode and
// optional serverId query params from the VA row and BASIC preferred_servers.
func (m *MapLink) ActiveFlightsMapPath(ctx context.Context, vaID string) (string, error) {
	if m == nil || m.VA == nil {
		return "", fmt.Errorf("map link: virtual airline repository required")
	}
	vaID = strings.TrimSpace(vaID)
	if vaID == "" {
		return routes.MapsFlightsActive, nil
	}

	va, err := m.VA.GetByID(ctx, vaID)
	if err != nil {
		return "", fmt.Errorf("map link lookup va: %w", err)
	}
	if va == nil {
		return "", fmt.Errorf("map link: virtual airline %q not found", vaID)
	}

	serverID := ""
	if m.Config != nil {
		block, err := m.Config.Get(ctx, vaID, vaconfig.TypeBasic)
		if err != nil {
			if !errors.Is(err, vaconfig.ErrNotFound) {
				return "", fmt.Errorf("map link load basic config: %w", err)
			}
		} else {
			basic, err := vaconfig.ParseBasic(block.Config)
			if err != nil {
				return "", fmt.Errorf("map link parse basic config: %w", err)
			}
			if len(basic.PreferredServers) > 0 {
				serverID = basic.PreferredServers[0]
			}
		}
	}

	return activeFlightsMapPath(va.Code, serverID), nil
}

func activeFlightsMapPath(vaCode, serverID string) string {
	vaCode = strings.ToUpper(strings.TrimSpace(vaCode))
	serverID = strings.TrimSpace(serverID)

	params := url.Values{}
	if vaCode != "" {
		params.Set("vaCode", vaCode)
	}
	if serverID != "" {
		params.Set("serverId", serverID)
	}
	encoded := params.Encode()
	if encoded == "" {
		return routes.MapsFlightsActive
	}
	return routes.MapsFlightsActive + "?" + encoded
}
