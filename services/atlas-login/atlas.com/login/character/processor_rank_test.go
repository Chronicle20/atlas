package character

import (
	"context"
	"errors"
	"testing"

	"atlas-login/ranking"

	"github.com/sirupsen/logrus"
)

func rankingModel(t *testing.T, characterId uint32, rank uint32, rankMove int32, jobRank uint32, jobRankMove int32) ranking.Model {
	t.Helper()
	rm := ranking.RestModel{Rank: rank, RankMove: rankMove, JobRank: jobRank, JobRankMove: jobRankMove}
	rm.Id = characterId
	m, err := ranking.Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// TestMergeRankings covers the happy path (a ranked character gets its real
// values merged, with a negative move surviving the round trip) and the
// partially-present bulk response case (an unranked character keeps zeros).
// Fixture values for rank/rankMove/jobRank/jobRankMove are all distinct so a
// field swap in MergeRankings or ranking.Extract would be caught.
func TestMergeRankings(t *testing.T) {
	cs := []Model{
		NewBuilder().SetId(1).SetName("A").Build(),
		NewBuilder().SetId(2).SetName("B").Build(),
	}
	rs := []ranking.Model{rankingModel(t, 1, 17, -2, 41, -9)}

	got := MergeRankings(cs, rs)
	if len(got) != 2 {
		t.Fatalf("merge changed slice length: %d", len(got))
	}

	// Character 1: ranked — real values must land in the matching fields,
	// with the negative move surviving the round trip. character.Model's
	// RankMove()/JobRankMove() getters return the two's-complement uint32
	// used by the packet wire (see character/model.go), so -2 -> 0xFFFFFFFE
	// and -9 -> 0xFFFFFFF7.
	if got[0].Rank() != 17 {
		t.Fatalf("char 1 rank not decorated: got %d want 17", got[0].Rank())
	}
	if got[0].RankMove() != uint32(0xFFFFFFFE) {
		t.Fatalf("char 1 rankMove sign lost in round-trip: got %#x want 0xFFFFFFFE", got[0].RankMove())
	}
	if got[0].JobRank() != 41 {
		t.Fatalf("char 1 jobRank not decorated: got %d want 41", got[0].JobRank())
	}
	if got[0].JobRankMove() != uint32(0xFFFFFFF7) {
		t.Fatalf("char 1 jobRankMove sign lost in round-trip: got %#x want 0xFFFFFFF7", got[0].JobRankMove())
	}
	if got[0].Name() != "A" {
		t.Fatalf("merge dropped unrelated fields: %+v", got[0])
	}

	// Character 2: absent from the bulk response — must keep zero ranks,
	// not error, not get skipped from the output slice.
	if got[1].Rank() != 0 || got[1].RankMove() != 0 || got[1].JobRank() != 0 || got[1].JobRankMove() != 0 {
		t.Fatalf("char 2 without entry must stay zero: %+v", got[1])
	}
	if got[1].Name() != "B" {
		t.Fatalf("merge dropped unrelated fields for unranked char: %+v", got[1])
	}
}

// TestDecorateRankingsFailsOpen proves the fail-open contract genuinely
// holds: when the rankings dependency errors (standing in for a network
// error or a 2s timeout), decorateRankings returns the original character
// list — full length, zero ranks, no error propagated.
func TestDecorateRankingsFailsOpen(t *testing.T) {
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		rankings: func(ids []uint32) ([]ranking.Model, error) {
			return nil, errors.New("rankings unavailable")
		},
	}
	cs := []Model{
		NewBuilder().SetId(1).SetName("A").Build(),
		NewBuilder().SetId(2).SetName("B").Build(),
	}

	got := p.decorateRankings(cs)
	if len(got) != 2 {
		t.Fatalf("fail-open must return the full original list: got %d characters", len(got))
	}
	for i, c := range got {
		if c.Rank() != 0 || c.RankMove() != 0 || c.JobRank() != 0 || c.JobRankMove() != 0 {
			t.Fatalf("fail-open must return zero ranks for character %d: %+v", i, c)
		}
	}
}
