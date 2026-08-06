package icons_test

import (
	"bytes"
	"compress/zlib"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// Marker pixels. wztest.Canvas hardcodes 1x1 / format 2 (FormatBGRA8888),
// so fixture canvases are indistinguishable by dimension — every test tells
// them apart by decoded pixel value instead. Payload byte order is B,G,R,A.
var (
	markDefault  = color.NRGBA{R: 0x33, G: 0x22, B: 0x11, A: 0xFF}
	markStand    = color.NRGBA{R: 0x66, G: 0x55, B: 0x44, A: 0xFF}
	markLink     = color.NRGBA{R: 0x99, G: 0x88, B: 0x77, A: 0xFF}
	markTopLevel = color.NRGBA{R: 0xCC, G: 0xBB, B: 0xAA, A: 0xFF}
)

// payloadFor returns the canvas payload that decodes to the given marker.
func payloadFor(t *testing.T, m color.NRGBA) []byte {
	t.Helper()
	return pixelPayload(t, m.B, m.G, m.R, m.A)
}

// pixelPayload builds a canvas payload for a single BGRA pixel. canvas
// .Decompress tries zlib first (isZlibHeader wants 0x78 followed by
// 0x9C/0xDA/0x01/0x5E); zlib.NewWriter's default compression emits 0x78 0x9C.
func pixelPayload(t *testing.T, b, g, r, a byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte{b, g, r, a}); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// openFixture serializes the builder to a temp Npc.wz and opens it.
func openFixture(t *testing.T, b *wztest.Builder) *wz.File {
	t.Helper()
	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "Npc.wz")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	f, err := wz.Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// newArchive returns a builder preloaded with the settings wz.Open accepts
// for these fixtures (verified: "Detected version 83 (hash=1876)").
func newArchive() *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone)
}

// pixelAt decodes the single pixel of a 1x1 fixture canvas.
func pixelAt(t *testing.T, img image.Image) color.NRGBA {
	t.Helper()
	if img == nil {
		t.Fatalf("nil image")
	}
	return color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
}
