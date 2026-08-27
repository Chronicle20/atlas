package character_test

import (
	"atlas-character/character"
	"testing"
)

func TestBuildPreservesHpMpUsed(t *testing.T) {
	m, err := character.NewEmptyBuilder().SetName("Atlas").SetHpMpUsed(7).Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}
	if m.HpMpUsed() != 7 {
		t.Fatalf("Build() dropped hpMpUsed: got %d, want 7", m.HpMpUsed())
	}
}

func TestCloneBuildRoundTripPreservesHpMpUsed(t *testing.T) {
	orig, err := character.NewEmptyBuilder().SetName("Atlas").SetHpMpUsed(3).Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}
	clone, err := character.CloneModel(orig).Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() returned unexpected error: %v", err)
	}
	if clone.HpMpUsed() != 3 {
		t.Fatalf("CloneModel().Build() dropped hpMpUsed: got %d, want 3", clone.HpMpUsed())
	}
}

// TestBuildErrorsWhenAccountIdSetWithoutName pins the derived reconstruction
// invariant (docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md):
// modelBuilder.Build() hydrates PARTIAL models across ~95 call sites (DB
// rows, REST Extract, kafka create commands, decorator rebuilds, test
// fixtures), so it cannot enforce the creation path's "accountId != 0 AND
// name != \"\"" invariant -- legitimate partials (e.g. a model built for
// HP/MP-gain resolution with only jobId and skills set) never carry an
// accountId or a name. The one thing every legitimate construction site
// agrees on: a model tied to a real account always carries a name too.
func TestBuildErrorsWhenAccountIdSetWithoutName(t *testing.T) {
	_, err := character.NewEmptyBuilder().SetAccountId(1000).Build()
	if err == nil {
		t.Fatalf("Build() should error when accountId is set without a name")
	}
}

// TestBuildSucceedsWithAccountIdAndName is the positive counterpart: the
// invariant only rejects the accountId-without-name combination, not
// accountId itself.
func TestBuildSucceedsWithAccountIdAndName(t *testing.T) {
	m, err := character.NewEmptyBuilder().SetAccountId(1000).SetName("Atlas").Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}
	if m.AccountId() != 1000 || m.Name() != "Atlas" {
		t.Fatalf("Build() dropped accountId/name: got accountId=%d name=%q", m.AccountId(), m.Name())
	}
}

// TestBuildSucceedsWithNeitherAccountIdNorName pins the partial-hydration
// case that constrains the invariant: character/hp_mp_gain_test.go builds a
// model setting only jobId and skills, with no accountId or name at all.
func TestBuildSucceedsWithNeitherAccountIdNorName(t *testing.T) {
	if _, err := character.NewEmptyBuilder().Build(); err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}
}
