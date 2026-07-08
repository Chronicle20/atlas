package game_test

import (
	"testing"

	"atlas-rps/game"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func testLadder() game.Ladder {
	return game.Ladder{
		EntryCostMeso: 100,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(1000000), Quantity: 1, Meso: 0},
			{Rung: 2, ItemId: item.Id(2000000), Quantity: 2, Meso: 500},
			{Rung: 3, ItemId: item.Id(3000000), Quantity: 3, Meso: 1000},
		},
	}
}

func TestPrizeAtReturnsMatchingRung(t *testing.T) {
	l := testLadder()

	got, ok := l.PrizeAt(2)
	if !ok {
		t.Fatalf("PrizeAt(2) ok = false, want true")
	}
	if got.Rung != 2 || got.ItemId != item.Id(2000000) || got.Quantity != 2 || got.Meso != 500 {
		t.Errorf("PrizeAt(2) = %+v, want rung 2 prize", got)
	}
}

func TestPrizeAtZeroIsNoPrize(t *testing.T) {
	l := testLadder()

	_, ok := l.PrizeAt(0)
	if ok {
		t.Errorf("PrizeAt(0) ok = true, want false (fresh/no-prize)")
	}
}

func TestPrizeAtOutOfRangeIsNoPrize(t *testing.T) {
	l := testLadder()

	_, ok := l.PrizeAt(4)
	if ok {
		t.Errorf("PrizeAt(4) ok = true, want false (beyond max rung)")
	}
}

func TestPrizeAtEmptyLadderIsNoPrize(t *testing.T) {
	l := game.Ladder{}

	_, ok := l.PrizeAt(1)
	if ok {
		t.Errorf("PrizeAt(1) on empty ladder ok = true, want false")
	}
}

func TestMaxRung(t *testing.T) {
	l := testLadder()

	if got := l.MaxRung(); got != 3 {
		t.Errorf("MaxRung() = %d, want 3", got)
	}
}

func TestMaxRungEmptyLadder(t *testing.T) {
	l := game.Ladder{}

	if got := l.MaxRung(); got != 0 {
		t.Errorf("MaxRung() on empty ladder = %d, want 0", got)
	}
}

func TestIsMax(t *testing.T) {
	l := testLadder()

	if !l.IsMax(3) {
		t.Errorf("IsMax(3) = false, want true")
	}
	if l.IsMax(2) {
		t.Errorf("IsMax(2) = true, want false")
	}
	if l.IsMax(0) {
		t.Errorf("IsMax(0) = true, want false")
	}
}
