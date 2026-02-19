package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

type Config struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

func DefaultConfig() Config {
	return Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
}

func (c Config) WithMaxRetries(n int) Config {
	c.MaxRetries = n
	return c
}

func (c Config) WithInitialDelay(d time.Duration) Config {
	c.InitialDelay = d
	return c
}

func (c Config) WithMaxDelay(d time.Duration) Config {
	c.MaxDelay = d
	return c
}

func (c Config) WithBackoffFactor(f float64) Config {
	c.BackoffFactor = f
	return c
}

// Try executes fn with exponential backoff and full jitter. The function fn
// returns (retry, error) — if retry is false, Try stops immediately regardless
// of the error. Context cancellation interrupts retry waits.
func Try(ctx context.Context, cfg Config, fn func(attempt int) (bool, error)) error {
	var lastErr error
	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		cont, err := fn(attempt)
		if !cont || err == nil {
			return err
		}
		lastErr = err

		if attempt == cfg.MaxRetries {
			break
		}

		delay := jitteredDelay(cfg, attempt)
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry interrupted: %w", ctx.Err())
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("after %d attempts, last error: %w", cfg.MaxRetries, lastErr)
}

func jitteredDelay(cfg Config, attempt int) time.Duration {
	calculated := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt-1))
	capped := math.Min(calculated, float64(cfg.MaxDelay))
	return time.Duration(rand.Float64() * capped)
}
