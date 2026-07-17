package ranking

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestBuilderRoundTrip(t *testing.T) {
	now := time.Now()
	m := NewBuilder().
		SetCharacterId(42).
		SetWorldId(world.Id(1)).
		SetJobCategory(2).
		SetOverallRank(17).
		SetOverallRankMove(2).
		SetJobRank(4).
		SetJobRankMove(-1).
		SetComputedAt(now).
		Build()

	if m.CharacterId() != 42 || m.WorldId() != world.Id(1) || m.JobCategory() != 2 {
		t.Fatalf("identity fields lost: %+v", m)
	}
	if m.OverallRank() != 17 || m.OverallRankMove() != 2 || m.JobRank() != 4 || m.JobRankMove() != -1 {
		t.Fatalf("rank fields lost: %+v", m)
	}
	if !m.ComputedAt().Equal(now) {
		t.Fatalf("computedAt lost")
	}
}

func TestMakeFromEntity(t *testing.T) {
	now := time.Now()
	e := Entity{CharacterId: 7, WorldId: world.Id(0), JobCategory: 21, OverallRank: 1, OverallRankMove: 0, JobRank: 1, JobRankMove: 3, ComputedAt: now}
	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}
	if m.CharacterId() != 7 || m.JobCategory() != 21 || m.JobRankMove() != 3 {
		t.Fatalf("Make lost fields: %+v", m)
	}
}
