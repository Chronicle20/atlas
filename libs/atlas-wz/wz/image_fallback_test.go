package wz

import (
	"bytes"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// mixedArchive mimics JMS v185: file-level None with one KMS-encrypted image.
func mixedArchive(payload []byte) *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(185).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("Plain", wztest.Str("name", "open"))).
		AddImage(wztest.ImgWithKey("Secret", crypto.EncryptionKMS,
			wztest.Str("name", "hidden"),
			wztest.Canvas("icon", payload),
		))
}

func imageByName(t *testing.T, f *File, name string) *Image {
	t.Helper()
	for _, img := range f.Root().Images() {
		if img.Name() == name {
			return img
		}
	}
	t.Fatalf("image %q not found", name)
	return nil
}

// TestPerImageKeyFallback: the KMS image inside a None file must parse via
// the fallback path with correct string values; the plain image is untouched.
func TestPerImageKeyFallback(t *testing.T) {
	path := writeFixture(t, mixedArchive([]byte{9, 9, 9}), "String.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if !f.EncryptionKey().IsEmpty() {
		t.Fatalf("file-level key should be None")
	}

	plainProps, err := imageByName(t, f, "Plain").Properties()
	if err != nil {
		t.Fatalf("plain properties: %v", err)
	}
	if s := plainProps[0].(*property.StringProperty); s.Value() != "open" {
		t.Fatalf("plain name = %q, want open", s.Value())
	}

	secretProps, err := imageByName(t, f, "Secret").Properties()
	if err != nil {
		t.Fatalf("secret properties (fallback) failed: %v", err)
	}
	if s := secretProps[0].(*property.StringProperty); s.Value() != "hidden" {
		t.Fatalf("secret name = %q, want hidden", s.Value())
	}
}

// TestCanvasKeyForFallbackImage: canvases inside a fallback-keyed image must
// resolve the fallback key; canvases elsewhere resolve the file key.
func TestCanvasKeyForFallbackImage(t *testing.T) {
	path := writeFixture(t, mixedArchive([]byte{9, 9, 9}), "String.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	secret := imageByName(t, f, "Secret")
	props, err := secret.Properties()
	if err != nil {
		t.Fatalf("secret properties: %v", err)
	}
	var cp *property.CanvasProperty
	for _, p := range props {
		if c, ok := p.(*property.CanvasProperty); ok {
			cp = c
		}
	}
	if cp == nil {
		t.Fatalf("no canvas in Secret image")
	}
	kms := crypto.GetKeyForRegion(crypto.EncryptionKMS).Bytes(0x10000)
	if got := f.CanvasEncryptionKeyFor(cp.DataOffset()); !bytes.Equal(got, kms) {
		t.Fatalf("canvas key for fallback image != KMS key")
	}
	// An offset far outside any registered image extent → file-level key (None → empty).
	if got := f.CanvasEncryptionKeyFor(1 << 40); len(got) != 0 {
		t.Fatalf("out-of-range offset should resolve the (empty) file key, got %d bytes", len(got))
	}
}

// TestPerImageFallbackConcurrent: fallback parse under parseMu is race-free.
func TestPerImageFallbackConcurrent(t *testing.T) {
	path := writeFixture(t, mixedArchive([]byte{1}), "String.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	var wg sync.WaitGroup
	for _, name := range []string{"Plain", "Secret", "Plain", "Secret"} {
		wg.Add(1)
		img := imageByName(t, f, name)
		go func() { //goroutine-guard:allow — test-local concurrency probe, joined by wg.Wait
			defer wg.Done()
			_, _ = img.Properties()
		}()
	}
	wg.Wait()
	props, err := imageByName(t, f, "Secret").Properties()
	if err != nil {
		t.Fatalf("secret after concurrent access: %v", err)
	}
	if s := props[0].(*property.StringProperty); s.Value() != "hidden" {
		t.Fatalf("secret name = %q, want hidden", s.Value())
	}
}
