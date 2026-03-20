package sdkmgr

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryOpts configures the retry behaviour.
type RetryOpts struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	// ShouldRetry decides whether an error is retryable. Defaults to always-retry.
	ShouldRetry func(error) bool
}

// DefaultRetryOpts returns sane defaults (3 attempts, 1s initial, 30s max).
func DefaultRetryOpts() RetryOpts {
	return RetryOpts{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
	}
}

// WithRetry executes fn with exponential backoff + jitter.
// It returns the result of the first successful call or the last error.
func WithRetry[T any](ctx context.Context, opts RetryOpts, fn func(ctx context.Context) (T, error)) (T, error) {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.InitialDelay <= 0 {
		opts.InitialDelay = time.Second
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 30 * time.Second
	}

	var lastErr error
	var zero T

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if opts.ShouldRetry != nil && !opts.ShouldRetry(err) {
			return zero, err
		}

		if attempt < opts.MaxAttempts-1 {
			delay := float64(opts.InitialDelay) * math.Pow(2, float64(attempt))
			if delay > float64(opts.MaxDelay) {
				delay = float64(opts.MaxDelay)
			}
			// Add jitter: ±25%.
			jitter := delay * 0.25 * (rand.Float64()*2 - 1)
			sleepDur := time.Duration(delay + jitter)

			logrus.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"max":     opts.MaxAttempts,
				"delay":   sleepDur.String(),
				"error":   err.Error(),
			}).Warn("retrying after error")

			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(sleepDur):
			}
		}
	}

	return zero, lastErr
}
