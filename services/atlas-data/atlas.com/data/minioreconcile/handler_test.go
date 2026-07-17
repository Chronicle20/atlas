package minioreconcile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atlas-data/rest"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// newTestHandler builds the same route InitResource wires up — POST
// /data/minio/reconcile decoding the JSON:API body into ReconcileInputModel
// and dispatching to reconcileInner — but around an injected fake Store and
// the package's `now` test clock (see reconcile_test.go), bypassing the
// ParseTenant middleware that RegisterInputHandler normally wraps around the
// decode step. This mirrors tenantpurge/handler_test.go's purgeViaRouter: the
// reconcile sweep is a cross-tenant operator action, not a tenant-scoped
// request, so routing straight to rest.ParseInput + the inner handler (rather
// than through server.RegisterInputHandler's ParseTenant gate) is the correct
// mirror of how this repo tests operator-gated, non-tenant-scoped handlers.
func newTestHandler(t *testing.T, store Store) http.Handler {
	t.Helper()
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	router := mux.NewRouter()
	router.HandleFunc("/data/minio/reconcile", rest.ParseInput[ReconcileInputModel](&d, &c, reconcileInner(store, now))).Methods(http.MethodPost)
	return router
}

func TestHandler_RequiresOperator(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/data/minio/reconcile", strings.NewReader(`{"data":{"type":"minioReconciles","attributes":{"keepTenantIds":["x"]}}}`))
	// no X-Atlas-Operator header
	newTestHandler(t, &fakeStore{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}

func TestHandler_EmptyKeepListIs422(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/data/minio/reconcile", strings.NewReader(`{"data":{"type":"minioReconciles","attributes":{"keepTenantIds":[]}}}`))
	req.Header.Set("X-Atlas-Operator", "1")
	newTestHandler(t, &fakeStore{buckets: []string{"atlas-wz"}, prefixes: map[string]map[string]PrefixInfo{"atlas-wz": {}}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rr.Code)
	}
}
