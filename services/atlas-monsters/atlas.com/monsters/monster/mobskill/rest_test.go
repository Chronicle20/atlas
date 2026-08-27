package mobskill

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives an Extract -> Transform ->
// Extract round trip, with distinct non-zero values for every field.
// Summons ([]uint32) is the codemod's SKIP reason and must be a copy, not an
// alias, of the source Model's backing slice.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		SkillId:      1,
		Level:        2,
		MpCon:        3,
		Duration:     4,
		Hp:           5,
		X:            6,
		Y:            7,
		Prop:         8,
		Interval:     9,
		Count:        10,
		Limit:        11,
		LtX:          12,
		LtY:          13,
		RbX:          14,
		RbY:          15,
		SummonEffect: 16,
		Summons:      []uint32{100, 200},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.SkillId != rm.SkillId || rm2.Level != rm.Level {
		t.Errorf("id mismatch. Expected (%d,%d), got (%d,%d)", rm.SkillId, rm.Level, rm2.SkillId, rm2.Level)
	}

	// Summons must be a copy, not an alias of the source Model's slice.
	if len(rm2.Summons) != 2 || rm2.Summons[0] != 100 {
		t.Fatalf("expected copied Summons, got %+v", rm2.Summons)
	}
	rm2.Summons[0] = 999
	if got := m.Summons(); got[0] != 100 {
		t.Errorf("mutating Transform's Summons output mutated the Model's summons: got %v, want 100", got[0])
	}
	rm2.Summons[0] = 100 // restore before final comparison

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot      %+v", m, m2)
	}
}
