package va

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
)

type webhookTargetStub struct {
	target virtualairlines.WebhookDispatchTarget
	err    error
}

func (s webhookTargetStub) LoadWebhookDispatchTarget(_ context.Context, _, _ string) (virtualairlines.WebhookDispatchTarget, error) {
	if s.err != nil {
		return virtualairlines.WebhookDispatchTarget{}, s.err
	}
	return s.target, nil
}

type webhookDispatchStub struct {
	result dispatch.Result
	err    error
}

func (s webhookDispatchStub) DispatchOne(_ context.Context, _ dispatch.Item) (dispatch.Result, error) {
	return s.result, s.err
}

func TestTriggerVAWebhookSuccess(t *testing.T) {
	handler := &Handler{
		vas: configTestVA(),
		webhookTargets: webhookTargetStub{target: virtualairlines.WebhookDispatchTarget{
			VAID: "va-1", VACode: "TST", WebhookType: "active_flights",
			URL: "https://discord.com/api/webhooks/1/t",
		}},
		webhookDispatch: webhookDispatchStub{result: dispatch.Result{
			Outcome:         dispatch.OutcomeSuccess,
			DiscordMessages: 1,
		}},
	}
	body := bytes.NewBufferString(`{"webhookType":"active_flights"}`)
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/webhook/trigger", body), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.TriggerVAWebhook(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var payload map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if payload["data"]["outcome"] != dispatch.OutcomeSuccess {
		t.Fatalf("data = %#v", payload["data"])
	}
}

func TestTriggerVAWebhookNotFound(t *testing.T) {
	handler := &Handler{
		vas:            configTestVA(),
		webhookTargets: webhookTargetStub{err: virtualairlines.ErrWebhookTypeNotFound},
		webhookDispatch: webhookDispatchStub{},
	}
	body := bytes.NewBufferString(`{"webhookType":"hourly"}`)
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/webhook/trigger", body), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.TriggerVAWebhook(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestTriggerVAWebhookInvalidBody(t *testing.T) {
	handler := &Handler{vas: configTestVA(), webhookTargets: webhookTargetStub{}, webhookDispatch: webhookDispatchStub{}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/webhook/trigger", bytes.NewBufferString(`{}`)), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.TriggerVAWebhook(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestTriggerVAWebhookDispatchError(t *testing.T) {
	handler := &Handler{
		vas: configTestVA(),
		webhookTargets: webhookTargetStub{target: virtualairlines.WebhookDispatchTarget{
			VAID: "va-1", WebhookType: "active_flights",
		}},
		webhookDispatch: webhookDispatchStub{err: errors.New("db down")},
	}
	body := bytes.NewBufferString(`{"webhookType":"active_flights"}`)
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/webhook/trigger", body), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.TriggerVAWebhook(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
}
