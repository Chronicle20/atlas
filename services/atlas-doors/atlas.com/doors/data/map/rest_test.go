package map_

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

func TestExtractDoorPortal(t *testing.T) {
	rm := PortalRestModel{Name: "tp", Type: 6, X: -100, Y: 200, TargetMapId: 999}
	p, err := ExtractPortal(rm)
	if err != nil || p.Type() != 6 || p.X() != -100 || p.TargetMapId() != 999 {
		t.Fatalf("portal extract wrong: %+v err=%v", p, err)
	}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestGetById_ServedWithPortals drives a real JSON:API document (with the
// portals to-many relationship + included portal resources) through the api2go
// unmarshal + SetReferencedStructs path and asserts the map and its portals are
// populated.
func TestGetById_ServedWithPortals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/maps/104000000") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "maps",
				"id": "104000000",
				"attributes": {
					"returnMapId": 104000000,
					"forcedReturnMapId": 999999999,
					"town": true,
					"fieldLimit": 0
				},
				"relationships": {
					"portals": {
						"data": [
							{ "type": "portals", "id": "1" },
							{ "type": "portals", "id": "2" }
						]
					}
				}
			},
			"included": [
				{
					"type": "portals",
					"id": "1",
					"attributes": { "name": "dp0", "type": 6, "x": -100, "y": 10, "targetMapId": 0 }
				},
				{
					"type": "portals",
					"id": "2",
					"attributes": { "name": "dp1", "type": 6, "x": 200, "y": 20, "targetMapId": 0 }
				}
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	m, err := NewProcessor(logrus.New(), ctx).GetById(104000000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}

	if m.Id() != 104000000 {
		t.Fatalf("map id: want 104000000, got %d", m.Id())
	}
	if !m.Town() {
		t.Fatalf("expected town=true")
	}
	if got := len(m.Portals()); got != 2 {
		t.Fatalf("expected 2 portals, got %d (%+v)", got, m.Portals())
	}
	p0 := m.Portals()[0]
	if p0.Type() != 6 || p0.X() != -100 || p0.Y() != 10 {
		t.Fatalf("portal[0] wrong: type=%d x=%d y=%d", p0.Type(), p0.X(), p0.Y())
	}
	p1 := m.Portals()[1]
	if p1.X() != 200 || p1.Y() != 20 {
		t.Fatalf("portal[1] wrong: x=%d y=%d", p1.X(), p1.Y())
	}
}
