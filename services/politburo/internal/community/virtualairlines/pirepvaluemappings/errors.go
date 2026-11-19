package pirepvaluemappings

import "errors"

var (
	ErrNotFound      = errors.New("pirep value mappings not found")
	ErrConflict      = errors.New("pirep value mappings revision conflict")
	ErrInvalid       = errors.New("invalid pirep value mappings")
	ErrPayloadLarge  = errors.New("payload too large")
)