package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMetricsRouteAutoMounted(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	sb := New(l)
	h := sb.routerProducer(l)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from auto-mounted /metrics, got %d", rec.Code)
	}
}
