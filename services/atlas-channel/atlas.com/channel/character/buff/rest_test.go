package buff

import (
	"atlas-channel/character/buff/stat"
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel (including the nested
// stat changes) survives a Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	changes := []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	m := NewBuff(1002, 3, 60, changes, createdAt, expiresAt, false)

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
