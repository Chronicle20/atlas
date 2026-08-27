package portal

import (
	"reflect"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:          1001,
		name:        "sp",
		target:      "target field",
		portalType:  2,
		x:           10,
		y:           -20,
		targetMapId: _map.Id(100000000),
		scriptName:  "script01",
	}

	rm := Transform(m)
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
