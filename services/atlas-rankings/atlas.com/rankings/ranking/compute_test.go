package ranking

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func rankedById(rs []Ranked) map[uint32]Ranked {
	m := make(map[uint32]Ranked, len(rs))
	for _, r := range rs {
		m[r.CharacterId] = r
	}
	return m
}

func TestJobCategory(t *testing.T) {
	cases := []struct {
		jobId job.Id
		want  uint16
	}{
		{job.Id(0), 0},     // beginner
		{job.Id(100), 1},   // warrior
		{job.Id(112), 1},   // hero
		{job.Id(200), 2},   // magician
		{job.Id(312), 3},   // bowman 4th
		{job.Id(412), 4},   // thief 4th
		{job.Id(522), 5},   // pirate 4th
		{job.Id(1000), 10}, // noblesse
		{job.Id(1112), 11}, // dawn warrior 3rd
		{job.Id(2000), 20}, // aran beginner
		{job.Id(2112), 21}, // aran 4th
	}
	for _, c := range cases {
		if got := JobCategory(c.jobId); got != c.want {
			t.Errorf("JobCategory(%d) = %d, want %d", c.jobId, got, c.want)
		}
	}
}

func TestRankOrderingAndTiebreaks(t *testing.T) {
	// level DESC, experience DESC, characterId ASC
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 100, Level: 50, Experience: 100},
		{CharacterId: 2, WorldId: 0, JobId: 100, Level: 70, Experience: 5},
		{CharacterId: 3, WorldId: 0, JobId: 100, Level: 50, Experience: 200},
		{CharacterId: 4, WorldId: 0, JobId: 100, Level: 50, Experience: 100}, // ties char 1 on level+exp; id 4 > 1
	}
	got := rankedById(Rank(inputs))
	if got[2].OverallRank != 1 {
		t.Errorf("char 2 (highest level) rank = %d, want 1", got[2].OverallRank)
	}
	if got[3].OverallRank != 2 {
		t.Errorf("char 3 (level tie, more exp) rank = %d, want 2", got[3].OverallRank)
	}
	if got[1].OverallRank != 3 || got[4].OverallRank != 4 {
		t.Errorf("characterId ASC tiebreak violated: char1=%d char4=%d, want 3 and 4", got[1].OverallRank, got[4].OverallRank)
	}
}

func TestRankUniquePerWorld(t *testing.T) {
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 0, Level: 10, Experience: 0},
		{CharacterId: 2, WorldId: 0, JobId: 0, Level: 10, Experience: 0},
		{CharacterId: 3, WorldId: 1, JobId: 0, Level: 5, Experience: 0},
	}
	got := rankedById(Rank(inputs))
	if got[1].OverallRank == got[2].OverallRank {
		t.Errorf("ranks must be unique within a world (strict total order)")
	}
	if got[3].OverallRank != 1 {
		t.Errorf("worlds must rank independently: char 3 rank = %d, want 1", got[3].OverallRank)
	}
}

func TestJobRankRestrictedToCategory(t *testing.T) {
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 100, Level: 90, Experience: 0}, // warrior, overall 1
		{CharacterId: 2, WorldId: 0, JobId: 200, Level: 80, Experience: 0}, // magician, overall 2
		{CharacterId: 3, WorldId: 0, JobId: 110, Level: 70, Experience: 0}, // warrior, overall 3
	}
	got := rankedById(Rank(inputs))
	if got[1].JobRank != 1 || got[3].JobRank != 2 {
		t.Errorf("warrior job ranks = %d,%d, want 1,2", got[1].JobRank, got[3].JobRank)
	}
	if got[2].JobRank != 1 {
		t.Errorf("magician job rank = %d, want 1 (own category)", got[2].JobRank)
	}
	if got[1].JobCategory != 1 || got[2].JobCategory != 2 {
		t.Errorf("job categories wrong: %+v", got)
	}
}

func TestMove(t *testing.T) {
	cases := []struct {
		prev, next uint32
		want       int32
	}{
		{0, 5, 0},  // first-seen → 0
		{5, 3, 2},  // moved up
		{3, 5, -2}, // moved down
		{4, 4, 0},  // unchanged
	}
	for _, c := range cases {
		if got := Move(c.prev, c.next); got != c.want {
			t.Errorf("Move(%d,%d) = %d, want %d", c.prev, c.next, got, c.want)
		}
	}
}
