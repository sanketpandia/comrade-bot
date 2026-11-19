package va

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
	"infinite-experiment/politburo/internal/transport/http/response"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func logDataConnectionHTTPFailure(r *http.Request, err error, kind string) {
	if err == nil {
		return
	}
	slog.Warn("data connections request failed",
		"component", "data_connections",
		"request_id", chimiddleware.GetReqID(r.Context()),
		"error_kind", kind,
		"error", err.Error(),
	)
}

// writeConnectionFamilyError maps data-connection and upstream provider errors to HTTP.
func writeConnectionFamilyError(w http.ResponseWriter, r *http.Request, err error) {
	var pe *connections.ProviderError
	if errors.As(err, &pe) {
		slog.Warn("data connection provider request failed",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"operation", pe.Operation,
			"base_id", pe.BaseID,
			"upstream_status", pe.StatusCode,
			"upstream_error", pe.UpstreamCode,
		)
		if errors.Is(pe.Kind, connections.ErrProviderThrottled) {
			if pe.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(pe.RetryAfter.Seconds())))
			}
			response.WriteError(w, http.StatusTooManyRequests, "PROVIDER_THROTTLED", "upstream provider throttled")
			return
		}
		if errors.Is(pe.Kind, connections.ErrProviderDenied) {
			response.WriteError(w, http.StatusUnprocessableEntity, "PROVIDER_CREDENTIAL_DENIED", "upstream provider denied access")
			return
		}
	}
	switch {
	case errors.Is(err, connections.ErrDisabled):
		response.WriteError(w, http.StatusServiceUnavailable, "DATA_CONNECTIONS_DISABLED", err.Error())
	case errors.Is(err, connections.ErrConflict):
		logDataConnectionHTTPFailure(r, err, "revision_conflict")
		response.WriteError(w, http.StatusPreconditionFailed, "REVISION_CONFLICT", err.Error())
	case errors.Is(err, connections.ErrInUse):
		response.WriteError(w, http.StatusConflict, "RESOURCE_IN_USE", err.Error())
	case errors.Is(err, connections.ErrDependency):
		response.WriteError(w, http.StatusConflict, "FEATURE_DEPENDENCY", err.Error())
	case errors.Is(err, connections.ErrNotFound), errors.Is(err, connections.ErrRecordNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, connections.ErrMissingMapping):
		logDataConnectionHTTPFailure(r, err, "missing_mapping")
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIGURATION", err.Error())
	case errors.Is(err, connections.ErrInvalidMapping), errors.Is(err, connections.ErrInvalidFilter), errors.Is(err, connections.ErrInvalidCredential):
		logDataConnectionHTTPFailure(r, err, "invalid_mapping")
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_CONFIGURATION", err.Error())
	case errors.Is(err, connections.ErrSchemaDrift):
		logDataConnectionHTTPFailure(r, err, "schema_drift")
		response.WriteError(w, http.StatusConflict, "SCHEMA_DRIFT", err.Error())
	case errors.Is(err, connections.ErrProviderTimeout):
		response.WriteError(w, http.StatusGatewayTimeout, "PROVIDER_TIMEOUT", err.Error())
	case errors.Is(err, connections.ErrProviderUpstream), errors.Is(err, connections.ErrProviderThrottled):
		response.WriteError(w, http.StatusBadGateway, "PROVIDER_UPSTREAM", "upstream provider error")
	case errors.Is(err, connections.ErrCorruptConfig):
		slog.Error("corrupt data connections document",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"error", err,
		)
		response.WriteError(w, http.StatusInternalServerError, "CONFIG_CORRUPT", "stored configuration is corrupt")
	default:
		slog.Error("data connections request failed",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"error", err,
		)
		response.WriteError(w, http.StatusInternalServerError, "REQUEST_FAILED", "request failed")
	}
}