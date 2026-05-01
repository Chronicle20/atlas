package characterimage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStanceUnknown(t *testing.T) {
	if err := ValidateStance("warp"); !errors.Is(err, ErrInvalidStance) {
		t.Fatalf("got %v, want ErrInvalidStance", err)
	}
}

func TestValidateStanceKnown(t *testing.T) {
	for _, s := range []string{"stand1", "stand2", "walk1", "alert", "jump"} {
		if err := ValidateStance(s); err != nil {
			t.Fatalf("ValidateStance(%q) = %v", s, err)
		}
	}
}

func TestValidateFrameOutOfRange(t *testing.T) {
	dir := t.TempDir()
	frameDir := filepath.Join(dir, "character-parts", "00002000", "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Only frame 0 exists. Asking for frame 1 must fail.
	if err := ValidateFrame(dir, "00002000", "stand1", 1); !errors.Is(err, ErrFrameOutOfRange) {
		t.Fatalf("got %v, want ErrFrameOutOfRange", err)
	}
	if err := ValidateFrame(dir, "00002000", "stand1", 0); err != nil {
		t.Fatalf("ValidateFrame frame 0: %v", err)
	}
}
