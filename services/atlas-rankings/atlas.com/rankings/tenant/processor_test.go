package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// tenantsFixture renders a JSON:API document for two tenants with distinct,
// non-overlapping region/version values so a field transposition in
// Extract/Transform would be caught. The real atlas-tenants /tenants
// response does not currently emit a relationships block (see
// services/atlas-tenants/atlas.com/tenants/tenant/rest.go — it has no
// SetToOneReferenceID/SetToManyReferenceIDs/SetReferencedStructs methods at
// all), so this fixture manufactures a synthetic named to-one relationship
// ("createdBy"), a synthetic to-many relationship ("configurations"), and a
// top-level "included" section, mirroring the working pattern in
// character/processor_test.go. This genuinely drives api2go's
// setRelationshipIDs/setIncludedIntoTarget unmarshal path (jsonapi/unmarshal.go)
// into all three stub methods on RestModel (rest.go): a non-empty
// "relationships" map with "data": {...} triggers SetToOneReferenceID, a
// "data": [...] array triggers SetToManyReferenceIDs, and a non-empty
// top-level "included" list triggers SetReferencedStructs. Without those
// three stubs, api2go's type assertion fails and GetAll returns an error
// instead of a decoded list — an empty "relationships": {} block would NOT
// catch that, since setRelationshipIDs's range loop performs zero
// iterations over an empty map.
func tenantsFixture(id1, id2 uuid.UUID) string {
	return `{
  "data": [
    {
      "type": "tenants",
      "id": "` + id1.String() + `",
      "attributes": {
        "name": "Alpha",
        "region": "GMS",
        "majorVersion": 83,
        "minorVersion": 1
      },
      "relationships": {
        "createdBy": {"data": {"type": "accounts", "id": "500"}},
        "configurations": {"data": [{"type": "configurations", "id": "cfg-1"}]}
      }
    },
    {
      "type": "tenants",
      "id": "` + id2.String() + `",
      "attributes": {
        "name": "Beta",
        "region": "JMS",
        "majorVersion": 185,
        "minorVersion": 2
      },
      "relationships": {
        "createdBy": {"data": {"type": "accounts", "id": "501"}},
        "configurations": {"data": [{"type": "configurations", "id": "cfg-2"}]}
      }
    }
  ],
  "included": [
    {"type": "configurations", "id": "cfg-1", "attributes": {}},
    {"type": "configurations", "id": "cfg-2", "attributes": {}}
  ]
}`
}

func TestGetAllDecodesTenantList(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tenants" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(tenantsFixture(id1, id2)))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	ts, err := NewProcessor(logrus.New(), context.Background()).GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(ts) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(ts))
	}

	byId := make(map[uuid.UUID]tenant.Model, len(ts))
	for _, m := range ts {
		byId[m.Id()] = m
	}

	a, ok := byId[id1]
	if !ok {
		t.Fatalf("tenant 1 (%s) missing from decoded list", id1)
	}
	if a.Region() != "GMS" || a.MajorVersion() != 83 || a.MinorVersion() != 1 {
		t.Fatalf("tenant 1 decoded wrong: region=%s major=%d minor=%d", a.Region(), a.MajorVersion(), a.MinorVersion())
	}

	b, ok := byId[id2]
	if !ok {
		t.Fatalf("tenant 2 (%s) missing from decoded list", id2)
	}
	if b.Region() != "JMS" || b.MajorVersion() != 185 || b.MinorVersion() != 2 {
		t.Fatalf("tenant 2 decoded wrong: region=%s major=%d minor=%d", b.Region(), b.MajorVersion(), b.MinorVersion())
	}
}
