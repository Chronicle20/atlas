package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name:   "nil error",
			err:    nil,
			expect: false,
		},
		{
			name:   "connection refused",
			err:    errors.New("connection refused"),
			expect: true,
		},
		{
			name:   "timeout error",
			err:    errors.New("context deadline exceeded"),
			expect: true,
		},
		{
			name:   "kafka error",
			err:    errors.New("kafka producer error"),
			expect: true,
		},
		{
			name:   "database connection error",
			err:    errors.New("driver: bad connection"),
			expect: true,
		},
		{
			name:   "network error",
			err:    errors.New("network unreachable"),
			expect: true,
		},
		{
			name:   "non-transient error",
			err:    errors.New("validation failed"),
			expect: false,
		},
		{
			name:   "business logic error",
			err:    errors.New("proposal already exists"),
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientError(tt.err)
			if result != tt.expect {
				t.Errorf("IsTransientError(%v) = %v, expected %v", tt.err, result, tt.expect)
			}
		})
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		if attempts == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}

	config := DefaultRetryConfig().
		WithMaxRetries(3).
		WithInitialDelay(10 * time.Millisecond)

	err := ExecuteWithRetry(config, operation)
	if err != nil {
		t.Errorf("Expected operation to succeed after retry, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestExecuteWithRetry_MaxRetriesExceeded(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("connection timeout") // Transient error that will be retried
	}

	config := DefaultRetryConfig().
		WithMaxRetries(2).
		WithInitialDelay(10 * time.Millisecond)

	err := ExecuteWithRetry(config, operation)
	if err == nil {
		t.Error("Expected operation to fail after max retries")
	}

	if attempts != 3 { // Initial attempt + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if !strings.Contains(err.Error(), "operation failed after 3 attempts") {
		t.Errorf("Error message should indicate failed attempts, got: %v", err)
	}
}

func TestExecuteWithRetry_NonTransientError(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("validation failed") // Non-transient error
	}

	config := DefaultRetryConfig().
		WithMaxRetries(3).
		WithInitialDelay(10 * time.Millisecond)

	err := ExecuteWithRetry(config, operation)
	if err == nil {
		t.Error("Expected operation to fail immediately for non-transient error")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-transient error, got %d", attempts)
	}
}

func TestExecuteWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	
	attempts := 0
	operation := func() error {
		attempts++
		if attempts == 1 {
			cancel() // Cancel context after first attempt
			return errors.New("temporary failure")
		}
		return nil
	}

	config := DefaultRetryConfig().
		WithContext(ctx).
		WithMaxRetries(3).
		WithInitialDelay(10 * time.Millisecond)

	err := ExecuteWithRetry(config, operation)
	if err == nil {
		t.Error("Expected operation to fail due to context cancellation")
	}

	if !strings.Contains(err.Error(), "operation cancelled") {
		t.Errorf("Error should indicate context cancellation, got: %v", err)
	}
}

func TestExecuteWithRetry_ExponentialBackoff(t *testing.T) {
	attempts := 0
	
	operation := func() error {
		attempts++
		if attempts <= 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	// Custom config to test backoff
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  2.0,
		RetryCondition: IsTransientError,
	}

	start := time.Now()
	err := ExecuteWithRetry(config, operation)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected operation to succeed, got error: %v", err)
	}

	// Should have taken at least: 100ms + 200ms + 400ms = 700ms
	expectedMinDuration := 700 * time.Millisecond
	if elapsed < expectedMinDuration {
		t.Errorf("Expected at least %v elapsed time for backoff, got %v", expectedMinDuration, elapsed)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected InitialDelay=100ms, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("Expected MaxDelay=5s, got %v", config.MaxDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor=2.0, got %f", config.BackoffFactor)
	}

	if config.RetryCondition == nil {
		t.Error("Expected RetryCondition to be set")
	}
}

func TestRetryConfig_FluentInterface(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	config := DefaultRetryConfig().
		WithLogger(logger).
		WithContext(ctx).
		WithMaxRetries(5).
		WithInitialDelay(200 * time.Millisecond).
		WithMaxDelay(10 * time.Second).
		WithBackoffFactor(1.5)

	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries=5, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 200*time.Millisecond {
		t.Errorf("Expected InitialDelay=200ms, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 10*time.Second {
		t.Errorf("Expected MaxDelay=10s, got %v", config.MaxDelay)
	}

	if config.BackoffFactor != 1.5 {
		t.Errorf("Expected BackoffFactor=1.5, got %f", config.BackoffFactor)
	}

	if config.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if config.Context != ctx {
		t.Error("Expected context to be set correctly")
	}
}

// TestLegacyTryFunction tests backward compatibility
func TestLegacyTryFunction(t *testing.T) {
	attempts := 0
	fn := func(attempt int) (bool, error) {
		attempts++
		if attempts == 1 {
			return true, errors.New("temporary failure")
		}
		return false, nil
	}

	err := Try(fn, 3)
	if err != nil {
		t.Errorf("Expected Try to succeed, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}