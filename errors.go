package javdbapi

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidConfig          = errors.New("invalid config")
	ErrInvalidQuery           = errors.New("invalid query")
	ErrNotFound               = errors.New("not found")
	ErrRateLimited            = errors.New("rate limited")
	ErrEmptyResult            = errors.New("empty result")
	ErrParse                  = errors.New("parse failed")
	ErrChallenge              = errors.New("challenge required")
	ErrAuthenticationRequired = errors.New("authentication required")
)

// OpError identifies the failing operation and, when safe, the request URL.
type OpError struct {
	Op  string
	URL string
	Err error
}

func (e *OpError) Error() string {
	if e.URL == "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v (url=%s)", e.Op, e.Err, e.URL)
}

func (e *OpError) Unwrap() error { return e.Err }

// HTTPError reports a non-2xx response status from a site request.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	if e.URL == "" {
		return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
	}
	return fmt.Sprintf("unexpected status code: %d (url=%s)", e.StatusCode, e.URL)
}

// Is only maps the stable 404/429 sentinels; other statuses are inspected via errors.As.
func (e *HTTPError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	default:
		return false
	}
}
