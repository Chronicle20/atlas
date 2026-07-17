package character

import "testing"

func TestRankBuilderRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetRank(17).
		SetRankMove(2).
		SetJobRank(4).
		SetJobRankMove(-1).
		Build()

	if m.Rank() != 17 || m.JobRank() != 4 {
		t.Fatalf("ranks lost: rank=%d jobRank=%d", m.Rank(), m.JobRank())
	}
	if m.RankMove() != 2 {
		t.Fatalf("rankMove = %d, want 2", m.RankMove())
	}

	// The packet field is uint32; the v83 client reinterprets it signed
	// (abs + sign branch). -1 must pass through as two's complement.
	if m.JobRankMove() != 0xFFFFFFFF {
		t.Fatalf("jobRankMove = %#x, want 0xFFFFFFFF", m.JobRankMove())
	}

	rt := m.ToBuilder().Build()
	if rt.Rank() != 17 || rt.RankMove() != 2 || rt.JobRank() != 4 || rt.JobRankMove() != 0xFFFFFFFF {
		t.Fatalf("ToBuilder dropped rank fields: %+v", rt)
	}
}

func TestRankDefaultsToZero(t *testing.T) {
	m := NewBuilder().SetId(1).Build()
	if m.Rank() != 0 || m.RankMove() != 0 || m.JobRank() != 0 || m.JobRankMove() != 0 {
		t.Fatal("unranked character must render all-zero rank fields")
	}
}
