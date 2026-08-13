package cash

import (
	"encoding/json"
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
