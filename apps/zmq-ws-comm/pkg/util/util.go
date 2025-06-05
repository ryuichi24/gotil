package util

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for the retry logic.
type RetryConfig struct {
	MaxAttempts  int           // Maximum number of attempts
	InitialDelay time.Duration // Initial delay between attempts
	MaxDelay     time.Duration // Maximum delay between attempts
}

// Retry executes the provided operation with retry logic.
func Retry[T any](ctx context.Context, config RetryConfig, operation func() (T, error)) (T, error) {
	var result T
	var err error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		result, err = operation()
		if err == nil {
			return result, nil
		}

		if attempt == config.MaxAttempts {
			break
		}

		// Apply jitter to delay
		jitter := time.Duration(rand.Int63n(int64(delay)))
		sleep := delay + jitter
		if sleep > config.MaxDelay {
			sleep = config.MaxDelay
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(sleep):
		}

		// Exponential backoff
		delay *= 2
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return result, fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, err)
}
