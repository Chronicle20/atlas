package characterrender

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWritePNGProducesFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "abc.png")
	data := []byte("\x89PNG\r\n\x1a\nfake")

	if err := AtomicWritePNG(target, bytes.NewReader(data)); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch")
	}
}

func TestAtomicWritePNGConcurrent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "abc.png")
	data := []byte("\x89PNG\r\n\x1a\nfake-concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := AtomicWritePNG(target, bytes.NewReader(data)); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	wg.Wait()

	f, err := os.Open(target)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch after concurrent writes")
	}
}
