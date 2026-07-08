package game

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Rung represents a single step of the reward ladder. Rung indices are
// 1-based; rung 0 is reserved to mean "fresh, no prize" and never appears
// in a Ladder's Rungs slice.
type Rung struct {
	Rung     int
	ItemId   item.Id
	Quantity uint32
	Meso     uint32
}

// Ladder is the ordered reward progression for an RPS NPC. Rungs is expected
// to be ordered by ascending Rung number, but PrizeAt resolves by matching
// the Rung field rather than by slice position, so gaps are tolerated.
type Ladder struct {
	EntryCostMeso uint32
	Rungs         []Rung
}

// PrizeAt resolves the prize for a given 1-based rung. Rung 0 (fresh, no
// prize) and any rung beyond the ladder's configured rungs both resolve to
// ok=false.
func (l Ladder) PrizeAt(rung int) (Rung, bool) {
	if rung <= 0 {
		return Rung{}, false
	}
	for _, r := range l.Rungs {
		if r.Rung == rung {
			return r, true
		}
	}
	return Rung{}, false
}

// MaxRung returns the highest rung number configured on the ladder, or 0 if
// the ladder has no rungs.
func (l Ladder) MaxRung() int {
	max := 0
	for _, r := range l.Rungs {
		if r.Rung > max {
			max = r.Rung
		}
	}
	return max
}

// IsMax reports whether the given rung is the ladder's highest configured
// rung (i.e. no further prizes remain).
func (l Ladder) IsMax(rung int) bool {
	return rung == l.MaxRung() && rung != 0
}
