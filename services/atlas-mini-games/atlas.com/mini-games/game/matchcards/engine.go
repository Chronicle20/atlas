package matchcards

import "math/rand"

// MatchesToWin returns the number of matched pairs required to win for a
// given piece-set pieceType (Cosmic MiniGame difficulty tiers): 0->6, 1->10,
// 2->15. Any other pieceType is invalid and returns ok=false.
func MatchesToWin(pieceType byte) (byte, bool) {
	switch pieceType {
	case 0:
		return 6, true
	case 1:
		return 10, true
	case 2:
		return 15, true
	default:
		return 0, false
	}
}

// BuildDeck returns an unshuffled deck of `pairs` distinct ids, each
// appearing exactly twice: [0, 0, 1, 1, ..., pairs-1, pairs-1].
func BuildDeck(pairs byte) []uint32 {
	deck := make([]uint32, 0, int(pairs)*2)
	for id := uint32(0); id < uint32(pairs); id++ {
		deck = append(deck, id, id)
	}
	return deck
}

// Shuffle randomizes deck in place using the injected rand source (Fisher-
// Yates). No global rand and no time seeding: callers own determinism by
// supplying r.
func Shuffle(deck []uint32, r *rand.Rand) {
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

// FlipResultType maps a card flip to its result code, mirroring Cosmic
// PlayerInteractionHandler.java:460-484: a match yields 2 for the owner's
// flip and 3 for the visitor's flip; a mismatch yields 0 for the owner's
// flip and 1 for the visitor's flip.
func FlipResultType(ownerFlipped bool, match bool) byte {
	if match {
		if ownerFlipped {
			return 2
		}
		return 3
	}
	if ownerFlipped {
		return 0
	}
	return 1
}
