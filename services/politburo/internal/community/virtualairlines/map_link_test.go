package virtualairlines

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/routes"
)

func TestActiveFlightsMapPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		vaCode   string
		serverID string
		want     string
	}{
		{name: "bare", want: routes.MapsFlightsActive},
		{name: "va only", vaCode: "tst", want: routes.MapsFlightsActive + "?vaCode=TST"},
		{name: "both", vaCode: "tst", serverID: "expert", want: routes.MapsFlightsActive + "?serverId=expert&vaCode=TST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activeFlightsMapPath(tt.vaCode, tt.serverID)
			if got != tt.want {
				t.Fatalf("activeFlightsMapPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

type stubVARepo struct {
	va  *VirtualAirline
	err error
}

func (s stubVARepo) GetByID(context.Context, string) (*VirtualAirline, error) {
	return s.va, s.err
}

type stubConfigRepo struct {
	block *vaconfig.Block
	err   error
}

func (s stubConfigRepo) Get(context.Context, string, string) (*vaconfig.Block, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.block, nil
}

func TestMapLinkActiveFlightsMapPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	basic, err := json.Marshal(map[string]any{
		"game_callsign":     map[string]string{"prefix": "X"},
		"preferred_servers": []string{"expert", "casual"},
	})
	if err != nil {
		t.Fatal(err)
	}

	link := &MapLink{
		VA:     stubVARepo{va: &VirtualAirline{ID: "va-1", Code: "TST"}},
		Config: stubConfigRepo{block: &vaconfig.Block{Config: basic}},
	}

	got, err := link.ActiveFlightsMapPath(ctx, "va-1")
	if err != nil {
		t.Fatalf("ActiveFlightsMapPath() err = %v", err)
	}
	if !strings.Contains(got, "vaCode=TST") || !strings.Contains(got, "serverId=casual") {
		t.Fatalf("path = %q", got)
	}

	got, err = link.ActiveFlightsMapPath(ctx, "")
	if err != nil {
		t.Fatalf("empty vaID err = %v", err)
	}
	if got != routes.MapsFlightsActive {
		t.Fatalf("empty vaID path = %q", got)
	}

	linkNoBasic := &MapLink{
		VA:     stubVARepo{va: &VirtualAirline{ID: "va-1", Code: "TST"}},
		Config: stubConfigRepo{err: vaconfig.ErrNotFound},
	}
	got, err = linkNoBasic.ActiveFlightsMapPath(ctx, "va-1")
	if err != nil {
		t.Fatalf("no basic err = %v", err)
	}
	if got != routes.MapsFlightsActive+"?vaCode=TST" {
		t.Fatalf("path = %q", got)
	}
}

func TestMapLinkActiveFlightsMapPathVAErrors(t *testing.T) {
	t.Parallel()
	link := &MapLink{
		VA:     stubVARepo{err: errors.New("db down")},
		Config: stubConfigRepo{},
	}
	_, err := link.ActiveFlightsMapPath(context.Background(), "va-1")
	if err == nil {
		t.Fatal("expected error")
	}
}
