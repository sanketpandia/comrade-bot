package va

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/transport/http/response"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const maxConfigBodyBytes = 64 << 10

type configBlocksRepo interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
	Upsert(ctx context.Context, vaID, configType string, config json.RawMessage, updatedBy string) (*vaconfig.Block, error)
}

type discordUserLookup interface {
	GetByDiscordID(ctx context.Context, discordID string) (*users.User, error)
}

func (h *Handler) GetVAConfig(w http.ResponseWriter, r *http.Request, configType string) {
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}

	block, err := h.configs.Get(r.Context(), vaID, configType)
	if err != nil {
		if errors.Is(err, vaconfig.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "config not found")
			return
		}
		slog.Error("load va config", "va_id", vaID, "type", configType, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load configuration")
		return
	}
	writeConfigResponse(w, http.StatusOK, block)
}

func (h *Handler) UpsertVAConfig(w http.ResponseWriter, r *http.Request, configType string) {
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}

	actorID, ok := h.resolveActorID(w, r)
	if !ok {
		return
	}

	raw, err := readConfigBody(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := vaconfig.Validate(configType, raw); err != nil {
		slog.Warn("reject va config",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"va_id", vaID,
			"type", configType,
			"error", err,
		)
		if errors.Is(err, vaconfig.ErrUnknownType) {
			response.WriteError(w, http.StatusBadRequest, "UNKNOWN_CONFIG_TYPE", err.Error())
			return
		}
		if errors.Is(err, vaconfig.ErrInvalidConfig) {
			response.WriteError(w, http.StatusBadRequest, "INVALID_CONFIG", err.Error())
			return
		}
		response.WriteError(w, http.StatusBadRequest, "INVALID_CONFIG", err.Error())
		return
	}

	block, err := h.configs.Upsert(r.Context(), vaID, configType, raw, actorID)
	if err != nil {
		slog.Error("upsert va config", "va_id", vaID, "type", configType, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "SAVE_FAILED", "failed to store configuration")
		return
	}
	writeConfigResponse(w, http.StatusOK, block)
}

func (h *Handler) vaContext(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return "", false
	}
	vaID := claims.PbServerID
	if vaID == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return "", false
	}
	va, err := h.vas.GetByID(r.Context(), vaID)
	if err != nil {
		slog.Error("load virtual airline for config", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load virtual airline")
		return "", false
	}
	if va == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return "", false
	}
	return vaID, true
}

func (h *Handler) resolveActorID(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return "", false
	}
	if claims.PbUserID != "" {
		return claims.PbUserID, true
	}
	if claims.DsUserID == "" || h.users == nil {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated Politburo user required")
		return "", false
	}
	user, err := h.users.GetByDiscordID(r.Context(), claims.DsUserID)
	if err != nil {
		slog.Error("resolve user for config audit", "discord_id", claims.DsUserID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to resolve user")
		return "", false
	}
	if user == nil {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated Politburo user required")
		return "", false
	}
	return user.ID, true
}

func readConfigBody(r *http.Request) (json.RawMessage, error) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxConfigBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, errors.New("invalid request body")
	}
	if len(data) == 0 {
		return nil, errors.New("request body required")
	}
	if len(data) > maxConfigBodyBytes {
		return nil, errors.New("request body too large")
	}
	if !json.Valid(data) {
		return nil, errors.New("invalid JSON")
	}
	return json.RawMessage(data), nil
}

func writeConfigResponse(w http.ResponseWriter, status int, block *vaconfig.Block) {
	var configObj map[string]any
	if err := json.Unmarshal(block.Config, &configObj); err != nil {
		configObj = map[string]any{}
	}
	response.WriteJSON(w, status, map[string]any{
		"data": map[string]any{
			"type":      block.ConfigType,
			"config":    configObj,
			"updatedAt": block.UpdatedAt,
			"updatedBy": block.UpdatedBy,
		},
	})
}
