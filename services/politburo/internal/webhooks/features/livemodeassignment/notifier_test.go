package livemodeassignment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/discord"
)

type configStub struct {
	block *vaconfig.Block
	err   error
}

func (s configStub) Get(context.Context, string, string) (*vaconfig.Block, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.block, nil
}

type posterStub struct {
	url     string
	embeds  []discord.Embed
	webType string
	err     error
}

func (p *posterStub) PostEmbeds(_ context.Context, webhookType, webhookURL string, embeds []discord.Embed) error {
	p.webType = webhookType
	p.url = webhookURL
	p.embeds = embeds
	return p.err
}

func TestNotifyNoConfig(t *testing.T) {
	n := New(configStub{err: vaconfig.ErrNotFound}, &posterStub{})
	if err := n.Notify(context.Background(), "va-1", Event{IFUserID: "x"}); err != nil {
		t.Fatalf("Notify() = %v", err)
	}
}

func TestNotifyMissingFeature(t *testing.T) {
	raw := json.RawMessage(`{"active_flights":{"url":"https://discord.com/api/webhooks/1/t","frequency":"30m","filters":{}}}`)
	n := New(configStub{block: &vaconfig.Block{Config: raw}}, &posterStub{})
	if err := n.Notify(context.Background(), "va-1", Event{IFUserID: "x"}); err != nil {
		t.Fatalf("Notify() = %v", err)
	}
}

func TestNotifyPosts(t *testing.T) {
	raw := json.RawMessage(`{"live_mode_assignment":{"url":"https://discord.com/api/webhooks/9/secret"}}`)
	poster := &posterStub{}
	n := New(configStub{block: &vaconfig.Block{Config: raw}}, poster)
	ev := Event{
		IFUserID:        "if-1",
		PilotUsername:   "Pilot",
		DiscordID:       "123",
		Registration:    "N123",
		OriginIcao:      "KJFK",
		DestinationIcao: "EGLL",
	}
	if err := n.Notify(context.Background(), "va-1", ev); err != nil {
		t.Fatalf("Notify() = %v", err)
	}
	if poster.webType != webhookconfig.FeatureLiveModeAssignment {
		t.Fatalf("webhook type = %q", poster.webType)
	}
	if poster.url == "" || len(poster.embeds) != 1 {
		t.Fatalf("post = url %q embeds %d", poster.url, len(poster.embeds))
	}
}

func TestNotifyPostError(t *testing.T) {
	raw := json.RawMessage(`{"live_mode_assignment":{"url":"https://discord.com/api/webhooks/9/secret"}}`)
	poster := &posterStub{err: errors.New("discord down")}
	n := New(configStub{block: &vaconfig.Block{Config: raw}}, poster)
	err := n.Notify(context.Background(), "va-1", Event{IFUserID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}
