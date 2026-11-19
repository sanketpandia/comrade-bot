package va

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
	"infinite-experiment/politburo/internal/community/virtualairlines/featureconfig"
	"infinite-experiment/politburo/internal/transport/http/response"
)

const maxFeatureConfigBodyBytes = 64 << 10

type featureMappings interface {
	Enabled() bool
	Get(ctx context.Context, vaID, feature string) (featureconfig.FeatureView, error)
	SaveStats(ctx context.Context, vaID, actor string, expected, connectionsRevision int, doc featureconfig.StatsDocument) (featureconfig.FeatureView, error)
	SavePIREP(ctx context.Context, vaID, actor string, expected, connectionsRevision int, doc featureconfig.PIREPDocument) (featureconfig.FeatureView, error)
	Delete(ctx context.Context, vaID, feature string, expected int) error
	Preview(ctx context.Context, vaID, feature string, in featureconfig.PreviewInput) (featureconfig.PreviewResult, error)
}

func (h *Handler) writeFeatureMappingsDisabled(w http.ResponseWriter) {
	response.WriteError(w, http.StatusServiceUnavailable, "FEATURE_MAPPINGS_DISABLED", "feature mappings require data connections")
}

func (h *Handler) requireFeatureMappings(w http.ResponseWriter) (featureMappings, bool) {
	if h.featureMappings == nil || !h.featureMappings.Enabled() {
		h.writeFeatureMappingsDisabled(w)
		return nil, false
	}
	return h.featureMappings, true
}

func (h *Handler) GetVAFeatureConfig(w http.ResponseWriter, r *http.Request, feature string) {
	svc, ok := h.requireFeatureMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	view, err := svc.Get(r.Context(), vaID, feature)
	if err != nil {
		writeFeatureConfigError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *Handler) PutVAFeatureConfig(w http.ResponseWriter, r *http.Request, feature string) {
	svc, ok := h.requireFeatureMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	actor, ok := h.resolveActorID(w, r)
	if !ok {
		return
	}
	expected, valid := parseIfMatch(r, true)
	if !valid {
		response.WriteError(w, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match header required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxFeatureConfigBodyBytes+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if len(body) > maxFeatureConfigBodyBytes {
		response.WriteError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
		return
	}
	var view featureconfig.FeatureView
	switch feature {
	case featureconfig.FeatureStats:
		var wrap struct {
			ConnectionsRevision int `json:"connectionsRevision"`
			featureconfig.StatsDocument
		}
		if err := json.Unmarshal(body, &wrap); err != nil {
			response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid stats document")
			return
		}
		wrap.StatsDocument.Revision = expected
		view, err = svc.SaveStats(r.Context(), vaID, actor, expected, wrap.ConnectionsRevision, wrap.StatsDocument)
	case featureconfig.FeaturePIREP:
		var wrap struct {
			ConnectionsRevision int `json:"connectionsRevision"`
			featureconfig.PIREPDocument
		}
		if err := json.Unmarshal(body, &wrap); err != nil {
			response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid pirep document")
			return
		}
		wrap.PIREPDocument.Revision = expected
		view, err = svc.SavePIREP(r.Context(), vaID, actor, expected, wrap.ConnectionsRevision, wrap.PIREPDocument)
	default:
		response.WriteError(w, http.StatusBadRequest, "INVALID_FEATURE", "unknown feature")
		return
	}
	if err != nil {
		writeFeatureConfigError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *Handler) DeleteVAFeatureConfig(w http.ResponseWriter, r *http.Request, feature string) {
	svc, ok := h.requireFeatureMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	expected, valid := parseIfMatch(r, true)
	if !valid {
		response.WriteError(w, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match header required")
		return
	}
	if err := svc.Delete(r.Context(), vaID, feature, expected); err != nil {
		writeFeatureConfigError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

func (h *Handler) PreviewVAFeatureConfig(w http.ResponseWriter, r *http.Request, feature string) {
	svc, ok := h.requireFeatureMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxFeatureConfigBodyBytes+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	var in featureconfig.PreviewInput
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid preview body")
			return
		}
	}
	result, err := svc.Preview(r.Context(), vaID, feature, in)
	if err != nil {
		if errors.Is(err, featureconfig.ErrDisabled) {
			w.Header().Set("Cache-Control", "no-store")
			response.WriteJSON(w, http.StatusOK, map[string]any{"data": featureconfig.PreviewResult{Disabled: true}})
			return
		}
		writeFeatureConfigError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func writeFeatureConfigError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, featureconfig.ErrConflict):
		response.WriteError(w, http.StatusPreconditionFailed, "CONFLICT", "configuration was updated elsewhere")
	case errors.Is(err, featureconfig.ErrStaleConnections):
		response.WriteError(w, http.StatusUnprocessableEntity, "STALE_CONNECTIONS", err.Error())
	case errors.Is(err, featureconfig.ErrPilotNotFound):
		response.WriteError(w, http.StatusNotFound, "PILOT_NOT_FOUND", err.Error())
	case errors.Is(err, featureconfig.ErrAmbiguousPilot):
		response.WriteError(w, http.StatusUnprocessableEntity, "AMBIGUOUS_PILOT", err.Error())
	case errors.Is(err, featureconfig.ErrInvalidDraft), errors.Is(err, featureconfig.ErrInvalidFeature):
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIG", err.Error())
	case errors.Is(err, connections.ErrDependency):
		response.WriteError(w, http.StatusConflict, "FEATURE_DEPENDENCY", err.Error())
	case errors.Is(err, config.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, connections.ErrDisabled):
		response.WriteError(w, http.StatusServiceUnavailable, "FEATURE_MAPPINGS_DISABLED", err.Error())
	default:
		var pe *connections.ProviderError
		if errors.As(err, &pe) {
			writeConnectionFamilyError(w, r, err)
			return
		}
		writeConnectionFamilyError(w, r, err)
	}
}