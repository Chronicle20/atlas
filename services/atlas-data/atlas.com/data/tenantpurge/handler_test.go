package tenantpurge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/sirupsen/logrus"
)

func TestPurgeNilMcReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := purgeInner(nil, nil)(&d, &c)
	req := httptest.NewRequest(http.MethodDelete, "/api/data/tenants/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("X-Atlas-Operator", "1")
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

// Note: testing the operator-gate / canonical-refused branches via the handler
// requires a non-nil *minio.Client. Those branches are covered at the purge
// function level (see purge_test.go) plus a unit-test on the gate is captured
// here only when mc is non-nil; the nil-mc path short-circuits first.
