package baseline

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-data/rest"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/sirupsen/logrus"
)

func TestPublishNilMcReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := publishInner(nil, nil, logrus.New())(&d, &c)
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/publish", bytes.NewReader([]byte(`{"region":"GMS"}`)))
	rr := httptest.NewRecorder()
	h(rr, req)
	// mc is nil -> 503, before the operator check.
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestPublishRequiresOperatorWhenMcPresent(t *testing.T) {
	// Smoke check that with a non-nil mc, missing X-Atlas-Operator yields 403.
	// We can't easily construct a real *minio.Client here without env, so we
	// only assert the 503 path above. This test is kept as a placeholder for
	// when mc construction is wired into a test harness.
	_ = rest.RegisterHandler // ensure import used
}

func TestRestoreNilMcReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := restoreInner(nil, nil, logrus.New())(&d, &c)
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/restore", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}
