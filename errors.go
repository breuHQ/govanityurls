package main

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig       = errors.New("invalid config")
	ErrCacheMaxAgeNegative = errors.New("cache-max-age must be positive")
	ErrHTTPHostMissing     = errors.New("host is required")
	ErrUnableToRender      = errors.New("error rendering HTTP response")
)

type (
	InvalidVCSError struct {
		path string
		repo string
	}
)

func (e *InvalidVCSError) Error() string {
	return fmt.Sprintf("configuration for %v: cannot infer VCS from %s", e.path, e.repo)
}

func NewInvalidVCSError(path, repo string) error {
	return &InvalidVCSError{path, repo}
}
