package map_

import (
	"reflect"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: the flattened foothold map and the scalar fields Extract reads
// (Clock, ReturnMapId, FieldLimit, Town) survive a Transform -> Extract
// round trip. RestModel carries additional fields (Name, StreetName, etc.)
// that Extract never reads; those are not populated by Transform and are
// not asserted here.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		clock:       true,
		returnMapId: _map.Id(100000000),
		fieldLimit:  42,
		town:        true,
		footholds: map[uint32]Foothold{
			1: {Id: 1, FirstX: 0, FirstY: 100, SecondX: 200, SecondY: 100},
			2: {Id: 2, FirstX: 200, FirstY: 100, SecondX: 200, SecondY: 300},
		},
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
