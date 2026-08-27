package drop

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip asserts Transform and Extract are exact inverses
// of each other over the domain Model.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		reactorId: 10,
		itemId:    20,
		questId:   30,
		chance:    40,
	}

	rm := Transform(m)
	got := Extract(rm)
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
