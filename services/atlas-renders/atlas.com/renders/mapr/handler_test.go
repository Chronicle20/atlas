package mapr

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

// TestCompositeFromWZPublicSurface is a compile-time check that the
// post-refactor entry point exists with the expected signature. Behavioral
// tests against CompositeFromWZ would need a real Map.wz fixture (the
// upstream mapimage package keeps its image-extraction tests behind a
// fixture-dependent build path for the same reason).
func TestCompositeFromWZPublicSurface(t *testing.T) {
	_ = CompositeFromWZ
}
