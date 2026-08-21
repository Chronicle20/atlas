package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func TestRegisterRoutes_AfterSeedRunsOnceWithTenantContext(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	t.Cleanup(backgroundSeeds.Wait)

	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))

	var mu sync.Mutex
	calls := 0
	var sawTenant uuid.UUID
	var sawGroup string

	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
		AfterSeed: func(ctx context.Context, _ *gorm.DB, res Result) error {
			mu.Lock()
			defer mu.Unlock()
			calls++
			tm := tenant.MustFromContext(ctx)
			sawTenant = tm.Id()
			sawGroup = res.GroupName
			return nil
		},
	}
	r := mux.NewRouter()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	req := requestWithTenant(t, "POST", srv.URL+"/widgets/seed", nil)
	wantTenant := req.Header.Get(tenant.ID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	backgroundSeeds.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("AfterSeed called %d times, want exactly 1", calls)
	}
	if sawTenant.String() != wantTenant {
		t.Fatalf("AfterSeed tenant = %s, want %s", sawTenant, wantTenant)
	}
	if sawGroup != "widgets-group" {
		t.Fatalf("AfterSeed result.GroupName = %q, want %q", sawGroup, "widgets-group")
	}
}

func TestRegisterRoutes_NilAfterSeedIsANoOp(t *testing.T) {
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

	resp, err := http.DefaultClient.Do(requestWithTenant(t, "POST", srv.URL+"/widgets/seed", nil))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	backgroundSeeds.Wait()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

// TestRegisterRoutes_AllFilesRejectedLogsError covers the scenario behind
// bug-event-definition-seed-yields-zero.md: every file in a subdomain fails
// to load (e.g. a type/id mismatch), Seed() still returns a nil error, and
// postSeed must not report that silently at info level as "Seed complete"
// with zero context. It must log at error level with the failure detail
// attached.
func TestRegisterRoutes_AllFilesRejectedLogsError(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	t.Cleanup(backgroundSeeds.Wait)

	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:      "all-rejected",
		URLPrefix: "/all-rejected",
		Subdomains: []SubdomainAny{
			AdaptSubdomain[widgetAttrs, widgetRow](&failingSubdomain{}),
		},
	}
	r := mux.NewRouter()
	logger, hook := logrustest.NewNullLogger()
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.DefaultClient.Do(requestWithTenant(t, "POST", srv.URL+"/all-rejected/seed", nil))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	backgroundSeeds.Wait()

	var found *logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Message == "Seed complete" {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatalf("no \"Seed complete\" log entry found; entries: %+v", hook.AllEntries())
	}
	if found.Level != logrus.ErrorLevel {
		t.Fatalf("level = %v, want error", found.Level)
	}
	failed, ok := found.Data["failed"].(map[string]int64)
	if !ok || failed["broken"] != 2 {
		t.Fatalf("failed field = %v, want map with broken=2", found.Data["failed"])
	}
	errs, ok := found.Data["errors"].(map[string][]string)
	if !ok || len(errs["broken"]) == 0 {
		t.Fatalf("errors field = %v, want non-empty broken entries", found.Data["errors"])
	}
	if found.Data["outcome"] != "failure" {
		t.Fatalf("outcome = %v, want failure", found.Data["outcome"])
	}
}
