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
	e := Entity{
		CharacterId:     7,
		WorldId:         world.Id(3),
		JobCategory:     21,
		OverallRank:     11,
		OverallRankMove: 5,
		JobRank:         13,
		JobRankMove:     -9,
		ComputedAt:      now,
	}
	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}
	if m.CharacterId() != 7 {
		t.Fatalf("CharacterId lost: %+v", m)
	}
	if m.WorldId() != world.Id(3) {
		t.Fatalf("WorldId lost: %+v", m)
	}
	if m.JobCategory() != 21 {
		t.Fatalf("JobCategory lost: %+v", m)
	}
	if m.OverallRank() != 11 {
		t.Fatalf("OverallRank lost: %+v", m)
	}
	if m.OverallRankMove() != 5 {
		t.Fatalf("OverallRankMove lost: %+v", m)
	}
	if m.JobRank() != 13 {
		t.Fatalf("JobRank lost: %+v", m)
	}
	if m.JobRankMove() != -9 {
		t.Fatalf("JobRankMove lost: %+v", m)
	}
	if !m.ComputedAt().Equal(now) {
		t.Fatalf("ComputedAt lost: %+v", m)
	}
}

func TestMakeCycleFromEntity_NonNilLastCompletedAt(t *testing.T) {
	started := time.Now().Add(-2 * time.Hour)
	completed := time.Now().Add(-1 * time.Hour)
	e := CycleEntity{
		LastStartedAt:    started,
		LastCompletedAt:  &completed,
		CharactersRanked: 250,
		DurationMs:       4321,
	}
	m, err := MakeCycle(e)
	if err != nil {
		t.Fatalf("MakeCycle failed: %v", err)
	}
	if !m.LastStartedAt().Equal(started) {
		t.Fatalf("LastStartedAt lost: %+v", m)
	}
	if m.LastCompletedAt() == nil {
		t.Fatalf("LastCompletedAt lost: got nil, want %v", completed)
	}
	if !m.LastCompletedAt().Equal(completed) {
		t.Fatalf("LastCompletedAt lost: %+v", m)
	}
	if m.CharactersRanked() != 250 {
		t.Fatalf("CharactersRanked lost: %+v", m)
	}
	if m.DurationMs() != 4321 {
		t.Fatalf("DurationMs lost: %+v", m)
	}
}

func TestMakeCycleFromEntity_NilLastCompletedAt(t *testing.T) {
	started := time.Now().Add(-3 * time.Hour)
	e := CycleEntity{
		LastStartedAt:    started,
		LastCompletedAt:  nil,
		CharactersRanked: 17,
		DurationMs:       8642,
	}
	m, err := MakeCycle(e)
	if err != nil {
		t.Fatalf("MakeCycle failed: %v", err)
	}
	if !m.LastStartedAt().Equal(started) {
		t.Fatalf("LastStartedAt lost: %+v", m)
	}
	if m.LastCompletedAt() != nil {
		t.Fatalf("LastCompletedAt should remain nil: got %v", m.LastCompletedAt())
	}
	if m.CharactersRanked() != 17 {
		t.Fatalf("CharactersRanked lost: %+v", m)
	}
	if m.DurationMs() != 8642 {
		t.Fatalf("DurationMs lost: %+v", m)
	}
}
