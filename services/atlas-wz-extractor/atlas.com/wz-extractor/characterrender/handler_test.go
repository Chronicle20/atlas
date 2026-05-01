package characterrender

import (
	"bytes"
	"context"
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

	atlasserver "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// testTenantId is the UUID used for all handler tests.
var testTenantId = uuid.MustParse("ec876921-aaaa-bbbb-cccc-deadbeef0000")

// makeAssetsRoot prepares a synthetic assets root with a body skin sprite.
// Returns the root and the tenant-scoped assets path.
func makeAssetsRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	tenantPath := filepath.Join(root, testTenantId.String(), "GMS", "83.1")

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

// newHandlerFunc constructs a mux router that routes through handleRender,
// with the tenant injected into d.Context() as the canonical pattern requires.
// mux.Vars are populated normally because the mux router is used for dispatch.
func newHandlerFunc(root string, comp *characterimage.Compositor, tm atlastenant.Model) *mux.Router {
	l := logrus.New()
	tenantCtx := atlastenant.WithContext(context.Background(), tm)
	d := atlasserver.NewHandlerDependency(l, tenantCtx)
	hc := atlasserver.NewHandlerContext(nil)
	gh := handleRender(root, comp)
	innerHF := gh(&d, &hc)

	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		innerHF).Methods(http.MethodGet)
	return r
}

func TestHandleRenderHappyPath(t *testing.T) {
	root, _ := makeAssetsRoot(t)

	tm, err := atlastenant.Create(testTenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	c := characterimage.NewCompositor()
	r := newHandlerFunc(root, c, tm)

	tenantStr := testTenantId.String()
	canonical := CanonicalLoadoutString(tenantStr, "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 1, nil)
	hash := LoadoutHash(canonical)

	url := "/api/wz/character/render/" + tenantStr + "/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=stand1&frame=0&resize=1&items="
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
	cached := filepath.Join(root, tenantStr, "GMS", "83.1", "character", hash+".png")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("cached file missing: %v", err)
	}
}

func TestHandleRenderHashMismatch(t *testing.T) {
	root, _ := makeAssetsRoot(t)

	tm, err := atlastenant.Create(testTenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	c := characterimage.NewCompositor()
	r := newHandlerFunc(root, c, tm)

	tenantStr := testTenantId.String()
	wrong := hex.EncodeToString(sha256.New().Sum(nil))[:16]
	url := "/api/wz/character/render/" + tenantStr + "/GMS/83.1/" + wrong + ".png?skin=0&hair=30030&face=20000"
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

	tm, err := atlastenant.Create(testTenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	c := characterimage.NewCompositor()
	r := newHandlerFunc(root, c, tm)

	tenantStr := testTenantId.String()
	canonical := CanonicalLoadoutString(tenantStr, "GMS", 83, 1, 0, 30030, 20000, "warp", 0, 1, nil)
	hash := LoadoutHash(canonical)
	url := "/api/wz/character/render/" + tenantStr + "/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=warp&frame=0&resize=1&items="
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

func TestHandleRenderTenantMismatch(t *testing.T) {
	root, _ := makeAssetsRoot(t)

	// Inject a context tenant with a *different* region than the URL path.
	tm, err := atlastenant.Create(testTenantId, "KMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	c := characterimage.NewCompositor()
	r := newHandlerFunc(root, c, tm)

	tenantStr := testTenantId.String()
	canonical := CanonicalLoadoutString(tenantStr, "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 1, nil)
	hash := LoadoutHash(canonical)
	url := "/api/wz/character/render/" + tenantStr + "/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=stand1&frame=0&resize=1&items="
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != 400 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"tenant-mismatch"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
