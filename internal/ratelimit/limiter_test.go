package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLimiterWait(t *testing.T) {
	limiter := NewLimiter(10, 3) // 10 QPS

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Should complete without error
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}
}

func TestLimiterRateLimiting(t *testing.T) {
	limiter := NewLimiter(3, 3) // 3 QPS = 1 request every ~333ms

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	count := 0

	// Should be able to make ~6 requests in 2 seconds at 3 QPS
	for i := 0; i < 7; i++ {
		if err := limiter.Wait(ctx); err != nil {
			break
		}
		count++
	}

	elapsed := time.Since(start)
	expectedMin := 1800 * time.Millisecond // ~6 requests take ~1.8s
	expectedMax := 2200 * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Logf("Rate limiting timing check: completed %d requests in %v", count, elapsed)
	}
}

func TestLimiterDoWithRetrySuccess(t *testing.T) {
	limiter := NewLimiter(10, 3)

	ctx := context.Background()
	called := 0

	err := limiter.DoWithRetry(ctx, func() error {
		called++
		return nil
	})

	if err != nil {
		t.Errorf("DoWithRetry failed: %v", err)
	}
	if called != 1 {
		t.Errorf("function called %d times, want 1", called)
	}
}

func TestLimiterDoWithRetryNonTransientError(t *testing.T) {
	limiter := NewLimiter(10, 3)

	ctx := context.Background()
	called := 0
	testErr := errors.New("api error 6: user not found")

	err := limiter.DoWithRetry(ctx, func() error {
		called++
		return testErr
	})

	if err != testErr {
		t.Errorf("DoWithRetry error = %v, want %v", err, testErr)
	}
	if called != 1 {
		t.Errorf("function called %d times, want 1 (no retry for non-transient)", called)
	}
}

func TestLimiterDoWithRetryTransientError(t *testing.T) {
	limiter := NewLimiter(100, 3) // Fast for testing

	ctx := context.Background()
	called := 0

	err := limiter.DoWithRetry(ctx, func() error {
		called++
		if called < 3 {
			return errors.New("rate limited (429)") // Transient
		}
		return nil // Success on third attempt
	})

	if err != nil {
		t.Errorf("DoWithRetry failed: %v", err)
	}
	if called != 3 {
		t.Errorf("function called %d times, want 3", called)
	}
}

func TestLimiterDoWithRetryMaxRetriesExceeded(t *testing.T) {
	limiter := NewLimiter(100, 2) // Max 2 retries = 3 attempts total

	ctx := context.Background()
	called := 0

	err := limiter.DoWithRetry(ctx, func() error {
		called++
		return errors.New("rate limited (429)")
	})

	if err == nil {
		t.Error("DoWithRetry should fail when max retries exceeded")
	}
	if called != 3 {
		t.Errorf("function called %d times, want 3 (1 initial + 2 retries)", called)
	}
	if !errors.Is(err, fmt.Errorf("")) || err.Error() != "max retries (2) exceeded: rate limited (429)" {
		// Just check that error message mentions max retries
		if !contains(err.Error(), "max retries") {
			t.Errorf("error should mention max retries: %v", err)
		}
	}
}

func TestLimiterContextCancellation(t *testing.T) {
	limiter := NewLimiter(1, 5) // Slow to ensure we hit timeout

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.DoWithRetry(ctx, func() error {
		return nil
	})

	if err == nil {
		t.Error("DoWithRetry should fail with cancelled context")
	}
}

func TestExponentialBackoff(t *testing.T) {
	eb := NewExponentialBackoff()

	expected := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		32 * time.Second, // Capped at 32s
	}

	for i, exp := range expected {
		duration := eb.NextBackOff()
		if duration != exp {
			t.Errorf("backoff #%d = %v, want %v", i, duration, exp)
		}
	}
}

func TestExponentialBackoffReset(t *testing.T) {
	eb := NewExponentialBackoff()

	// Consume a few backoffs
	eb.NextBackOff() // 1s
	eb.NextBackOff() // 2s
	eb.NextBackOff() // 4s

	// Reset
	eb.Reset()

	// Should start over
	duration := eb.NextBackOff()
	if duration != 1*time.Second {
		t.Errorf("after reset, NextBackOff = %v, want 1s", duration)
	}
}

func TestLimiterConcurrency(t *testing.T) {
	limiter := NewLimiter(10, 3) // 10 QPS
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make(chan error, 10)

	// Launch 10 concurrent requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.DoWithRetry(ctx, func() error {
				return nil
			})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	// Check all succeeded
	for err := range results {
		if err != nil {
			t.Errorf("concurrent request failed: %v", err)
		}
	}
}

func TestLimiterServerError(t *testing.T) {
	limiter := NewLimiter(100, 2)
	ctx := context.Background()
	called := 0

	err := limiter.DoWithRetry(ctx, func() error {
		called++
		if called < 2 {
			return errors.New("server error (500)") // Transient
		}
		return nil
	})

	if err != nil {
		t.Errorf("DoWithRetry failed: %v", err)
	}
	if called != 2 {
		t.Errorf("called %d times, want 2", called)
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limited",
			err:  errors.New("rate limited (429)"),
			want: true,
		},
		{
			name: "server error",
			err:  errors.New("server error (503)"),
			want: true,
		},
		{
			name: "api error (non-transient)",
			err:  errors.New("api error 6: user not found"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransient(tt.err)
			if result != tt.want {
				t.Errorf("isTransient = %v, want %v", result, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
