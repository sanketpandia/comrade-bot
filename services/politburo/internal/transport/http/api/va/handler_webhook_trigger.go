package va

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/transport/http/response"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
)

const maxWebhookTriggerBodyBytes = 4 << 10

type webhookTargetLoader interface {
	LoadWebhookDispatchTarget(ctx context.Context, vaID, webhookType string) (virtualairlines.WebhookDispatchTarget, error)
}

type webhookDispatcher interface {
	DispatchOne(ctx context.Context, item dispatch.Item) (dispatch.Result, error)
}

type webhookTriggerRequest struct {
	WebhookType string `json:"webhookType"`
}

func (h *Handler) TriggerVAWebhook(w http.ResponseWriter, r *http.Request) {
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	if h.webhookTargets == nil || h.webhookDispatch == nil {
		slog.Error("webhook trigger not configured")
		response.WriteError(w, http.StatusInternalServerError, "DISPATCH_FAILED", "webhook dispatch unavailable")
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

	target, err := h.webhookTargets.LoadWebhookDispatchTarget(r.Context(), vaID, webhookType)
	if err != nil {
		if errors.Is(err, virtualairlines.ErrWebhookConfigNotFound) || errors.Is(err, virtualairlines.ErrWebhookTypeNotFound) {
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		slog.Error("load webhook dispatch target", "va_id", vaID, "webhook_type", webhookType, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load webhook configuration")
		return
	}

	result, err := h.webhookDispatch.DispatchOne(r.Context(), dispatch.Item{
		VAID:             target.VAID,
		VACode:           target.VACode,
		WebhookType:      target.WebhookType,
		URL:              target.URL,
		Filters:          target.Filters,
		PreferredServers: target.PreferredServers,
	})
	if err != nil {
		slog.Error("dispatch webhook", "va_id", vaID, "webhook_type", webhookType, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "DISPATCH_FAILED", "failed to record webhook dispatch")
		return
	}

	data := map[string]any{
		"webhookType": target.WebhookType,
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
