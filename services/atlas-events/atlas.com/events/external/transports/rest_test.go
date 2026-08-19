package transports

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// routeFixture is a JSON:API document shaped exactly like a real
// GET /transports/routes/{routeId} response: atlas-transports'
// transport.RestModel unconditionally declares a to-many "schedule"
// relationship in GetReferences(), so every real response carries a
// relationships.schedule.data array, even when this client only cares about
// state and voyageId. See transport.RestModel.GetReferences in
// services/atlas-transports/atlas.com/transports/transport/rest.go:70-79.
const routeFixture = `{
  "data": {
    "type": "routes",
    "id": "11111111-1111-1111-1111-111111111111",
    "attributes": {
      "state": "in_transit",
      "voyageId": "22222222-2222-2222-2222-222222222222"
    },
    "relationships": {
      "schedule": {
        "data": [
          {"type": "trip-schedule", "id": "33333333-3333-3333-3333-333333333333"},
          {"type": "trip-schedule", "id": "44444444-4444-4444-4444-444444444444"}
        ]
      }
    }
  }
}`

// F1 (CRITICAL): every real atlas-transports route response carries a
// relationships.schedule.data array. Without SetToOneReferenceID /
// SetToManyReferenceIDs on the events-side RestModel, api2go's
// jsonapi.Unmarshal errors on that block and GetRoute can never succeed
// against a real response. See libs/atlas-rest/CLAUDE.md.
func TestRestModelDecodesRelationshipsBlock(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(routeFixture), &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if rm.Id != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("Id: got %q", rm.Id)
	}
	if rm.State != "in_transit" {
		t.Errorf("State: got %q, want %q", rm.State, "in_transit")
	}
	if rm.VoyageID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("VoyageID: got %q, want %q", rm.VoyageID, "22222222-2222-2222-2222-222222222222")
	}
}
