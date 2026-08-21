package monster

import (
	"atlas-monsters/monster"
	"encoding/json"
	"testing"
)

// FR-P5: a producer that omits the provenance fields must produce byte-identical
// output to today. This pins the omitempty tags.
func TestSpawnFieldBodyOmitsProvenanceWhenUnset(t *testing.T) {
	b, err := json.Marshal(spawnFieldCommandBody{MonsterId: 8150000, X: 1, Y: 2, Fh: 3, Team: 0})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"monsterId":8150000,"x":1,"y":2,"fh":3,"team":0}`
	if string(b) != want {
		t.Fatalf("wire shape changed:\n got %s\nwant %s", b, want)
	}
}

func TestSpawnFieldBodyCarriesProvenanceWhenSet(t *testing.T) {
	b, err := json.Marshal(spawnFieldCommandBody{
		MonsterId: 8150000, SpawnSourceType: "EVENT", SpawnSourceId: "occ-1",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"monsterId":8150000,"x":0,"y":0,"fh":0,"team":0,"spawnSourceType":"EVENT","spawnSourceId":"occ-1"}`
	if string(b) != want {
		t.Fatalf("wire shape:\n got %s\nwant %s", b, want)
	}
}

// FR-P1: absent or empty normalizes to CYCLIC, once, at the boundary.
func TestNormalizeSpawnSourceType(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", monster.SpawnSourceTypeCyclic},
		{"EVENT", "EVENT"},
		{"GM", "GM"},
	} {
		if got := normalizeSpawnSourceType(tc.in); got != tc.want {
			t.Fatalf("normalizeSpawnSourceType(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
