package template

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel survives a
// Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:        1002001,
		name:      "Maple Admin",
		trunkPut:  100,
		trunkGet:  200,
		storebank: true,
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
