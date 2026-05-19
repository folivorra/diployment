package retry

import (
	"context"
	"time"
)

const (
	DefaultAttempts = 3
	DefaultWait     = 2 * time.Second
)

// WithRetry выполняет fn до attempts раз с экспоненциальным backoff.
func WithRetry(ctx context.Context, attempts int, wait time.Duration, fn func() error) error {
	var err error
	for i := range attempts {
		err = fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait * time.Duration(i+1)):
			}
		}
	}
	return err
}