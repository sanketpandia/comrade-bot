package featureconfig

import "errors"

var (
	ErrInvalidFeature = errors.New("invalid feature")
	ErrInvalidDraft   = errors.New("invalid feature draft")
	ErrConflict       = errors.New("config revision conflict")
	ErrDisabled       = errors.New("feature mapping disabled")
	ErrNotConfigured  = errors.New("feature mapping not configured")
	ErrPilotNotFound  = errors.New("pilot not found")
	ErrAmbiguousPilot = errors.New("ambiguous pilot match")
	ErrStaleConnections = errors.New("connections configuration revision stale")
)