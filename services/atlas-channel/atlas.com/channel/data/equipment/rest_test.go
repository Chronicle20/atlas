package equipment

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. PetAbilities is a []string field (the codemod's SKIP reason);
// Transform must copy it rather than alias the source slice into the
// RestModel.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:           1,
		PetAbilities: []string{"consumeHP", "sweepForDrop"},
		NotExtend:    true,
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

	// PetAbilities must be a copy, not an alias of the source slice.
	if len(rm2.PetAbilities) > 0 {
		rm2.PetAbilities[0] = "mutated"
	}
	if got := m.PetAbilities()[0]; got != "consumeHP" {
		t.Errorf("mutating Transform's PetAbilities output mutated the Model's slice: got %q, want %q", got, "consumeHP")
	}
	rm2.PetAbilities[0] = "consumeHP" // restore before final comparison

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
