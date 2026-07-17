package chalkboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestHasOpen_Served drives a real JSON:API chalkboard document through the
// unmarshal path; a present resource means the character has an open
// chalkboard (EXT-02).
func TestHasOpen_Served(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chalkboards/777") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "chalkboards",
				"id": "777",
				"attributes": { "message": "selling stuff" }
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	open, err := NewProcessor(logrus.New(), ctx).HasOpen(777)
	if err != nil {
		t.Fatalf("HasOpen: %v", err)
	}
	if !open {
		t.Fatalf("HasOpen: want true for a served chalkboard, got false")
	}
}

// TestHasOpen_NotFound asserts a 404 is treated as "no open chalkboard"
// (fail-open: absence must not block opening a mini-room).
func TestHasOpen_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	open, err := NewProcessor(logrus.New(), ctx).HasOpen(777)
	if err != nil {
		t.Fatalf("HasOpen (404): unexpected error %v", err)
	}
	if open {
		t.Fatalf("HasOpen (404): want false, got true")
	}
}
