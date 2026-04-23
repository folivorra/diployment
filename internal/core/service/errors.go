package service

import "errors"

var (
	ErrCodeExchange = errors.New("code exchange failed")
	ErrProviderAPI  = errors.New("provider API error")
)
