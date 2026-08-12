package effect

import "testing"

// TestExtractRange pins that the channel decodes the WZ `range` attribute
// atlas-data already serves. Monster Magnet has no lt/rb, so `range` is the
// only WZ input to its server-side target region (task-215 design §3).
func TestExtractRange(t *testing.T) {
	m, err := Extract(RestModel{Range: 450, MobCount: 7})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.Range() != 450 {
		t.Fatalf("Range = %d, want 450", m.Range())
	}
	if m.MobCount() != 7 {
		t.Fatalf("MobCount = %d, want 7", m.MobCount())
	}
}

func TestExtractRangeAbsentIsZero(t *testing.T) {
	m, err := Extract(RestModel{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.Range() != 0 {
		t.Fatalf("Range = %d, want 0 when the attribute is absent", m.Range())
	}
}
