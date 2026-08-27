package character

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:         11,
		x:          22,
		y:          33,
		stance:     44,
		spawnPoint: 55,
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if got.SpawnPoint() != 55 {
		t.Errorf("SpawnPoint() = %d, want 55", got.SpawnPoint())
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}

// TestExtract_SpawnPoint asserts the inbound seam directly. The round-trip
// test above cannot prove it alone: a field dropped by both Extract and
// Transform is zero on both sides of a DeepEqual and hides in plain sight.
func TestExtract_SpawnPoint(t *testing.T) {
	rm := RestModel{Id: 11, SpawnPoint: 55}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if got := m.SpawnPoint(); got != 55 {
		t.Errorf("SpawnPoint() = %d, want 55", got)
	}
}
