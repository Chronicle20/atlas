package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
// realistic JSON:API document for a character resource, INCLUDING a
// relationships block, and drives it through the real requests.GetRequest
// decode path. This proves the SetToOneReferenceID/SetToManyReferenceIDs
// stubs added per libs/atlas-rest/CLAUDE.md let api2go decode a response
// that carries relationships, and pins the wire-tag mapping for hp/level.
func TestRequestById_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "characters",
				"id": "42",
				"attributes": {
					"level": 120,
					"hp": 5000
				},
				"relationships": {
					"map": {
						"data": {"type": "maps", "id": "100000000"}
					},
					"inventories": {
						"data": [
							{"type": "inventories", "id": "1"},
							{"type": "inventories", "id": "2"}
						]
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rm, err := RequestById(42)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("RequestById: %v", err)
	}
	if rm.Id != 42 {
		t.Errorf("Id = %d, want 42", rm.Id)
	}
	if rm.Level != 120 {
		t.Errorf("Level = %d, want 120", rm.Level)
	}
	if rm.Hp != 5000 {
		t.Errorf("Hp = %d, want 5000", rm.Hp)
	}
}
