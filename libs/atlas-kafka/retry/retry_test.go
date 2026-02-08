package retry

import (
	"errors"
	"testing"
)

func TestTry_SuccessOnFirstAttempt(t *testing.T) {
	err := Try(func(attempt int) (bool, error) {
		return false, nil
	}, 3)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}

func TestTry_ErrorNoRetry(t *testing.T) {
	expected := errors.New("fatal error")
	err := Try(func(attempt int) (bool, error) {
		return false, expected
	}, 3)
	if err != expected {
		t.Fatalf("Expected %v, got %v", expected, err)
	}
}

func TestTry_MaxRetriesExhausted(t *testing.T) {
	attempts := 0
	err := Try(func(attempt int) (bool, error) {
		attempts++
		return true, errors.New("transient")
	}, 3)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "max retry reached" {
		t.Fatalf("Expected 'max retry reached', got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
}

func TestTry_RetryThenSuccess(t *testing.T) {
	attempts := 0
	err := Try(func(attempt int) (bool, error) {
		attempts++
		if attempt < 3 {
			return true, errors.New("transient")
		}
		return false, nil
	}, 5)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
}
