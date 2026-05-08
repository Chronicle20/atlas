package wz

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

// makeCanvasBlob builds a synthetic WZ canvas blob region in a temp file.
// The blob at each position is: [flagByte] [payload...].
// ReadCanvasData skips the flag byte and reads payload bytes.
// Returns the open *os.File and the slice of expected payloads indexed by blob number.
func makeCanvasTestFile(t *testing.T, blobCount int, blobPayloadSize int) (*os.File, [][]byte) {
	t.Helper()

	// Each blob: 1 flag byte + blobPayloadSize payload bytes.
	blobStride := 1 + blobPayloadSize
	data := make([]byte, blobCount*blobStride)
	payloads := make([][]byte, blobCount)

	for i := 0; i < blobCount; i++ {
		base := i * blobStride
		data[base] = 0xAB // flag byte (skipped by ReadCanvasData)
		payload := make([]byte, blobPayloadSize)
		for j := range payload {
			payload[j] = byte((i*blobPayloadSize + j) & 0xFF)
		}
		copy(data[base+1:], payload)
		payloads[i] = payload
	}

	f, err := os.CreateTemp("", "wz_file_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	return f, payloads
}

// TestReadCanvasDataConcurrentSafe spawns 16 goroutines that call
// ReadCanvasData concurrently against distinct offsets and verifies each
// goroutine receives exactly its own payload — not someone else's.
//
// Run with -race to confirm there is no data race on the file seek pointer.
func TestReadCanvasDataConcurrentSafe(t *testing.T) {
	const goroutines = 16
	const blobPayloadSize = 64

	f, payloads := makeCanvasTestFile(t, goroutines, blobPayloadSize)

	// Build a minimal wz.File that wraps the temp file.
	// We only exercise ReadCanvasData, so header fields are zero-valued.
	wzFile := &File{
		f:      f,
		reader: NewReader(f),
	}

	blobStride := int64(1 + blobPayloadSize)
	dataSize := int32(blobPayloadSize + 1) // size includes the flag byte

	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		offset := int64(i) * blobStride
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := wzFile.ReadCanvasData(offset, dataSize)
			if err != nil {
				errs[i] = fmt.Errorf("goroutine %d ReadCanvasData: %w", i, err)
				return
			}
			if !bytes.Equal(got, payloads[i]) {
				errs[i] = fmt.Errorf("goroutine %d: got %v, want %v", i, got, payloads[i])
			}
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
}

// TestReadCanvasDataSizeLeOne verifies the early-exit for size <= 1.
func TestReadCanvasDataSizeLeOne(t *testing.T) {
	f, _ := makeCanvasTestFile(t, 1, 8)
	wzFile := &File{f: f, reader: NewReader(f)}

	got, err := wzFile.ReadCanvasData(0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for size=1, got %v", got)
	}

	got, err = wzFile.ReadCanvasData(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for size=0, got %v", got)
	}
}
