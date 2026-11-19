package pireps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/virtualairlines/pirepvaluemappings"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type valueMappings interface {
	Get(ctx context.Context, vaID string) (pirepvaluemappings.View, error)
	ResolveForVA(ctx context.Context, vaID string, input pirepvaluemappings.ResolveInput) (pirepvaluemappings.ResolveResult, error)
}

type Handler struct {
	mappings valueMappings
}

func NewHandler(mappings valueMappings) *Handler {
	return &Handler{mappings: mappings}
}

func (h *Handler) GetPirepConfig(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	vaID := claims.PbServerID
	if vaID == "" || h.mappings == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}
	view, err := h.mappings.Get(r.Context(), vaID)
	if err != nil {
		logPirepFilingFailure(vaID, "config", "load_mappings", err, false, false)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load mappings")
		return
	}
	input := pirepvaluemappings.ResolveInput{
		AircraftID: r.URL.Query().Get("aircraftId"),
		LiveryID:   r.URL.Query().Get("liveryId"),
	}
	resolve, err := h.mappings.ResolveForVA(r.Context(), vaID, input)
	if err != nil {
		logPirepFilingFailure(vaID, "config", "resolve", err, input.AircraftID != "", input.LiveryID != "")
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to resolve filing values")
		return
	}
	if resolve.Aircraft.Source == pirepvaluemappings.SourceNeedsValue || resolve.Airline.Source == pirepvaluemappings.SourceNeedsValue {
		slog.Warn("pirep config unresolved filing values",
			"component", "pirep_filing",
			"va_id", vaID,
			"aircraft_source", resolve.Aircraft.Source,
			"airline_source", resolve.Airline.Source,
			"has_aircraft_id", input.AircraftID != "",
			"has_livery_id", input.LiveryID != "",
			"mappings_revision", view.Revision,
		)
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user_info": map[string]any{
				"current_aircraft": resolve.Aircraft.Value,
				"current_livery":   resolve.Airline.Value,
			},
			"available_modes":      []any{},
			"mappingsRevision":     view.Revision,
			"resolvedAircraftRaw":  resolve.Aircraft.RawInput,
			"resolvedAirlineRaw":   resolve.Airline.RawInput,
			"resolvedAircraftRule": resolve.Aircraft.Source,
			"resolvedAirlineRule":  resolve.Airline.Source,
		},
	})
}

type submitBody struct {
	Mode            string `json:"mode"`
	MappingRevision *int64 `json:"mappingRevision"`
}

func (h *Handler) SubmitPirep(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	vaID := claims.PbServerID
	if vaID == "" || h.mappings == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	var in submitBody
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
			return
		}
	}
	view, err := h.mappings.Get(r.Context(), vaID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load mappings")
		return
	}
	if in.MappingRevision != nil && *in.MappingRevision != view.Revision {
		slog.Warn("pirep submit stale mappings revision",
			"component", "pirep_filing",
			"va_id", vaID,
			"client_revision", *in.MappingRevision,
			"server_revision", view.Revision,
		)
		response.WriteError(w, http.StatusPreconditionFailed, "STALE_MAPPINGS", "pirep value mappings changed; refresh preview")
		return
	}
	resolve, err := h.mappings.ResolveForVA(r.Context(), vaID, pirepvaluemappings.ResolveInput{
		AircraftID: r.URL.Query().Get("aircraftId"),
		LiveryID:   r.URL.Query().Get("liveryId"),
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to resolve filing values")
		return
	}
	if resolve.Aircraft.Source == pirepvaluemappings.SourceNeedsValue || resolve.Airline.Source == pirepvaluemappings.SourceNeedsValue {
		slog.Warn("pirep submit unresolved filing values",
			"component", "pirep_filing",
			"va_id", vaID,
			"aircraft_source", resolve.Aircraft.Source,
			"airline_source", resolve.Airline.Source,
		)
		response.WriteError(w, http.StatusUnprocessableEntity, "UNRESOLVED_VALUES", "aircraft or airline filing values are not configured")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"pirep_id":     fmt.Sprintf("pending-%s", vaID),
			"aircraft":     resolve.Aircraft.Value,
			"livery":       resolve.Airline.Value,
			"providerNote": "destination write pending MAP_PIREP provider wiring",
		},
	})
}

func logPirepFilingFailure(vaID, operation, step string, err error, hasAircraftID, hasLiveryID bool) {
	slog.Warn("pirep filing request failed",
		"component", "pirep_filing",
		"va_id", vaID,
		"operation", operation,
		"step", step,
		"has_aircraft_id", hasAircraftID,
		"has_livery_id", hasLiveryID,
		"error", err.Error(),
	)
}