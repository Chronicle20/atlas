package wzinput

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/sirupsen/logrus"
)

func TestUploadHandlerNilMinIOReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := uploadHandler(nil)(&d, &c)
	req := httptest.NewRequest(http.MethodPatch, "/api/data/wz", &bytes.Buffer{})
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestStatusHandlerNilMinIOReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := statusHandler(nil)(&d, &c)
	req := httptest.NewRequest(http.MethodGet, "/api/data/wz", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}
