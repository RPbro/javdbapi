package javdbapi

import "errors"

var (
	ErrInvalidConfig    = errors.New("invalid config")
	ErrInvalidQuery     = errors.New("invalid query")
	ErrUnexpectedStatus = errors.New("unexpected status")
	ErrEmptyResult      = errors.New("empty result")
)
