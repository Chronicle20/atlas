package seeder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// requestWithTenant builds a request with the four tenant headers
// server.ParseTenant requires.
func requestWithTenant(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	tm := tenantGMS83(t)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set(tenant.ID, tm.Id().String())
	req.Header.Set(tenant.Region, tm.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", tm.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", tm.MinorVersion()))
	return req
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
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.DefaultClient.Do(requestWithTenant(t, "POST", srv.URL+"/widgets/seed", nil))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func TestRegisterRoutes_PostMissingTenantReturns400(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
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
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/widgets/seed", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
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
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.DefaultClient.Do(requestWithTenant(t, "GET", srv.URL+"/widgets/seed/status", nil))
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

func TestRegisterRoutes_GetStatusMissingTenantReturns400(t *testing.T) {
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
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/widgets/seed/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
