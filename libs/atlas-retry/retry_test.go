package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTry_SuccessOnFirstAttempt(t *testing.T) {
	err := Try(context.Background(), DefaultConfig(), func(attempt int) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}

func TestTry_ErrorNoRetry(t *testing.T) {
	expected := errors.New("fatal error")
	err := Try(context.Background(), DefaultConfig(), func(attempt int) (bool, error) {
		return false, expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("Expected %v, got %v", expected, err)
	}
}

func TestTry_MaxRetriesExhausted(t *testing.T) {
	attempts := 0
	cfg := DefaultConfig().WithMaxRetries(3).WithInitialDelay(time.Millisecond).WithMaxDelay(time.Millisecond)
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		attempts++
		return true, errors.New("transient")
	})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, errors.New("")) {
		// just verify it's not nil — wrapping is tested separately
	}
}

func TestTry_WrapsOriginalError(t *testing.T) {
	original := errors.New("connection refused")
	cfg := DefaultConfig().WithMaxRetries(2).WithInitialDelay(time.Millisecond).WithMaxDelay(time.Millisecond)
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		return true, original
	})
	if !errors.Is(err, original) {
		t.Fatalf("Expected wrapped error to contain original, got %v", err)
	}
}

func TestTry_RetryThenSuccess(t *testing.T) {
	attempts := 0
	cfg := DefaultConfig().WithMaxRetries(5).WithInitialDelay(time.Millisecond).WithMaxDelay(time.Millisecond)
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		attempts++
		if attempt < 3 {
			return true, errors.New("transient")
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
}

func TestTry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := DefaultConfig().WithMaxRetries(10).WithInitialDelay(time.Second).WithMaxDelay(time.Second)
	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := Try(ctx, cfg, func(attempt int) (bool, error) {
		attempts++
		return true, errors.New("transient")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
	if attempts < 1 {
		t.Fatal("Expected at least 1 attempt")
	}
}

func TestTry_ExponentialBackoffTiming(t *testing.T) {
	cfg := Config{
		MaxRetries:    4,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
	// With full jitter, delays are random in [0, calculated).
	// Max total delay: 100ms + 200ms + 400ms = 700ms (if jitter always picks max).
	// Should complete faster than 700ms on average.
	start := time.Now()
	Try(context.Background(), cfg, func(attempt int) (bool, error) {
		return true, errors.New("fail")
	})
	elapsed := time.Since(start)
	// Must complete within a generous upper bound (no jitter = 700ms, so 1s is safe)
	if elapsed > 1*time.Second {
		t.Fatalf("Took too long: %v", elapsed)
	}
}

func TestTry_MaxDelayCap(t *testing.T) {
	cfg := Config{
		MaxRetries:    3,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond, // cap below initial
		BackoffFactor: 2.0,
	}
	start := time.Now()
	Try(context.Background(), cfg, func(attempt int) (bool, error) {
		return true, errors.New("fail")
	})
	elapsed := time.Since(start)
	// 2 sleeps, each capped at [0, 100ms). Should be well under 300ms.
	if elapsed > 300*time.Millisecond {
		t.Fatalf("Max delay cap not working: %v", elapsed)
	}
}

func TestTry_SingleAttemptNoSleep(t *testing.T) {
	cfg := DefaultConfig().WithMaxRetries(1)
	start := time.Now()
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		return true, errors.New("fail")
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Expected error")
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("Single attempt should not sleep, took %v", elapsed)
	}
}

func TestJitteredDelay(t *testing.T) {
	cfg := Config{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
	for i := 0; i < 100; i++ {
		d := jitteredDelay(cfg, 1)
		if d < 0 || d > cfg.InitialDelay {
			t.Fatalf("Jittered delay out of range for attempt 1: %v", d)
		}
	}
	for i := 0; i < 100; i++ {
		d := jitteredDelay(cfg, 3)
		expected := time.Duration(float64(cfg.InitialDelay) * 4) // 2^2 = 4
		if d < 0 || d > expected {
			t.Fatalf("Jittered delay out of range for attempt 3: %v (max %v)", d, expected)
		}
	}
}
