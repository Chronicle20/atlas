package characterrender

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"atlas-wz-extractor/characterimage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// makeAssetsRoot prepares a synthetic assets root with a body skin sprite.
func makeAssetsRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	tenantPath := filepath.Join(root, "tenant-a", "GMS", "83.1")

	// character-meta
	if err := os.MkdirAll(filepath.Join(tenantPath, "character-meta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(tenantPath, "character-meta", "zmap.json"),
		[]byte(`["body","arm"]`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-meta", "smap.json"),
		[]byte(`{}`), 0o644)

	// body skin 0 — stripped form per normalizeId
	bodyDir := filepath.Join(tenantPath, "character-parts", "2000", "stand1", "0")
	_ = os.MkdirAll(bodyDir, 0o755)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	f, _ := os.Create(filepath.Join(bodyDir, "body.png"))
	defer f.Close()
	_ = png.Encode(f, img)
	_ = os.WriteFile(filepath.Join(bodyDir, "body.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{"neck":{"x":0,"y":-3}},"z":"body"}`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-parts", "2000", "info.json"),
		[]byte(`{"islot":"Bd","vslot":"Bd","cash":0}`), 0o644)

	// hair 30030 (stripped form per normalizeId)
	hairDir := filepath.Join(tenantPath, "character-parts", "30030", "stand1", "0")
	_ = os.MkdirAll(hairDir, 0o755)
	hImg := image.NewRGBA(image.Rect(0, 0, 4, 4))
	hf, _ := os.Create(filepath.Join(hairDir, "hair.png"))
	defer hf.Close()
	_ = png.Encode(hf, hImg)
	_ = os.WriteFile(filepath.Join(hairDir, "hair.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{},"z":"hairOverHead"}`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-parts", "30030", "info.json"),
		[]byte(`{"islot":"Hr","vslot":"Hr","cash":0}`), 0o644)

	// face 20000 (stripped form per normalizeId)
	faceDir := filepath.Join(tenantPath, "character-parts", "20000", "stand1", "0")
	_ = os.MkdirAll(faceDir, 0o755)
	faImg := image.NewRGBA(image.Rect(0, 0, 4, 4))
	faF, _ := os.Create(filepath.Join(faceDir, "face.png"))
	defer faF.Close()
	_ = png.Encode(faF, faImg)
	_ = os.WriteFile(filepath.Join(faceDir, "face.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{},"z":"head"}`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-parts", "20000", "info.json"),
		[]byte(`{"islot":"Fc","vslot":"Fc","cash":0}`), 0o644)

	return root, tenantPath
}

func TestHandleRenderHappyPath(t *testing.T) {
	root, _ := makeAssetsRoot(t)

	c := characterimage.NewCompositor()
	h := NewHandler(root, c)

	canonical := CanonicalLoadoutString("tenant-a", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 1, nil)
	hash := LoadoutHash(canonical)

	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=stand1&frame=0&resize=1&items="
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("cache-control = %q", cc)
	}
	if e := rec.Header().Get("ETag"); e != "\""+hash+"\"" {
		t.Fatalf("etag = %q", e)
	}

	// Verify the file landed on disk.
	cached := filepath.Join(root, "tenant-a", "GMS", "83.1", "character", hash+".png")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("cached file missing: %v", err)
	}
}

func TestHandleRenderHashMismatch(t *testing.T) {
	root, _ := makeAssetsRoot(t)
	c := characterimage.NewCompositor()
	h := NewHandler(root, c)
	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	wrong := hex.EncodeToString(sha256.New().Sum(nil))[:16]
	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + wrong + ".png?skin=0&hair=30030&face=20000"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"hash-mismatch"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHandleRenderInvalidStance(t *testing.T) {
	root, _ := makeAssetsRoot(t)
	c := characterimage.NewCompositor()
	h := NewHandler(root, c)
	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	canonical := CanonicalLoadoutString("tenant-a", "GMS", 83, 1, 0, 30030, 20000, "warp", 0, 1, nil)
	hash := LoadoutHash(canonical)
	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=warp&frame=0&resize=1&items="
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != 400 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"invalid-stance"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	_ = strconv.IntSize // pacify import
}
