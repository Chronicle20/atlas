package pet

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(100, 2, "Fido")

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot %+v", m, got)
	}
}
