package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/time/rate"
)

// HTTPError wraps an error with an optional HTTP response for Retry-After extraction
type HTTPError struct {
	Err      error
	Response *http.Response
}

func (e *HTTPError) Error() string {
	return e.Err.Error()
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

// Limiter enforces rate limiting with exponential backoff retry logic
type Limiter struct {
	limiter     *rate.Limiter
	backoffFunc backoff.BackOff
	maxRetries  int
}

// NewLimiter creates a rate limiter with specified QPS (queries per second)
func NewLimiter(qps float64, maxRetries int) *Limiter {
	return &Limiter{
		limiter:     rate.NewLimiter(rate.Limit(qps), 1),
		backoffFunc: NewExponentialBackoff(),
		maxRetries:  maxRetries,
	}
}

// Wait blocks until a token is available
func (l *Limiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// DoWithRetry executes a function with exponential backoff retry on transient errors
func (l *Limiter) DoWithRetry(ctx context.Context, fn func() error) error {
	l.backoffFunc.Reset()
	attempt := 0

	for {
		// Check context
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context error: %w", err)
		}

		// Wait for rate limiter
		if err := l.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		// Check if error is transient (retryable)
		if !isTransient(err) {
			return err
		}

		// Check max retries
		attempt++
		if attempt > l.maxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", l.maxRetries, err)
		}

		// Try to extract Retry-After header if error includes HTTP response
		var retryAfterDuration time.Duration
		if httpErr, ok := err.(*HTTPError); ok && httpErr.Response != nil {
			if duration, parseErr := ParseRetryAfter(httpErr.Response); parseErr == nil && duration > 0 {
				retryAfterDuration = duration
			}
		}

		// Use Retry-After if available, otherwise use exponential backoff
		var backoffDuration time.Duration
		if retryAfterDuration > 0 {
			backoffDuration = retryAfterDuration
		} else {
			backoffDuration = l.backoffFunc.NextBackOff()
			if backoffDuration == backoff.Stop {
				return fmt.Errorf("backoff exhausted: %w", err)
			}
		}

		// Wait before retrying
		select {
		case <-time.After(backoffDuration):
			// Continue to retry
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// isTransient determines if an error is retryable
func isTransient(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Rate limit errors (429)
	if errMsg == "rate limited (429)" {
		return true
	}

	// Server errors (5xx)
	if errMsg[:len("server error")] == "server error" {
		return true
	}

	// Network errors
	if errMsg == "request failed: context deadline exceeded" ||
		errMsg == "request failed: connection reset" ||
		errMsg == "request failed: connection refused" {
		return true
	}

	return false
}

// ExponentialBackoff provides exponential backoff with jitter
type ExponentialBackoff struct {
	mu              sync.Mutex
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
	currentInterval time.Duration
}

// NewExponentialBackoff creates a backoff strategy: 1s, 2s, 4s, 8s, 16s, 32s (capped)
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		initialInterval: 1 * time.Second,
		maxInterval:     32 * time.Second,
		multiplier:      2.0,
		currentInterval: 1 * time.Second,
	}
}

// NextBackOff returns the next backoff duration
func (eb *ExponentialBackoff) NextBackOff() time.Duration {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	duration := eb.currentInterval

	// Calculate next interval
	nextInterval := time.Duration(float64(eb.currentInterval) * eb.multiplier)
	if nextInterval > eb.maxInterval {
		nextInterval = eb.maxInterval
	}
	eb.currentInterval = nextInterval

	return duration
}

// Reset resets the backoff to initial state
func (eb *ExponentialBackoff) Reset() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.currentInterval = eb.initialInterval
}

// ParseRetryAfter parses the Retry-After header from an HTTP response.
// It supports both formats: seconds (integer) and HTTP date (RFC1123).
// Returns 0 duration if header is missing or invalid.
func ParseRetryAfter(resp *http.Response) (time.Duration, error) {
	if resp == nil {
		return 0, nil
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0, nil
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("negative Retry-After value: %d", seconds)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	// Try parsing as HTTP date (RFC1123)
	retryTime, err := time.Parse(time.RFC1123, retryAfter)
	if err != nil {
		return 0, fmt.Errorf("invalid Retry-After format: %w", err)
	}

	duration := time.Until(retryTime)
	if duration < 0 {
		// Date is in the past, return 0
		return 0, nil
	}

	return duration, nil
}
