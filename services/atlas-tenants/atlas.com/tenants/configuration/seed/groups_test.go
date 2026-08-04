package seed_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/configuration/seed"
	"atlas-tenants/test"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/sirupsen/logrus"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newRouter(t *testing.T) *mux.Router {
	t.Helper()
	// test.SetupTestDB migrates tenant.Entity + configuration.Entity;
	// seed_state and the outbox are this task's additions.
	db := test.SetupTestDB(t)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	if err := db.AutoMigrate(&seeder.SeedState{}); err != nil {
		t.Fatalf("migrate seed_state: %v", err)
	}
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	l := logrus.New()
	l.SetOutput(io.Discard)
	r := mux.NewRouter()
	seed.InitResource(db)(r, l)
	configuration.RegisterRoutes(db)(stubServerInformation{})(r, l)
	return r
}

type stubServerInformation struct{}

func (stubServerInformation) GetBaseURL() string { return "" }
func (stubServerInformation) GetPrefix() string  { return "/api/" }

func withTenantHeaders(req *http.Request) *http.Request {
	req.Header.Set(tenant.ID, uuid.New().String())
	req.Header.Set(tenant.Region, "GMS")
	req.Header.Set(tenant.MajorVersion, "83")
	req.Header.Set(tenant.MinorVersion, "1")
	return req
}

func TestSeedRoutesDispatch(t *testing.T) {
	r := newRouter(t)
	for _, res := range []string{"routes", "vessels", "instance-routes"} {
		for _, c := range []struct {
			method string
			path   string
			want   int
		}{
			{http.MethodPost, "/tenants/configurations/" + res + "/seed", http.StatusAccepted},
			{http.MethodGet, "/tenants/configurations/" + res + "/seed/status", http.StatusOK},
		} {
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, withTenantHeaders(httptest.NewRequest(c.method, c.path, nil)))
			if rr.Code != c.want {
				t.Errorf("%s %s = %d, want %d", c.method, c.path, rr.Code, c.want)
			}
		}
	}
}

func TestSeedEndpointsRequireTenantHeaders(t *testing.T) {
	r := newRouter(t)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/configurations/routes/seed", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 without tenant headers", rr.Code)
	}
}

// The literal seed paths must not shadow the surviving path-scoped CRUD
// routes, and vice versa.
func TestCrudRoutesStillDispatch(t *testing.T) {
	r := newRouter(t)
	path := "/tenants/" + uuid.New().String() + "/configurations/routes"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	if rr.Code == http.StatusNotFound {
		t.Fatalf("GET %s returned 404 — the seed routes shadowed the CRUD handler", path)
	}
}

// The three removed path-scoped seed endpoints must be gone; the two
// out-of-scope ones must remain.
func TestRemovedAndRetainedSeedEndpoints(t *testing.T) {
	r := newRouter(t)
	tid := uuid.New().String()
	for _, res := range []string{"routes", "vessels", "instance-routes"} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/"+tid+"/configurations/"+res+"/seed", nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("POST path-scoped %s/seed = %d, want 404 (endpoint must be removed)", res, rr.Code)
		}
	}
	for _, res := range []string{"rps-rewards", "mts-configs"} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/"+tid+"/configurations/"+res+"/seed", nil))
		if rr.Code == http.StatusNotFound {
			t.Errorf("POST path-scoped %s/seed = 404, want it retained (out of scope)", res)
		}
	}
}
