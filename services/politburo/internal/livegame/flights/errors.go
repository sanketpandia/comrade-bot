package flights

import "errors"

var (
	ErrCacheMiss      = errors.New("active flights cache miss")
	ErrCacheCorrupt   = errors.New("active flights cache corrupt")
	ErrCacheRead      = errors.New("active flights cache read failed")
	ErrUnknownServer  = errors.New("unknown normalized server name")
	ErrInvalidFilter  = errors.New("invalid filter value")
	ErrSessionNames   = errors.New("session names cache unavailable")
)
