package matchcards

import (
	"math/rand"
	"testing"
)

func TestMatchesToWin_Mapping(t *testing.T) {
	cases := []struct {
		pieceType byte
		want      byte
	}{
		{0, 6},
		{1, 10},
		{2, 15},
	}
	for _, c := range cases {
		got, ok := MatchesToWin(c.pieceType)
		if !ok {
			t.Errorf("MatchesToWin(%d) = (_, false), want (%d, true)", c.pieceType, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("MatchesToWin(%d) = %d, want %d", c.pieceType, got, c.want)
		}
	}
}

func TestMatchesToWin_Invalid(t *testing.T) {
	for _, pieceType := range []byte{3, 4, 255} {
		if got, ok := MatchesToWin(pieceType); ok {
			t.Errorf("MatchesToWin(%d) = (%d, true), want (_, false)", pieceType, got)
		}
	}
}

func TestBuildDeck_Length(t *testing.T) {
	for _, pairs := range []byte{0, 1, 6, 15} {
		deck := BuildDeck(pairs)
		if len(deck) != int(pairs)*2 {
			t.Errorf("len(BuildDeck(%d)) = %d, want %d", pairs, len(deck), int(pairs)*2)
		}
	}
}

func TestBuildDeck_EachIdExactlyTwice(t *testing.T) {
	const pairs = 15
	deck := BuildDeck(pairs)
	counts := make(map[uint32]int)
	for _, id := range deck {
		counts[id]++
	}
	if len(counts) != pairs {
		t.Fatalf("BuildDeck(%d) produced %d distinct ids, want %d", pairs, len(counts), pairs)
	}
	for id := uint32(0); id < pairs; id++ {
		if counts[id] != 2 {
			t.Errorf("BuildDeck(%d) id %d appears %d times, want 2", pairs, id, counts[id])
		}
	}
}

func TestBuildDeck_Unshuffled(t *testing.T) {
	// Per the interface contract, BuildDeck returns ids 0..pairs-1 each twice,
	// unshuffled: id 0 and id 1 occupy the first two slots, in ascending order.
	deck := BuildDeck(3)
	want := []uint32{0, 0, 1, 1, 2, 2}
	if len(deck) != len(want) {
		t.Fatalf("len(deck) = %d, want %d", len(deck), len(want))
	}
	for i := range want {
		if deck[i] != want[i] {
			t.Errorf("deck[%d] = %d, want %d", i, deck[i], want[i])
		}
	}
}

func TestShuffle_DeterministicWithSeededRand(t *testing.T) {
	deck1 := BuildDeck(15)
	deck2 := BuildDeck(15)

	Shuffle(deck1, rand.New(rand.NewSource(42)))
	Shuffle(deck2, rand.New(rand.NewSource(42)))

	for i := range deck1 {
		if deck1[i] != deck2[i] {
			t.Fatalf("same seed produced different order at index %d: %d != %d", i, deck1[i], deck2[i])
		}
	}
}

func TestShuffle_PreservesMultiset(t *testing.T) {
	deck := BuildDeck(15)
	before := make(map[uint32]int)
	for _, id := range deck {
		before[id]++
	}

	Shuffle(deck, rand.New(rand.NewSource(7)))

	after := make(map[uint32]int)
	for _, id := range deck {
		after[id]++
	}

	if len(before) != len(after) {
		t.Fatalf("Shuffle changed distinct id count: before %d, after %d", len(before), len(after))
	}
	for id, count := range before {
		if after[id] != count {
			t.Errorf("Shuffle changed count of id %d: before %d, after %d", id, count, after[id])
		}
	}
}

func TestShuffle_DifferentSeedsCanDiffer(t *testing.T) {
	deck1 := BuildDeck(15)
	deck2 := BuildDeck(15)

	Shuffle(deck1, rand.New(rand.NewSource(1)))
	Shuffle(deck2, rand.New(rand.NewSource(2)))

	same := true
	for i := range deck1 {
		if deck1[i] != deck2[i] {
			same = false
			break
		}
	}
	if same {
		t.Errorf("Shuffle with different seeds produced identical order; test is not discriminating")
	}
}

// FlipResultType truth table per Cosmic PlayerInteractionHandler.java:460-484:
// owner match -> 2, visitor match -> 3, owner mismatch -> 0, visitor mismatch -> 1.
func TestFlipResultType_TruthTable(t *testing.T) {
	cases := []struct {
		name         string
		ownerFlipped bool
		match        bool
		want         byte
	}{
		{"owner match", true, true, 2},
		{"visitor match", false, true, 3},
		{"owner mismatch", true, false, 0},
		{"visitor mismatch", false, false, 1},
	}
	for _, c := range cases {
		got := FlipResultType(c.ownerFlipped, c.match)
		if got != c.want {
			t.Errorf("%s: FlipResultType(%v, %v) = %d, want %d", c.name, c.ownerFlipped, c.match, got, c.want)
		}
	}
}
