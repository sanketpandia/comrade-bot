// Package livemodeassignment posts Discord notifications after a successful assign-to-me.
package livemodeassignment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/discord"
)

type configLoader interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
}

type poster interface {
	PostEmbeds(ctx context.Context, webhookType, webhookURL string, embeds []discord.Embed) error
}

// Notifier loads a VA live_mode_assignment webhook URL and posts one embed per assignment.
type Notifier struct {
	configs configLoader
	discord poster
	now     func() time.Time
}

func New(configs configLoader, discord poster) *Notifier {
	return &Notifier{configs: configs, discord: discord, now: time.Now}
}

// Notify posts the assignment embed when configured. Missing WEBHOOKS config or feature entry is a no-op.
func (n *Notifier) Notify(ctx context.Context, vaID string, ev Event) error {
	if n == nil || n.configs == nil || n.discord == nil {
		return nil
	}
	vaID = strings.TrimSpace(vaID)
	if vaID == "" {
		return nil
	}
	block, err := n.configs.Get(ctx, vaID, vaconfig.TypeWebhooks)
	if err != nil {
		if errors.Is(err, vaconfig.ErrNotFound) {
			return nil
		}
		return err
	}
	entries, err := webhookconfig.Parse(block.Config)
	if err != nil {
		return fmt.Errorf("parse webhooks config: %w", err)
	}
	var url string
	for _, entry := range entries {
		if entry.Key == webhookconfig.FeatureLiveModeAssignment {
			url = strings.TrimSpace(entry.URL)
			break
		}
	}
	if url == "" {
		return nil
	}
	embed := buildEmbed(n.now(), ev)
	return n.discord.PostEmbeds(ctx, webhookconfig.FeatureLiveModeAssignment, url, []discord.Embed{embed})
}
