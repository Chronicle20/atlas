package configuration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// TestDefaultFetcherRoundTripsTenantConfig serves a real JSON:API kite-configs
// document -- including a relationships block, the exact shape that trips
// api2go when a RestModel is missing SetToOneReferenceID/SetToManyReferenceIDs
// (libs/atlas-rest/CLAUDE.md) -- and asserts defaultFetcher decodes it into a
// Model carrying the served knob values. EXT-02 disqualifies Extract-only unit
// tests as insufficient because they never exercise api2go's unmarshal path;
// this does.
func TestDefaultFetcherRoundTripsTenantConfig(t *testing.T) {
	tenantId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"` + tenantId.String() + `","type":"kite-configs","attributes":{"maxPerMap":3,"maxMessageLength":40,"blockedMapPrefixes":[91,92]},"relationships":{"tenant":{"data":{"id":"` + tenantId.String() + `","type":"tenants"}}}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	m, err := defaultFetcher(testLogger(t), context.Background(), tenantId)
	if err != nil {
		t.Fatalf("defaultFetcher: %v", err)
	}
	if m.MaxPerMap() != 3 {
		t.Errorf("MaxPerMap = %d, want 3", m.MaxPerMap())
	}
	if m.MaxMessageLength() != 40 {
		t.Errorf("MaxMessageLength = %d, want 40", m.MaxMessageLength())
	}
	if len(m.BlockedMapPrefixes()) != 2 || m.BlockedMapPrefixes()[0] != 91 || m.BlockedMapPrefixes()[1] != 92 {
		t.Errorf("BlockedMapPrefixes = %v, want [91 92]", m.BlockedMapPrefixes())
	}
}

// TestGetTenantConfigFallsBackToDefaultOnFetchFailure proves that a genuine
// fetch failure (here, a 5xx from atlas-tenants) still yields DefaultConfig
// via the registry's cache path -- the deliberate degrade-gracefully behaviour
// (Finding 5), which must survive now that GetTenantConfig also distinguishes
// 404 from other failures for logging purposes.
func TestGetTenantConfigFallsBackToDefaultOnFetchFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	reg := newRegistryWithFetcher(defaultFetcher)
	cfg := reg.GetTenantConfig(testLogger(t), context.Background(), uuid.New())

	d := DefaultConfig()
	if cfg.MaxPerMap() != d.MaxPerMap() {
		t.Errorf("MaxPerMap = %d, want default %d", cfg.MaxPerMap(), d.MaxPerMap())
	}
	if cfg.MaxMessageLength() != d.MaxMessageLength() {
		t.Errorf("MaxMessageLength = %d, want default %d", cfg.MaxMessageLength(), d.MaxMessageLength())
	}
	if len(cfg.BlockedMapPrefixes()) != len(d.BlockedMapPrefixes()) {
		t.Errorf("BlockedMapPrefixes = %v, want default %v", cfg.BlockedMapPrefixes(), d.BlockedMapPrefixes())
	}
}

// TestGetTenantConfigFallsBackToDefaultOn404 proves the un-provisioned-tenant
// path (a genuine 404, no kite-configs row seeded) also degrades to
// DefaultConfig -- same outcome as the 5xx case, but the log line it produces
// is distinguished (Info, not Warn) so the two failure modes are visible
// separately in practice, per Finding 5.
func TestGetTenantConfigFallsBackToDefaultOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	reg := newRegistryWithFetcher(defaultFetcher)
	cfg := reg.GetTenantConfig(testLogger(t), context.Background(), uuid.New())

	d := DefaultConfig()
	if cfg.MaxPerMap() != d.MaxPerMap() {
		t.Errorf("MaxPerMap = %d, want default %d", cfg.MaxPerMap(), d.MaxPerMap())
	}
}
