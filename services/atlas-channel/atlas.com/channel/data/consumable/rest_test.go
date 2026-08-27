package consumable

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. Spec is a map field (the codemod's SKIP reason); Transform must copy
// it rather than alias the source map into the RestModel.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id: 1,
		Spec: map[SpecType]int32{
			SpecTypeHP:         10,
			SpecTypeMPRecovery: 5,
		},
		Npc:         2430000,
		Script:      "test",
		RunOnPickup: true,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}

	// Spec must be a copy, not an alias of the source map.
	v, ok := rm2.Spec[SpecTypeHP]
	if !ok || v != 10 {
		t.Fatalf("expected copied Spec to contain SpecTypeHP=10, got %+v", rm2.Spec)
	}
	rm2.Spec[SpecTypeHP] = 999
	if val, _ := m.GetSpec(SpecTypeHP); val != 10 {
		t.Errorf("mutating Transform's Spec output mutated the Model's spec: got %d, want 10", val)
	}
	rm2.Spec[SpecTypeHP] = 10 // restore before final comparison

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
