package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
	"infinite-experiment/politburo/internal/secretstore"
)

type stubBlocks struct {
	raw json.RawMessage
}

func (s stubBlocks) Get(context.Context, string, string) (*vaconfig.Block, error) {
	return &vaconfig.Block{Config: s.raw}, nil
}

func (s stubBlocks) UpsertIfRevision(context.Context, string, string, json.RawMessage, int, string) (*vaconfig.Block, error) {
	return nil, errors.New("unexpected write")
}

func TestListStopsBeforeProviderOnBadCursorOrCredential(t *testing.T) {
	root := bytes.Repeat([]byte{3}, 32)
	keys, err := secretstore.Derive(root)
	if err != nil {
		t.Fatalf("Derive() = %v", err)
	}
	token := "pat-secret-do-not-leak"
	env, err := keys.AtRest.Encrypt(secretstore.Context{
		Purpose:  airtablePATPurpose,
		OwnerID:  "va-1",
		RecordID: "conn-1",
	}, []byte(token))
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	doc := sampleDocument(env)
	raw, err := doc.Marshal()
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if strings.Contains(string(raw), token) {
		t.Fatalf("stored document contains the PAT: %s", raw)
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	t.Cleanup(srv.Close)
	client := airtable.NewClient(nil, time.Second).WithBaseURL(srv.URL)
	svc, err := NewService(stubBlocks{raw: raw}, client, keys.AtRest, keys.CursorMAC, nil)
	if err != nil {
		t.Fatalf("NewService() = %v", err)
	}

	_, err = svc.List(context.Background(), "va-1", EntityPilots, nil, 25, "not-a-cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("List() error = %v, want invalid cursor", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls after invalid cursor = %d", calls)
	}

	swapped := doc.Connections[0]
	swapped.ID = "conn-2"
	if _, err := svc.tokenForConnection("va-1", swapped); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("swapped connection error = %v", err)
	}
	if _, err := svc.tokenForConnection("va-2", doc.Connections[0]); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("swapped VA error = %v", err)
	}

	doc.Connections[0].Airtable.Credential.Sealed = []byte{1, 2, 3}
	corrupt, err := doc.Marshal()
	if err != nil {
		t.Fatalf("Marshal() corrupt = %v", err)
	}
	svc, err = NewService(stubBlocks{raw: corrupt}, client, keys.AtRest, keys.CursorMAC, nil)
	if err != nil {
		t.Fatalf("NewService() corrupt = %v", err)
	}
	_, err = svc.List(context.Background(), "va-1", EntityPilots, nil, 25, "")
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("List() corrupt error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls after corrupt credential = %d", calls)
	}
}

func sampleDocument(env secretstore.Envelope) Document {
	return Document{
		Version:  documentVersion,
		Revision: 2,
		Connections: []Connection{{
			ID:       "conn-1",
			Name:     "Operations",
			Provider: ProviderAirtable,
			Airtable: &AirtableSetup{
				Credential: env,
				Bases:      []BaseSetup{{ID: "base-setup", Name: "Ops", BaseID: "appExample"}},
			},
		}},
		Mappings: map[string]Mapping{
			EntityPilots: {
				ConnectionID: "conn-1",
				BaseSetupID:  "base-setup",
				TableID:      "tblExample",
				Fields:       map[string]string{"callsign": "fldExample"},
			},
		},
	}
}
