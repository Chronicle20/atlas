package mapr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-renders/storage"

	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// TestHandlerNoStorageReturns503 verifies the handler short-circuits with a
// 503 when storage is unavailable, before any tenant-context lookup. This is
// the path the readiness checker hits when MinIO init failed at startup.
func TestHandlerNoStorageReturns503(t *testing.T) {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetOutput(io.Discard)
	r.HandleFunc("/api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png", Handler(l, nil)).Methods(http.MethodGet)
	req := httptest.NewRequest(http.MethodGet, "/api/wz/map/render/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/GMS/83.1/100000000/minimap.png", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rr.Code)
	}
}

// TestCompositeStacksLayersInZMapOrder builds an in-memory MapEntry with two
// 4x4 layer PNGs and a zmap that orders them back-to-front. After Composite
// the front layer's color should be visible at the overlap.
func TestCompositeStacksLayersInZMapOrder(t *testing.T) {
	back := solidPNG(t, 4, 4, color.NRGBA{R: 255, A: 255})
	front := solidPNG(t, 4, 4, color.NRGBA{B: 255, A: 255})

	entry := &storage.MapEntry{
		Layout: maplayout.Layout{
			MapID:  100000000,
			Bounds: maplayout.Bounds{Left: 0, Top: 0, Right: 4, Bottom: 4},
			Layers: []maplayout.Layer{
				{ID: 0, Name: "layer-0", Z: 0},
				{ID: 1, Name: "layer-1", Z: 1},
			},
			ZMap: []string{"layer-0", "layer-1"}, // back, then front
		},
		Layers: map[int][]byte{
			0: back,
			1: front,
		},
	}

	l := logrus.New()
	l.SetOutput(io.Discard)
	img, err := Composite(l, entry)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	c := img.At(2, 2)
	r, g, b, _ := c.RGBA()
	if r != 0 || g != 0 || b == 0 {
		t.Fatalf("front layer (blue) not visible at (2,2): got rgb=(%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

// TestCompositeRejectsEmptyBounds guards against zero-size canvases (would
// otherwise produce a 0x0 PNG that the client cannot use).
func TestCompositeRejectsEmptyBounds(t *testing.T) {
	entry := &storage.MapEntry{Layout: maplayout.Layout{Bounds: maplayout.Bounds{}}}
	if _, err := Composite(logrus.New(), entry); err == nil {
		t.Fatal("expected error for empty bounds")
	}
}

func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}
