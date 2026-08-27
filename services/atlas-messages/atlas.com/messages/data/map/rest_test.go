package _map

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract. Model in this package is an intentionally empty struct{} (see
// the Transform doc comment): atlas-messages never needs any map field
// data, only existence, so both Extract and Transform are legitimately
// no-ops and the round trip is over empty values.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{}

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
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot      %+v", m, m2)
	}
}
