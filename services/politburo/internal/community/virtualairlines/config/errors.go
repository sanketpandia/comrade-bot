package config

import "errors"

var (
	ErrNotFound      = errors.New("config not found")
	ErrUnknownType   = errors.New("unknown config type")
	ErrInvalidConfig = errors.New("invalid config")
)
