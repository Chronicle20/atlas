package dataskill

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

// TestRequestById_RoundTrip stands up an httptest server returning a
// realistic JSON:API document for a data-skill resource, INCLUDING a
// relationships block, and drives it through the real requests.GetRequest
// decode path. This proves the SetToOneReferenceID/SetToManyReferenceIDs
// stubs added per libs/atlas-rest/CLAUDE.md let api2go decode a response
// that carries relationships.
//
// It also pins the per-level effect "x" attribute (the berserk threshold
// percentage): if the effects array failed to decode, the berserk
// threshold lookup would silently fail and berserk would never activate.
func TestRequestById_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/skills/1320006") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "skills",
				"id": "1320006",
				"attributes": {
					"effects": [
						{"x": 0},
						{"x": 50}
					]
				},
				"relationships": {
					"job": {
						"data": {"type": "jobs", "id": "112"}
					},
					"consumeItems": {
						"data": []
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rm, err := RequestById(1320006)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("RequestById: %v", err)
	}
	if rm.Id != 1320006 {
		t.Errorf("Id = %d, want 1320006", rm.Id)
	}
	if len(rm.Effects) != 2 {
		t.Fatalf("len(Effects) = %d, want 2", len(rm.Effects))
	}
	if rm.Effects[0].X != 0 {
		t.Errorf("Effects[0].X = %d, want 0", rm.Effects[0].X)
	}
	if rm.Effects[1].X != 50 {
		t.Errorf("Effects[1].X = %d, want 50 (x attribute)", rm.Effects[1].X)
	}
}
