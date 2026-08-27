package rates

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip asserts Transform and Extract are exact inverses
// of each other over the domain Model.
func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(1.5, 2.5, 3.5, 4.5)

	body := Transform(m)
	got := Extract(body)
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
