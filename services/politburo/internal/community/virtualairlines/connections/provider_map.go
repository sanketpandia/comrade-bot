package connections

import (
	"context"
	"errors"
	"net/http"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func mapAirtableError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var resp *airtable.ResponseError
	if errors.As(err, &resp) {
		switch {
		case resp.Status == http.StatusUnprocessableEntity:
			return ErrInvalidFilter
		case errors.Is(resp.Err, airtable.ErrThrottle):
			return &ProviderError{Kind: ErrProviderThrottled, StatusCode: resp.Status, RetryAfter: resp.RetryAfter, UpstreamCode: resp.Code}
		case errors.Is(resp.Err, airtable.ErrDenied):
			return &ProviderError{Kind: ErrProviderDenied, StatusCode: resp.Status, UpstreamCode: resp.Code}
		case errors.Is(resp.Err, airtable.ErrNotFound):
			return ErrRecordNotFound
		case errors.Is(resp.Err, airtable.ErrTimeout):
			return ErrProviderTimeout
		default:
			return &ProviderError{Kind: ErrProviderUpstream, StatusCode: resp.Status, UpstreamCode: resp.Code}
		}
	}
	if errors.Is(err, airtable.ErrTimeout) {
		return ErrProviderTimeout
	}
	return ErrProviderUpstream
}
