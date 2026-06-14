package mount

import "math"

// CAP is the maximum reachable mount level. A mount may level up only while
// its current level is strictly below CAP.
const CAP = 31

// mountExp is the per-level exp requirement table (index = level). Values are
// pinned from HeavenMS (context.md §8.2) and must not be changed.
//
// NOTE: the table has only 29 entries (valid indices 0..28) while CAP=31
// allows levels up to 30. Levels at or past len(mountExp) have no defined
// requirement; ExpNeededForLevel returns a sentinel for those so the level-up
// loop terminates instead of indexing out of bounds.
var mountExp = []int32{
	1, 24, 50, 105, 134, 196, 254, 263, 315, 367,
	430, 543, 587, 679, 725, 897, 1146, 1394, 1701, 2247,
	2543, 2898, 3156, 3313, 3584, 3923, 4150, 4305, 4550,
}

// ExpNeededForLevel returns the exp required to advance from the given level to
// the next. For levels outside the pinned table (negative, or >= len(mountExp))
// it returns math.MaxInt32, a sentinel that no attainable exp total can reach,
// causing the feed level-up loop to stop without an out-of-bounds access.
func ExpNeededForLevel(level int) int32 {
	if level < 0 || level >= len(mountExp) {
		return math.MaxInt32
	}
	return mountExp[level]
}

// FeedInput is the immutable input to ApplyFeed. HealMax is supplied by the
// caller (sourced from the feed event / atlas-consumables config) and is never
// hardcoded here.
type FeedInput struct {
	Level     int
	Exp       int
	Tiredness int
	HealMax   int
}

// FeedResult is the computed outcome of a feed action: updated progression plus
// whether at least one level-up occurred.
type FeedResult struct {
	Level     int
	Exp       int
	Tiredness int
	LevelUp   bool
}

// ApplyFeed computes the heal → exp → level-up progression for a single feed,
// matching Cosmic's UseMountFoodHandler exactly:
//
//	heal      = min(tiredness, healMax)
//	tiredness = tiredness - heal
//	exp      += ceil((heal / healMax) * (2*level + 6))    // float division
//	if level < CAP && exp >= ExpNeededForLevel(level):     // cumulative threshold
//	    level++; LevelUp = true                             // exp is NOT reset
//
// Exp is a cumulative running total (never decremented) and the mount gains at
// most ONE level per feed — Cosmic levels via a single `if`, not a loop, and does
// not subtract the threshold. The earlier subtract-and-loop form diverged: it
// required the SUM of thresholds to reach a level instead of the cumulative
// threshold mount[level], so leveling was far too slow. Pure: no I/O, no input mutation.
func ApplyFeed(in FeedInput) FeedResult {
	level := in.Level
	exp := in.Exp
	tiredness := in.Tiredness

	heal := tiredness
	if in.HealMax < heal {
		heal = in.HealMax
	}
	tiredness -= heal

	if in.HealMax > 0 && heal > 0 {
		gain := math.Ceil((float64(heal) / float64(in.HealMax)) * float64(2*level+6))
		exp += int(gain)
	}

	levelUp := false
	if level < CAP && exp >= int(ExpNeededForLevel(level)) {
		level++
		levelUp = true
	}

	return FeedResult{
		Level:     level,
		Exp:       exp,
		Tiredness: tiredness,
		LevelUp:   levelUp,
	}
}
