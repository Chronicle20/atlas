package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTry_SuccessOnFirstAttempt(t *testing.T) {
	cfg := DefaultConfig().WithMaxRetries(3).WithInitialDelay(time.Millisecond)
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}

func TestTry_ErrorNoRetry(t *testing.T) {
	expected := errors.New("fatal error")
	cfg := DefaultConfig().WithMaxRetries(3).WithInitialDelay(time.Millisecond)
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
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

func TestLegacyTry_BackwardCompat(t *testing.T) {
	attempts := 0
	err := LegacyTry(func(attempt int) (bool, error) {
		attempts++
		if attempt < 2 {
			return true, errors.New("transient")
		}
		return false, nil
	}, 3)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("Expected 2 attempts, got %d", attempts)
	}
}
