package va

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/community/virtualairlines"
)

func configTestVA() vaRepoStub {
	return vaRepoStub{va: &virtualairlines.VirtualAirline{ID: "va-1", Name: "Test", Code: "TST", IsActive: true}}
}

type configRepoStub struct {
	block *vaconfig.Block
	getErr error
	upsert *vaconfig.Block
	upsertErr error
}

func (s configRepoStub) Get(_ context.Context, vaID, configType string) (*vaconfig.Block, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.block == nil {
		return nil, vaconfig.ErrNotFound
	}
	return s.block, nil
}

func (s configRepoStub) Upsert(_ context.Context, vaID, configType string, config json.RawMessage, updatedBy string) (*vaconfig.Block, error) {
	if s.upsertErr != nil {
		return nil, s.upsertErr
	}
	if s.upsert != nil {
		return s.upsert, nil
	}
	return &vaconfig.Block{
		VAID:       vaID,
		ConfigType: configType,
		Config:     config,
		UpdatedAt:  time.Now().UTC(),
		UpdatedBy:  updatedBy,
	}, nil
}

type userRepoStub struct {
	user *users.User
	err  error
}

func (s userRepoStub) GetByDiscordID(_ context.Context, _ string) (*users.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func TestGetVAConfigNotFound(t *testing.T) {
	handler := &Handler{vas: configTestVA(), configs: configRepoStub{}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/config?type=BASIC", nil), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.GetVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGetVAConfigOK(t *testing.T) {
	updated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	handler := &Handler{vas: configTestVA(), configs: configRepoStub{block: &vaconfig.Block{
		VAID:       "va-1",
		ConfigType: vaconfig.TypeBasic,
		Config:     json.RawMessage(`{"game_callsign":{"prefix":"IE","suffix":""}}`),
		UpdatedAt:  updated,
		UpdatedBy:  "user-1",
	}}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/config?type=BASIC", nil), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.GetVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	if data["type"] != "BASIC" || data["updatedBy"] != "user-1" {
		t.Fatalf("data = %#v", data)
	}
}

func TestUpsertVAConfigRequiresActor(t *testing.T) {
	handler := &Handler{vas: configTestVA(), configs: configRepoStub{}}
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"game_callsign":{"prefix":"IE","suffix":""}}`)
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", body), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.UpsertVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestUpsertVAConfigOK(t *testing.T) {
	handler := &Handler{
		vas:     configTestVA(),
		configs: configRepoStub{},
	}
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"game_callsign":{"prefix":"IE","suffix":""}}`)
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", body), auth.Claims{
		PbServerID: "va-1",
		PbUserID:   "user-1",
		Role:       "administrator",
	})
	handler.UpsertVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body)
	}
}

func TestUpsertVAConfigInvalidConfig(t *testing.T) {
	handler := &Handler{vas: configTestVA(), configs: configRepoStub{}}
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"game_callsign":{"prefix":1}}`)
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", body), auth.Claims{
		PbServerID: "va-1",
		PbUserID:   "user-1",
		Role:       "administrator",
	})
	handler.UpsertVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestUpsertVAConfigResolvesDiscordUser(t *testing.T) {
	handler := &Handler{
		vas:     configTestVA(),
		configs: configRepoStub{},
		users:   userRepoStub{user: &users.User{ID: "user-from-discord"}},
	}
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"game_callsign":{"prefix":"X","suffix":""}}`)
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", body), auth.Claims{
		PbServerID: "va-1",
		DsUserID:   "discord-1",
		Role:       "administrator",
	})
	handler.UpsertVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body)
	}
}

func TestUpsertVAConfigSaveError(t *testing.T) {
	handler := &Handler{
		vas:     configTestVA(),
		configs: configRepoStub{upsertErr: errors.New("db")},
	}
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"game_callsign":{"prefix":"IE","suffix":""}}`)
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", body), auth.Claims{
		PbServerID: "va-1",
		PbUserID:   "user-1",
		Role:       "administrator",
	})
	handler.UpsertVAConfig(recorder, req, vaconfig.TypeBasic)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
