package service

import "errors"

var (
	ErrCodeExchange   = errors.New("code exchange failed")
	ErrProviderAPI    = errors.New("provider API error")
	ErrForbidden      = errors.New("access denied")
	ErrLogUnavailable = errors.New("log unavailable")
)
