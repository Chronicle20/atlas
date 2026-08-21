package configuration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestGetByTenantId_Configured asserts the JSON:API decode of a captured
// tenants/{id}/configurations/player-npcs payload.
func TestGetByTenantId_Configured(t *testing.T) {
	tenantId := uuid.New()
	wantPath := "/tenants/" + tenantId.String() + "/configurations/player-npcs"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"player-npcs","id":"` + tenantId.String() + `","attributes":{
			"initialX":100,"initialY":200,"areaX":50,"areaY":60,"areaSteps":2,
			"organizeArea":false,"autoDeployEnabled":false
		}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	m := NewProcessor(logrus.New(), testCtx(t)).GetByTenantId(tenantId)
	if m.InitialX() != 100 || m.InitialY() != 200 || m.AreaX() != 50 || m.AreaY() != 60 || m.AreaSteps() != 2 {
		t.Errorf("m = %+v, want configured values", m)
	}
	if m.OrganizeArea() || m.AutoDeployEnabled() {
		t.Errorf("m = %+v, want organizeArea/autoDeployEnabled false", m)
	}
}

// TestGetByTenantId_DefaultsOn404 asserts a 404 from atlas-tenants (the
// unconfigured state) yields the FR-4.7 defaults.
func TestGetByTenantId_DefaultsOn404(t *testing.T) {
	tenantId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	m := NewProcessor(logrus.New(), testCtx(t)).GetByTenantId(tenantId)
	want := DefaultModel()
	if m != want {
		t.Errorf("m = %+v, want defaults %+v", m, want)
	}
}

// TestGetByTenantId_DefaultsOnOtherError asserts that a non-404 upstream
// failure also falls back to defaults rather than stalling deployment.
func TestGetByTenantId_DefaultsOnOtherError(t *testing.T) {
	tenantId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	m := NewProcessor(logrus.New(), testCtx(t)).GetByTenantId(tenantId)
	want := DefaultModel()
	if m != want {
		t.Errorf("m = %+v, want defaults %+v", m, want)
	}
}
