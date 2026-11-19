package connections

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDisabled          = errors.New("data connections capability disabled")
	ErrConflict          = errors.New("configuration revision conflict")
	ErrInUse             = errors.New("resource is in use")
	ErrNotFound          = errors.New("not found")
	ErrMissingMapping    = errors.New("entity mapping not configured")
	ErrInvalidFilter     = errors.New("invalid filter")
	ErrSchemaDrift       = errors.New("schema drift")
	ErrRecordNotFound    = errors.New("record not found")
	ErrProviderDenied    = errors.New("provider access denied")
	ErrProviderThrottled = errors.New("provider throttled")
	ErrProviderTimeout   = errors.New("provider timeout")
	ErrProviderUpstream  = errors.New("provider upstream error")
	ErrInvalidMapping    = errors.New("invalid mapping")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrInvalidCursor     = errors.New("invalid pagination cursor")
	ErrBaseIDImmutable   = errors.New("physical base id cannot change")
)

// ProviderError carries sanitized upstream failure details for logs and HTTP mapping.
type ProviderError struct {
	Kind         error
	StatusCode   int
	RetryAfter   time.Duration
	Operation    string
	BaseID       string
	UpstreamCode string
}

func (e *ProviderError) Error() string {
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "provider error"
}

func (e *ProviderError) Unwrap() error { return e.Kind }

func providerOutcome(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "canceled"
	}
	switch {
	case errors.Is(err, ErrProviderThrottled):
		return "throttled"
	case errors.Is(err, ErrProviderTimeout):
		return "timeout"
	case errors.Is(err, ErrProviderDenied):
		return "denied"
	default:
		var pe *ProviderError
		if errors.As(err, &pe) && pe.StatusCode >= 400 && pe.StatusCode < 500 {
			return "client_error"
		}
	}
	return "server_error"
}

func readOutcome(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "canceled"
	}
	switch {
	case errors.Is(err, ErrMissingMapping):
		return "missing_mapping"
	case errors.Is(err, ErrInvalidFilter):
		return "invalid_filter"
	case errors.Is(err, ErrSchemaDrift), errors.Is(err, ErrInvalidMapping):
		return "schema_drift"
	case errors.Is(err, ErrRecordNotFound):
		return "not_found"
	case errors.Is(err, ErrProviderDenied):
		return "denied"
	case errors.Is(err, ErrProviderThrottled):
		return "throttled"
	case errors.Is(err, ErrProviderTimeout):
		return "timeout"
	case errors.Is(err, ErrProviderUpstream), errors.Is(err, ErrInvalidCredential):
		return "upstream"
	default:
		var pe *ProviderError
		if errors.As(err, &pe) {
			return readOutcome(pe.Kind)
		}
	}
	return "upstream"
}
