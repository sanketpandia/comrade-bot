package airtable

import (
	"errors"
	"time"
)

var (
	ErrDenied   = errors.New("airtable access denied")
	ErrThrottle = errors.New("airtable throttled")
	ErrTimeout  = errors.New("airtable timeout")
	ErrUpstream = errors.New("airtable upstream error")
	ErrNotFound = errors.New("airtable not found")
)

// ResponseError is returned for non-success HTTP statuses.
// Code is Airtable's error type when the body has one. It is empty when the
// body is missing or not a short token of letters, digits, '_', '-', or '.'.
type ResponseError struct {
	Status     int
	RetryAfter time.Duration
	Err        error
	Code       string
}

func (e *ResponseError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "airtable response error"
}

func (e *ResponseError) Unwrap() error { return e.Err }
