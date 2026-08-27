package cash

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestRestModelSpecRoundTripsMorphKeys pins FR-3.1: the three spec keys a
// transformation coupon needs survive the atlas-data REST hop. The literal JSON
// below is the shape atlas-data emits for 5300000 (PRD §5); if a constant here
// drifts from atlas-data's string value the spec silently reads as zero at
// runtime and the coupon becomes an inert consume, so this test asserts the
// wire strings, not just the Go identifiers.
func TestRestModelSpecRoundTripsMorphKeys(t *testing.T) {
	const body = `{"slotMax":200,"spec":{"morph":1,"hp":50,"time":600000}}`

	var rm RestModel
	if err := json.Unmarshal([]byte(body), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	for _, tc := range []struct {
		spec SpecType
		want int32
	}{
		{SpecTypeMorph, 1},
		{SpecTypeHp, 50},
		{SpecTypeTime, 600000},
	} {
		got, ok := m.GetSpec(tc.spec)
		if !ok {
			t.Fatalf("GetSpec(%q) missing", tc.spec)
		}
		if got != tc.want {
			t.Errorf("GetSpec(%q) = %d, want %d", tc.spec, got, tc.want)
		}
	}

	if m.slotMax != 200 {
		t.Errorf("slotMax = %d, want 200", m.slotMax)
	}
}

// TestSpecTypeWireValues pins the exact JSON keys against atlas-data's
// (services/atlas-data/atlas.com/data/cash/rest.go). These two SpecType sets
// live in separate Go modules, so a rename in one and not the other fails no
// build — it decodes into a zero-valued spec, silently.
func TestSpecTypeWireValues(t *testing.T) {
	for _, tc := range []struct {
		spec SpecType
		want string
	}{
		{SpecTypeMorph, "morph"},
		{SpecTypeHp, "hp"},
		{SpecTypeTime, "time"},
	} {
		if string(tc.spec) != tc.want {
			t.Errorf("SpecType = %q, want %q", tc.spec, tc.want)
		}
	}
}

// TestGetSpecAbsentKey pins the negative half FR-3.7 depends on: an absent key
// reports ok=false rather than a zero value indistinguishable from a real zero.
func TestGetSpecAbsentKey(t *testing.T) {
	m, err := Extract(RestModel{Spec: map[SpecType]int32{SpecTypeTime: 600000}})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if v, ok := m.GetSpec(SpecTypeMorph); ok {
		t.Errorf("GetSpec(morph) = (%d, true), want ok=false when the key is absent", v)
	}
}

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. Spec (map[SpecType]int32) is the codemod's SKIP reason; PetSkills
// ([]string) is another reference-typed field. Both must be copies, not
// aliases of the source Model's backing storage.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:      1,
		SlotMax: 200,
		Spec: map[SpecType]int32{
			SpecTypeMorph: 1,
			SpecTypeTime:  60000,
		},
		PetSkills:   []string{"skillA", "skillB"},
		PetSkillAdd: true,
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
	v, ok := rm2.Spec[SpecTypeTime]
	if !ok || v != 60000 {
		t.Fatalf("expected copied Spec to contain SpecTypeTime=60000, got %+v", rm2.Spec)
	}
	rm2.Spec[SpecTypeTime] = 999
	if val, _ := m.GetSpec(SpecTypeTime); val != 60000 {
		t.Errorf("mutating Transform's Spec output mutated the Model's spec: got %d, want 60000", val)
	}
	rm2.Spec[SpecTypeTime] = 60000 // restore before final comparison

	// PetSkills must be a copy, not an alias of the source slice.
	if len(rm2.PetSkills) != 2 || rm2.PetSkills[0] != "skillA" {
		t.Fatalf("expected copied PetSkills, got %+v", rm2.PetSkills)
	}
	rm2.PetSkills[0] = "mutated"
	if got := m.PetSkills(); got[0] != "skillA" {
		t.Errorf("mutating Transform's PetSkills output mutated the Model's petSkills: got %v, want skillA", got[0])
	}
	rm2.PetSkills[0] = "skillA" // restore before final comparison

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
