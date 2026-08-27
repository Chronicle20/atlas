package teleportrock

import (
	"reflect"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: the regular and vip map lists survive a Transform -> Extract
// round trip. RestModel.Id is not read by Extract (json:"-" resource id,
// carries no Model field) and is not asserted here.
func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(
		[]_map.Id{1, 2, 3},
		[]_map.Id{4, 5},
	)

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
