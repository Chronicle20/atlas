package character_test

import (
	"testing"

	"atlas-character/character"
)

func TestBuildPreservesHpMpUsed(t *testing.T) {
	m := character.NewModelBuilder().SetName("Atlas").SetHpMpUsed(7).Build()
	if m.HpMpUsed() != 7 {
		t.Fatalf("Build() dropped hpMpUsed: got %d, want 7", m.HpMpUsed())
	}
}

func TestCloneBuildRoundTripPreservesHpMpUsed(t *testing.T) {
	orig := character.NewModelBuilder().SetName("Atlas").SetHpMpUsed(3).Build()
	clone := character.CloneModel(orig).Build()
	if clone.HpMpUsed() != 3 {
		t.Fatalf("CloneModel().Build() dropped hpMpUsed: got %d, want 3", clone.HpMpUsed())
	}
}
