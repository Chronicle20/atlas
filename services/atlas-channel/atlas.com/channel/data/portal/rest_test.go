package portal

import (
	"reflect"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel survives a
// Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:          5,
		name:        "sp",
		target:      "town",
		portalType:  2,
		x:           100,
		y:           -200,
		targetMapId: _map.Id(100000000),
		scriptName:  "script1",
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
