package movement

import (
	"testing"
)

func TestNarrowSkill_HappyPath(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(100, 2)
	if !ok || id != 100 || lvl != 2 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 100 2 true", id, lvl, ok)
	}
}

func TestNarrowSkill_NegativeRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(-1, 1); ok {
		t.Fatalf("expected reject for negative skillId")
	}
	if _, _, ok := narrowSkillBytes(1, -1); ok {
		t.Fatalf("expected reject for negative skillLevel")
	}
}

func TestNarrowSkill_OverflowRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(256, 1); ok {
		t.Fatalf("expected reject for skillId > 255")
	}
	if _, _, ok := narrowSkillBytes(1, 256); ok {
		t.Fatalf("expected reject for skillLevel > 255")
	}
}

func TestNarrowSkill_BoundaryAccepted(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(255, 255)
	if !ok || id != 255 || lvl != 255 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 255 255 true", id, lvl, ok)
	}
}
