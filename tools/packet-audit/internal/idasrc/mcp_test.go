package idasrc

import (
	"context"
	"errors"
	"testing"
)

func TestMCPSourceWithoutClient(t *testing.T) {
	src := NewMCPSource(nil)
	_, err := src.Resolve(context.Background(), "any")
	if !errors.Is(err, ErrMCPUnavailable) {
		t.Errorf("expected ErrMCPUnavailable, got %v", err)
	}
}
