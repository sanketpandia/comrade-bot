package config

import "errors"

var (
	ErrNotFound      = errors.New("config not found")
	ErrInvalidConfig = errors.New("invalid config")
	ErrConflict      = errors.New("config revision conflict")
)
