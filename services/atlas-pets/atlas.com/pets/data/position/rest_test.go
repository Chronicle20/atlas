package position

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip asserts Transform and Extract are exact inverses
// of each other over the domain Model.
func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(42, 100, 200, 300, 400)

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
