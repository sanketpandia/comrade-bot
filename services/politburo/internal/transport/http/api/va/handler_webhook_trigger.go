package va

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/transport/http/response"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
)

const maxWebhookTriggerBodyBytes = 4 << 10

type webhookTrigger interface {
	Trigger(ctx context.Context, vaID, webhookType string) (dispatch.Result, error)
}

type webhookTriggerRequest struct {
	WebhookType string `json:"webhookType"`
}

func (h *Handler) TriggerVAWebhook(w http.ResponseWriter, r *http.Request) {
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}

	body, err := readWebhookTriggerBody(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	webhookType := strings.TrimSpace(body.WebhookType)
	if webhookType == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "webhookType is required")
		return
	}

	result, err := h.webhooks.Trigger(r.Context(), vaID, webhookType)
	if err != nil {
		switch {
		case errors.Is(err, vaconfig.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "webhooks config not found")
		case errors.Is(err, webhookjobs.ErrFeatureNotFound):
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		default:
			slog.Error("trigger webhook", "va_id", vaID, "webhook_type", webhookType, "error", err)
			response.WriteError(w, http.StatusInternalServerError, "DISPATCH_FAILED", "failed to dispatch webhook")
		}
		return
	}

	data := map[string]any{
		"webhookType": webhookType,
		"outcome":     result.Outcome,
	}
	if result.SkipReason != "" {
		data["skipReason"] = result.SkipReason
	}
	if result.DiscordMessages > 0 {
		data["discordMessages"] = result.DiscordMessages
	}
	if result.ErrorMessage != "" {
		data["errorMessage"] = result.ErrorMessage
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

func readWebhookTriggerBody(r *http.Request) (webhookTriggerRequest, error) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxWebhookTriggerBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return webhookTriggerRequest{}, errors.New("invalid request body")
	}
	if len(data) == 0 {
		return webhookTriggerRequest{}, errors.New("request body required")
	}
	if len(data) > maxWebhookTriggerBodyBytes {
		return webhookTriggerRequest{}, errors.New("request body too large")
	}
	var body webhookTriggerRequest
	if err := json.Unmarshal(data, &body); err != nil {
		return webhookTriggerRequest{}, errors.New("invalid JSON")
	}
	return body, nil
}
