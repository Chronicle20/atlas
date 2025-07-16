package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type RepeatableFunc func(attempt int) (retry bool, err error)

type RepeatableFuncWithResponse func(attempt int) (bool, interface{}, error)

// RetryConfig defines configuration for retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Logger          logrus.FieldLogger
	Context         context.Context
	RetryCondition  func(error) bool
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		BackoffFactor:  2.0,
		RetryCondition: IsTransientError,
	}
}

// WithLogger sets the logger for retry operations
func (c *RetryConfig) WithLogger(logger logrus.FieldLogger) *RetryConfig {
	c.Logger = logger
	return c
}

// WithContext sets the context for retry operations
func (c *RetryConfig) WithContext(ctx context.Context) *RetryConfig {
	c.Context = ctx
	return c
}

// WithMaxRetries sets the maximum number of retry attempts
func (c *RetryConfig) WithMaxRetries(maxRetries int) *RetryConfig {
	c.MaxRetries = maxRetries
	return c
}

// WithInitialDelay sets the initial delay between retries
func (c *RetryConfig) WithInitialDelay(delay time.Duration) *RetryConfig {
	c.InitialDelay = delay
	return c
}

// WithMaxDelay sets the maximum delay between retries
func (c *RetryConfig) WithMaxDelay(delay time.Duration) *RetryConfig {
	c.MaxDelay = delay
	return c
}

// WithBackoffFactor sets the exponential backoff factor
func (c *RetryConfig) WithBackoffFactor(factor float64) *RetryConfig {
	c.BackoffFactor = factor
	return c
}

// WithRetryCondition sets a custom condition for determining if an error should trigger a retry
func (c *RetryConfig) WithRetryCondition(condition func(error) bool) *RetryConfig {
	c.RetryCondition = condition
	return c
}

// IsTransientError determines if an error is likely transient and worth retrying
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	
	// Common transient error patterns
	errStr := err.Error()
	transientPatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"connection reset",
		"broken pipe",
		"context deadline exceeded",
		"driver: bad connection",
		"kafka",
	}
	
	for _, pattern := range transientPatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}
	
	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		   (len(s) > len(substr) && 
		    len(s) >= len(substr) && 
		    contains(toLower(s), toLower(substr))))
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, r := range []byte(s) {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ExecuteWithRetry executes a function with retry logic according to the configuration
func ExecuteWithRetry(config *RetryConfig, operation func() error) error {
	if config == nil {
		config = DefaultRetryConfig()
	}
	
	var lastErr error
	delay := config.InitialDelay
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context cancellation
		if config.Context != nil {
			select {
			case <-config.Context.Done():
				return fmt.Errorf("operation cancelled: %w", config.Context.Err())
			default:
			}
		}
		
		// Execute the operation
		err := operation()
		if err == nil {
			if config.Logger != nil && attempt > 0 {
				config.Logger.WithField("attempts", attempt+1).Info("Operation succeeded after retry")
			}
			return nil
		}
		
		lastErr = err
		
		// Check if we should retry this error
		if config.RetryCondition != nil && !config.RetryCondition(err) {
			if config.Logger != nil {
				config.Logger.WithError(err).Debug("Error is not retryable, aborting")
			}
			return err
		}
		
		// Don't sleep after the last attempt
		if attempt == config.MaxRetries {
			break
		}
		
		if config.Logger != nil {
			config.Logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err,
				"delay":   delay,
			}).Warn("Operation failed, retrying")
		}
		
		// Sleep with exponential backoff
		if config.Context != nil {
			select {
			case <-config.Context.Done():
				return fmt.Errorf("operation cancelled during retry delay: %w", config.Context.Err())
			case <-time.After(delay):
			}
		} else {
			time.Sleep(delay)
		}
		
		// Calculate next delay with exponential backoff
		nextDelay := time.Duration(float64(delay) * config.BackoffFactor)
		if nextDelay > config.MaxDelay {
			delay = config.MaxDelay
		} else {
			delay = nextDelay
		}
	}
	
	if config.Logger != nil {
		config.Logger.WithFields(logrus.Fields{
			"attempts": config.MaxRetries + 1,
			"error":    lastErr,
		}).Error("Operation failed after all retry attempts")
	}
	
	return fmt.Errorf("operation failed after %d attempts, last error: %w", config.MaxRetries+1, lastErr)
}

// Legacy functions for backward compatibility
func Try(fn RepeatableFunc, retries int) error {
	attempt := 1
	for {
		cont, err := fn(attempt)
		if !cont || err == nil {
			return nil
		}
		attempt++
		if attempt > retries {
			return errors.New("max retry reached")
		}
		time.Sleep(1 * time.Second)
	}
}

