package area

import (
	"atlas-reactors/reactor/data/point"
	"reflect"
	"testing"
)

// TestTransformRoundTrip verifies Transform is the faithful inverse of
// Extract: Extract(Transform(m)) reproduces m.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		TL: point.RestModel{X: -53, Y: 24},
		BR: point.RestModel{X: 62, Y: 69},
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
