package state

import (
	"atlas-reactors/reactor/data/item"
	"reflect"
	"testing"
)

// TestTransformRoundTrip verifies Transform is the faithful inverse of
// Extract: Extract(Transform(m)) reproduces m.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Type:         1,
		ReactorItem:  &item.RestModel{ItemId: 2000000, Quantity: 5},
		ActiveSkills: []uint32{1000, 2000},
		NextState:    2,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (round trip) failed: %v", err)
	}

	if !reflect.DeepEqual(m2, m) {
		t.Errorf("round trip mismatch. want %+v, got %+v", m, m2)
	}
}
