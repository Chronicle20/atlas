package portal

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:          42,
		name:        "town_portal",
		target:      "spawn_point",
		portalType:  2,
		x:           150,
		y:           -50,
		targetMapId: 100000000,
		scriptName:  "portal_script",
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
