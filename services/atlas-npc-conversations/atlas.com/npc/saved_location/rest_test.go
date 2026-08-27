package saved_location

import (
	"reflect"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransformRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetCharacterId(1001).
		SetLocationType("BOAT_DOCK").
		SetMapId(_map.Id(200000101)).
		SetPortalId(3).
		Build()

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
