package va

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"infinite-experiment/politburo/internal/community/virtualairlines/pirepvaluemappings"
	"infinite-experiment/politburo/internal/transport/http/response"
)

const maxPIREPValueMappingsBody = 1 << 20

func (h *Handler) requirePIREPValueMappings(w http.ResponseWriter) (*pirepvaluemappings.Service, bool) {
	if h.pirepValueMappings == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "pirep value mappings unavailable")
		return nil, false
	}
	return h.pirepValueMappings, true
}

func (h *Handler) GetVAPIREPValueMappings(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requirePIREPValueMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	view, err := svc.Get(r.Context(), vaID)
	if err != nil {
		writePIREPValueMappingsError(w, vaID, "get", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.FormatInt(view.Revision, 10))
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *Handler) PutVAPIREPValueMappings(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requirePIREPValueMappings(w)
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
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPIREPValueMappingsBody+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if len(body) > maxPIREPValueMappingsBody {
		response.WriteError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
		return
	}
	doc, err := pirepvaluemappings.ParseDocument(body)
	if err != nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIG", err.Error())
		return
	}
	view, err := svc.Save(r.Context(), vaID, actor, int64(expected), doc)
	if err != nil {
		writePIREPValueMappingsError(w, vaID, "save", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.FormatInt(view.Revision, 10))
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *Handler) ExportVAPIREPValueMappings(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requirePIREPValueMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	raw, err := svc.ExportBytes(r.Context(), vaID)
	if err != nil {
		writePIREPValueMappingsError(w, vaID, "export", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pirep-value-mappings.json"`)
	_, _ = w.Write(raw)
}

func (h *Handler) GetVAPIREPValueMappingsCatalog(w http.ResponseWriter, r *http.Request, kind string, search string, limit, offset int) {
	svc, ok := h.requirePIREPValueMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	catalogKind := pirepvaluemappings.CatalogAircraft
	if kind == "airline" {
		catalogKind = pirepvaluemappings.CatalogAirline
	}
	items, total, err := svc.CatalogPage(r.Context(), vaID, catalogKind, search, limit, offset)
	if err != nil {
		writePIREPValueMappingsError(w, vaID, "catalog", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"items": items, "total": total}})
}

func (h *Handler) GetVAPIREPValueMappingsCatalogCSV(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requirePIREPValueMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="aircraft-and-liveries.csv"`)
	if err := svc.WriteCatalogCSV(r.Context(), w); err != nil {
		writePIREPValueMappingsError(w, vaID, "catalog_csv", err)
	}
}

func (h *Handler) PreviewVAPIREPValueMappings(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requirePIREPValueMappings(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPIREPValueMappingsBody+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	var in pirepvaluemappings.PreviewInput
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			response.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid preview body")
			return
		}
	}
	if in.Document != nil {
		if err := pirepvaluemappings.ValidateDocument(in.Document.Normalize()); err != nil {
			response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIG", err.Error())
			return
		}
	}
	result, err := svc.Preview(r.Context(), vaID, in)
	if err != nil {
		writePIREPValueMappingsError(w, vaID, "preview", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func writePIREPValueMappingsError(w http.ResponseWriter, vaID, operation string, err error) {
	logPIREPValueMappingsFailure(vaID, operation, err)
	switch {
	case errors.Is(err, pirepvaluemappings.ErrConflict):
		response.WriteError(w, http.StatusPreconditionFailed, "CONFLICT", "configuration was updated elsewhere")
	case errors.Is(err, pirepvaluemappings.ErrInvalid), errors.Is(err, pirepvaluemappings.ErrPayloadLarge):
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIG", err.Error())
	default:
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "operation failed")
	}
}