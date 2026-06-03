package drop_test

import (
	"strings"
	"testing"

	"atlas-drops-information/reactor/drop"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// The reactor-drop catalog files use a JSON:API envelope where the
// per-drop attributes live in `included[]`, NOT inline under
// `data.attributes`. The seeder library's pipeline passes the full file
// bytes to Decode, and Decode must walk included[] to materialize the
// drop list. This shape comes from how the original WZ-to-JSON:API
// conversion was done; the live PR-env catalog ships these files
// verbatim, so the test fixture mirrors that exact shape.
//
// A regression in this Decode (e.g. reverting to a flat
// data.attributes.drops parse) would silently produce zero drops, which
// is exactly the symptom that surfaced in atlas-pr-543 before this
// fix landed.
const reactorDropFixture = `{
  "jsonapi": {"version": "1.1"},
  "data": {
    "type": "reactor-drop",
    "id": "1002008",
    "relationships": {
      "drops": {
        "data": [
          {"type": "drops", "id": "1002008:4032452:22502"},
          {"type": "drops", "id": "1002008:1:0"}
        ]
      }
    }
  },
  "included": [
    {
      "type": "drops",
      "id": "1002008:4032452:22502",
      "attributes": {"itemId": 4032452, "chance": 1, "questId": 22502}
    },
    {
      "type": "drops",
      "id": "1002008:1:0",
      "attributes": {"itemId": 1, "chance": 1000000}
    }
  ]
}`

func TestSubdomain_Decode_WalksIncluded(t *testing.T) {
	sd := drop.Subdomain{}
	jm, err := sd.Decode([]byte(reactorDropFixture))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(jm.Drops) != 2 {
		t.Fatalf("len(Drops) = %d, want 2; got %+v", len(jm.Drops), jm.Drops)
	}
	// Ordered the same way they appear in included[].
	if jm.Drops[0].ItemID != 4032452 || jm.Drops[0].Chance != 1 || jm.Drops[0].QuestID != 22502 {
		t.Errorf("Drops[0] = %+v, want itemId=4032452 chance=1 questId=22502", jm.Drops[0])
	}
	if jm.Drops[1].ItemID != 1 || jm.Drops[1].Chance != 1000000 || jm.Drops[1].QuestID != 0 {
		t.Errorf("Drops[1] = %+v, want itemId=1 chance=1000000 questId=0", jm.Drops[1])
	}
}

func TestSubdomain_Decode_IgnoresNonDropIncluded(t *testing.T) {
	// included[] may contain entries of types other than "drops"; the
	// decoder should skip them rather than failing or treating their
	// attributes as a drop.
	const mixed = `{
		"data": {"type":"reactor-drop","id":"1","relationships":{"drops":{"data":[{"type":"drops","id":"1:1:0"}]}}},
		"included": [
			{"type":"some-other-type","id":"x","attributes":{"itemId":999}},
			{"type":"drops","id":"1:1:0","attributes":{"itemId":1001,"chance":5}}
		]
	}`
	sd := drop.Subdomain{}
	jm, err := sd.Decode([]byte(mixed))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(jm.Drops) != 1 {
		t.Fatalf("len(Drops) = %d, want 1 (non-drop included ignored)", len(jm.Drops))
	}
	if jm.Drops[0].ItemID != 1001 {
		t.Errorf("Drops[0].ItemID = %d, want 1001", jm.Drops[0].ItemID)
	}
}

func TestSubdomain_Decode_MalformedJSON(t *testing.T) {
	sd := drop.Subdomain{}
	_, err := sd.Decode([]byte(`{"data":`))
	if err == nil {
		t.Fatalf("expected error on malformed JSON")
	}
	if !strings.Contains(err.Error(), "reactor-drop") {
		t.Errorf("expected reactor-drop-prefixed error, got: %v", err)
	}
}

func TestSubdomain_Build_BuildsOneRowPerDrop(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	sd := drop.Subdomain{}
	jm, err := sd.Decode([]byte(reactorDropFixture))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	models, err := sd.Build(tm, "1002008", jm)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	for i, m := range models {
		if m.ReactorId() != 1002008 {
			t.Errorf("models[%d].ReactorId = %d, want 1002008", i, m.ReactorId())
		}
	}
}
