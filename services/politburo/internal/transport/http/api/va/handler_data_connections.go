package va

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
	"infinite-experiment/politburo/internal/transport/http/response"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const maxDataConnectionsBodyBytes = 64 << 10

type dataConnections interface {
	Enabled() bool
	GetView(ctx context.Context, vaID string) (connections.StoredView, error)
	CreateConnection(ctx context.Context, vaID, actor string, expectedRevision int, in connections.CreateConnectionInput) (connections.StoredView, error)
	UpdateConnection(ctx context.Context, vaID, actor, connectionID string, expectedRevision int, in connections.UpdateConnectionInput) (connections.StoredView, error)
	DeleteConnection(ctx context.Context, vaID, actor, connectionID string, expectedRevision int) (connections.StoredView, error)
	AddBase(ctx context.Context, vaID, actor, connectionID string, expectedRevision int, in connections.AddBaseInput) (connections.StoredView, error)
	UpdateBase(ctx context.Context, vaID, actor, connectionID, baseSetupID string, expectedRevision int, in connections.UpdateBaseInput) (connections.StoredView, error)
	DeleteBase(ctx context.Context, vaID, actor, connectionID, baseSetupID string, expectedRevision int) (connections.StoredView, error)
	SaveMapping(ctx context.Context, vaID, actor, entity string, expectedRevision int, mapping connections.Mapping) (connections.StoredView, error)
	ClearMapping(ctx context.Context, vaID, actor, entity string, expectedRevision int) (connections.StoredView, error)
	Preview(ctx context.Context, vaID, entity string, mapping connections.Mapping) ([]connections.MappedRecord, error)
	Inspect(ctx context.Context, vaID, connectionID, baseSetupID string) (connections.InspectResult, error)
}

func (h *Handler) writeDataConnectionsDisabled(w http.ResponseWriter) {
	response.WriteError(w, http.StatusServiceUnavailable, "DATA_CONNECTIONS_DISABLED", "data connections are not configured on this deployment")
}

func (h *Handler) requireDataConnections(w http.ResponseWriter) (dataConnections, bool) {
	if h.dataConnections == nil || !h.dataConnections.Enabled() {
		h.writeDataConnectionsDisabled(w)
		return nil, false
	}
	return h.dataConnections, true
}

func parseIfMatch(r *http.Request, required bool) (int, bool) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" {
		return 0, !required
	}
	raw = strings.Trim(raw, "\"")
	rev, err := strconv.Atoi(raw)
	if err != nil || rev < 0 {
		return 0, false
	}
	return rev, true
}

func (h *Handler) GetVADataConnections(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requireDataConnections(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	view, err := svc.GetView(r.Context(), vaID)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) CreateVADataConnection(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.requireDataConnections(w)
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
	var body struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Token    string `json:"token"`
		BaseName string `json:"baseName"`
		BaseID   string `json:"baseId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	view, err := svc.CreateConnection(r.Context(), vaID, actor, expected, connections.CreateConnectionInput{
		Name: body.Name, Provider: body.Provider, Token: body.Token, BaseName: body.BaseName, BaseID: body.BaseID,
	})
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusCreated, view)
}

func (h *Handler) PatchVADataConnection(w http.ResponseWriter, r *http.Request, connectionID string) {
	svc, ok := h.requireDataConnections(w)
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
	var body struct {
		Name  *string `json:"name"`
		Token *string `json:"token"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	view, err := svc.UpdateConnection(r.Context(), vaID, actor, connectionID, expected, connections.UpdateConnectionInput{Name: body.Name, Token: body.Token})
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) DeleteVADataConnection(w http.ResponseWriter, r *http.Request, connectionID string) {
	svc, ok := h.requireDataConnections(w)
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
	view, err := svc.DeleteConnection(r.Context(), vaID, actor, connectionID, expected)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) AddVADataConnectionBase(w http.ResponseWriter, r *http.Request, connectionID string) {
	svc, ok := h.requireDataConnections(w)
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
	var body struct {
		Name   string `json:"name"`
		BaseID string `json:"baseId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	view, err := svc.AddBase(r.Context(), vaID, actor, connectionID, expected, connections.AddBaseInput{Name: body.Name, BaseID: body.BaseID})
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) PatchVADataConnectionBase(w http.ResponseWriter, r *http.Request, connectionID, baseID string) {
	svc, ok := h.requireDataConnections(w)
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
	var body struct {
		Name   *string `json:"name"`
		BaseID *string `json:"baseId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	view, err := svc.UpdateBase(r.Context(), vaID, actor, connectionID, baseID, expected, connections.UpdateBaseInput{Name: body.Name, BaseID: body.BaseID})
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) DeleteVADataConnectionBase(w http.ResponseWriter, r *http.Request, connectionID, baseID string) {
	svc, ok := h.requireDataConnections(w)
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
	view, err := svc.DeleteBase(r.Context(), vaID, actor, connectionID, baseID, expected)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) GetVADataConnectionSchema(w http.ResponseWriter, r *http.Request, connectionID, baseID string) {
	svc, ok := h.requireDataConnections(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	result, err := svc.Inspect(r.Context(), vaID, connectionID, baseID)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *Handler) PutVADataConnectionMapping(w http.ResponseWriter, r *http.Request, entity string) {
	svc, ok := h.requireDataConnections(w)
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
	var mapping connections.Mapping
	if err := decodeJSON(r, &mapping); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	view, err := svc.SaveMapping(r.Context(), vaID, actor, entity, expected, mapping)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) DeleteVADataConnectionMapping(w http.ResponseWriter, r *http.Request, entity string) {
	svc, ok := h.requireDataConnections(w)
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
	view, err := svc.ClearMapping(r.Context(), vaID, actor, entity, expected)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", strconv.Itoa(view.Revision))
	writeDataConnectionsView(w, http.StatusOK, view)
}

func (h *Handler) PreviewVADataConnectionMapping(w http.ResponseWriter, r *http.Request, entity string) {
	svc, ok := h.requireDataConnections(w)
	if !ok {
		return
	}
	vaID, ok := h.vaContext(w, r)
	if !ok {
		return
	}
	var mapping connections.Mapping
	if err := decodeJSON(r, &mapping); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	records, err := svc.Preview(r.Context(), vaID, entity, mapping)
	if err != nil {
		writeDataConnectionsError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"records": records}})
}

func decodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxDataConnectionsBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return errors.New("invalid request body")
	}
	if len(data) == 0 {
		return errors.New("request body required")
	}
	if len(data) > maxDataConnectionsBodyBytes {
		return errors.New("request body too large")
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return errors.New("invalid JSON")
	}
	return nil
}

func writeDataConnectionsView(w http.ResponseWriter, status int, view connections.StoredView) {
	payload := map[string]any{
		"revision":    view.Revision,
		"connections": view.Connections,
		"mappings":    view.Mappings,
		"catalog":     view.Catalog,
	}
	if !view.UpdatedAt.IsZero() {
		payload["updatedAt"] = view.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if view.UpdatedBy != "" {
		payload["updatedBy"] = view.UpdatedBy
	}
	response.WriteJSON(w, status, map[string]any{"data": payload})
}

func writeDataConnectionsError(w http.ResponseWriter, r *http.Request, err error) {
	var pe *connections.ProviderError
	if errors.As(err, &pe) {
		if errors.Is(pe.Kind, connections.ErrProviderThrottled) {
			if pe.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(pe.RetryAfter.Seconds())))
			}
			response.WriteError(w, http.StatusTooManyRequests, "PROVIDER_THROTTLED", "upstream provider throttled")
			return
		}
		if errors.Is(pe.Kind, connections.ErrProviderDenied) {
			response.WriteError(w, http.StatusForbidden, "PROVIDER_DENIED", "upstream provider denied access")
			return
		}
	}
	switch {
	case errors.Is(err, connections.ErrDisabled):
		response.WriteError(w, http.StatusServiceUnavailable, "DATA_CONNECTIONS_DISABLED", err.Error())
	case errors.Is(err, connections.ErrConflict):
		response.WriteError(w, http.StatusPreconditionFailed, "REVISION_CONFLICT", err.Error())
	case errors.Is(err, connections.ErrInUse):
		response.WriteError(w, http.StatusConflict, "RESOURCE_IN_USE", err.Error())
	case errors.Is(err, connections.ErrNotFound), errors.Is(err, connections.ErrRecordNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, connections.ErrInvalidMapping), errors.Is(err, connections.ErrInvalidFilter), errors.Is(err, connections.ErrInvalidCredential):
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIGURATION", err.Error())
	case errors.Is(err, connections.ErrSchemaDrift):
		response.WriteError(w, http.StatusConflict, "SCHEMA_DRIFT", err.Error())
	case errors.Is(err, connections.ErrProviderTimeout):
		response.WriteError(w, http.StatusGatewayTimeout, "PROVIDER_TIMEOUT", err.Error())
	case errors.Is(err, connections.ErrProviderUpstream), errors.Is(err, connections.ErrProviderThrottled):
		response.WriteError(w, http.StatusBadGateway, "PROVIDER_UPSTREAM", "upstream provider error")
	default:
		slog.Error("data connections request failed",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"error", err,
		)
		response.WriteError(w, http.StatusInternalServerError, "REQUEST_FAILED", "request failed")
	}
}