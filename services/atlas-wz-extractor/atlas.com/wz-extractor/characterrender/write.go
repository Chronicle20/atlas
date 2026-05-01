package characterrender

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicWritePNG writes `r` to `dst` such that no reader ever observes
// partial bytes. Concurrent writes for the same `dst` produce identical
// results when their inputs are identical (last-rename wins).
func AtomicWritePNG(dst string, r io.Reader) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync: %w", err)
	}
	// os.CreateTemp creates files with mode 0600 — atlas-assets nginx runs as
	// a non-root user and serves the cached PNGs directly from the PVC, so
	// the file must be world-readable.
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
