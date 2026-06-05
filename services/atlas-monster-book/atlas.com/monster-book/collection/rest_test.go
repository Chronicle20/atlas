package collection

import "testing"

func TestTransformIncludesCoverMonsterId(t *testing.T) {
	m := NewModelBuilder().
		SetCharacterId(1).
		SetCoverCardId(2380000).
		SetCoverMobId(100100).
		MustBuild()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.CoverMonsterId != 100100 {
		t.Errorf("CoverMonsterId = %d, want 100100", rm.CoverMonsterId)
	}
	if uint32(rm.CoverCardId) != 2380000 {
		t.Errorf("CoverCardId = %d, want 2380000 (must remain card id)", rm.CoverCardId)
	}
}
