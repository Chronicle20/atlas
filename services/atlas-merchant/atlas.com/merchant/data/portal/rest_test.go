package portal

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip asserts Extract(Transform(m)) == m. RestModel.Target
// and RestModel.ScriptName have no Model counterpart and are never read by
// Extract (rest.go), so Transform does not populate them; that is not a
// lossy round trip in the resolution-#4 sense since no Model field is
// dropped.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:          1,
		name:        "portalName",
		portalType:  2,
		x:           3,
		y:           4,
		targetMapId: 5,
	}

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
