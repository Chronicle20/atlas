package pet

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(7, 3, 5000029, 250)

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
