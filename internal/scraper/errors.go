package scraper

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrRateLimited = errors.New("rate limited")
)

type ParseError struct {
	Context string
	Err     error
}

func (e *ParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parse error (%s): %v", e.Context, e.Err)
	}
	return fmt.Sprintf("parse error: %s", e.Context)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

type NetworkError struct {
	URL        string
	StatusCode int
	Err        error
}

func (e *NetworkError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("network error: %s returned status %d", e.URL, e.StatusCode)
	}
	if e.Err != nil {
		return fmt.Sprintf("network error: %s: %v", e.URL, e.Err)
	}
	return fmt.Sprintf("network error: %s", e.URL)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

func IsNetworkError(err error) bool {
	var netErr *NetworkError
	return errors.As(err, &netErr)
}

func IsParseError(err error) bool {
	var parseErr *ParseError
	return errors.As(err, &parseErr)
}
