package seeder

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func tenantMiddleware(t *testing.T, next http.Handler) http.Handler {
	tm := tenantGMS83(t)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tenant.WithContext(r.Context(), tm)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestRegisterRoutes_PostReturns202(t *testing.T) {
	// Register ResetMetricsForTest first (LIFO: runs last in cleanup).
	t.Cleanup(ResetMetricsForTest)
	// Register backgroundSeeds.Wait() second (LIFO: runs first in cleanup).
	// This drains outstanding Seed goroutines before metrics are reset,
	// preventing a data race between the goroutine writing metrics and
	// ResetMetricsForTest zeroing the same pointers.
	t.Cleanup(backgroundSeeds.Wait)

	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	r := mux.NewRouter()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(tenantMiddleware(t, r))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/widgets/seed", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func TestRegisterRoutes_GetStatusReturnsJSON(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	r := mux.NewRouter()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(tenantMiddleware(t, r))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/widgets/seed/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["catalogRevision"]; !ok {
		t.Fatalf("missing catalogRevision: %v", body)
	}
	b, _ := json.Marshal(body)
	if !strings.Contains(string(b), "widgets") {
		t.Fatalf("subdomain key missing: %s", string(b))
	}
}
