package skill

import (
	"testing"

	"atlas-doors/data/skill/effect"
)

// TestGetEffectLevelIndexing verifies that Extract correctly maps per-level
// effects and that the 1-based level index (Effects()[level-1]) returns the
// right effect without any network I/O.
func TestGetEffectLevelIndexing(t *testing.T) {
	// Build a fixed RestModel with two levels: level-1 has Duration=30000 (30s),
	// level-2 has Duration=60000 (60s).
	rm := RestModel{
		Id: 2311001,
		Effects: []effect.RestModel{
			{Duration: 30000, MPConsume: 10, ItemConsume: 4006000},
			{Duration: 60000, MPConsume: 20, ItemConsume: 0},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(m.Effects()) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(m.Effects()))
	}

	// Level 1 (1-based) → Effects()[0]
	lvl1 := m.Effects()[0]
	if lvl1.Duration() != 30000 {
		t.Errorf("level-1 Duration: want 30000, got %d", lvl1.Duration())
	}
	if lvl1.MPConsume() != 10 {
		t.Errorf("level-1 MPConsume: want 10, got %d", lvl1.MPConsume())
	}
	if lvl1.ItemConsume() != 4006000 {
		t.Errorf("level-1 ItemConsume: want 4006000, got %d", lvl1.ItemConsume())
	}

	// Level 2 (1-based) → Effects()[1]
	lvl2 := m.Effects()[1]
	if lvl2.Duration() != 60000 {
		t.Errorf("level-2 Duration: want 60000, got %d", lvl2.Duration())
	}
	if lvl2.MPConsume() != 20 {
		t.Errorf("level-2 MPConsume: want 20, got %d", lvl2.MPConsume())
	}
}

// TestGetEffectDurationSentinel verifies the -1 "no duration" sentinel is
// preserved through Extract.
func TestGetEffectDurationSentinel(t *testing.T) {
	rm := RestModel{
		Id: 2311002,
		Effects: []effect.RestModel{
			{Duration: -1, MPConsume: 5, ItemConsume: 0},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if m.Effects()[0].Duration() != -1 {
		t.Errorf("expected -1 sentinel, got %d", m.Effects()[0].Duration())
	}
}
