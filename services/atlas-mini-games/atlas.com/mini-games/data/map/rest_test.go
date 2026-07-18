package mapdata

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

// TestFieldLimit_ServedWithRelationships drives a REAL atlas-data map document
// (which ALWAYS carries a portals/reactors/npcs/monsters relationships block)
// through the api2go unmarshal path. Without the SetToManyReferenceIDs no-op
// stub this fails with "does not implement UnmarshalToManyRelations" and
// FieldLimit returns an error, which would kill room creation in production
// (EXT-01, task-037 failure class).
func TestFieldLimit_ServedWithRelationships(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/maps/910000000") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "maps",
				"id": "910000000",
				"attributes": {
					"fieldLimit": 128
				},
				"relationships": {
					"portals": {
						"data": [
							{ "type": "portals", "id": "1" },
							{ "type": "portals", "id": "2" }
						]
					},
					"reactors": { "data": [] },
					"npcs": { "data": [ { "type": "npcs", "id": "9010000" } ] },
					"monsters": { "data": [] }
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	fl, err := NewProcessor(logrus.New(), ctx).FieldLimit(910000000)
	if err != nil {
		t.Fatalf("FieldLimit: %v", err)
	}
	if fl != 128 {
		t.Fatalf("fieldLimit: want 128, got %d", fl)
	}
}
