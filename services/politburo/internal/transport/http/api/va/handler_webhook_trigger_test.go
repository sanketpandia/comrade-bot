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
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
)

type webhookTriggerStub struct {
	result      dispatch.Result
	err         error
	vaID        string
	webhookType string
}

func (s *webhookTriggerStub) Trigger(_ context.Context, vaID, webhookType string) (dispatch.Result, error) {
	s.vaID, s.webhookType = vaID, webhookType
	return s.result, s.err
}

func TestTriggerVAWebhookSuccess(t *testing.T) {
	trigger := &webhookTriggerStub{result: dispatch.Result{Outcome: dispatch.OutcomeSuccess, DiscordMessages: 1}}
	handler := &Handler{vas: configTestVA(), webhooks: trigger}
	body := bytes.NewBufferString(`{"webhookType":" active_flights "}`)
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
	if payload["data"]["outcome"] != dispatch.OutcomeSuccess || payload["data"]["webhookType"] != "active_flights" {
		t.Fatalf("data = %#v", payload["data"])
	}
	if trigger.vaID != "va-1" || trigger.webhookType != "active_flights" {
		t.Fatalf("trigger = %+v", trigger)
	}
}

func TestTriggerVAWebhookNotFound(t *testing.T) {
	for _, err := range []error{webhookjobs.ErrFeatureNotFound, vaconfig.ErrNotFound} {
		handler := &Handler{vas: configTestVA(), webhooks: &webhookTriggerStub{err: err}}
		body := bytes.NewBufferString(`{"webhookType":"hourly"}`)
		recorder := httptest.NewRecorder()
		req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/va/webhook/trigger", body), auth.Claims{
			PbServerID: "va-1",
			Role:       "administrator",
		})
		handler.TriggerVAWebhook(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%v: status = %d", err, recorder.Code)
		}
	}
}

func TestTriggerVAWebhookInvalidBody(t *testing.T) {
	handler := &Handler{vas: configTestVA(), webhooks: &webhookTriggerStub{}}
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
	handler := &Handler{vas: configTestVA(), webhooks: &webhookTriggerStub{err: errors.New("db down")}}
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
