package character

import (
	"fmt"
)

// RebalanceTarget is the character-domain input to RebalanceAP: a primary stat
// name and the floor value it should be raised to. Callers (the kafka consumer)
// translate wire-level types into this local type before invoking the processor.
type RebalanceTarget struct {
	Stat  string
	Floor uint16
}

const (
	statStrength     = "strength"
	statDexterity    = "dexterity"
	statIntelligence = "intelligence"
	statLuck         = "luck"
)

// rebalanceResult is the output of computeRebalance: the new primary stat values
// and the new unallocated AP total. All values are post-rebalance.
type rebalanceResult struct {
	Str, Dex, Int, Luk uint16
	Unallocated        uint16
}

// computeRebalance implements the first-job AP rebalance algorithm.
//  1. reclaimed = Σ max(0, stat - 4) over the four primary stats.
//  2. All four primaries reset to 4.
//  3. For each target, the corresponding stat is raised to target.Floor.
//  4. cost = Σ (target.Floor - 4) across targets.
//  5. newUnallocated = unallocated + reclaimed - cost.
//
// Returns an error if newUnallocated would be negative. Callers must ensure targets
// contain no duplicate stats — the helper trusts that invariant.
func computeRebalance(str, dex, in_, luk, unallocated uint16, targets []RebalanceTarget) (rebalanceResult, error) {
	const base uint16 = 4

	reclaimed := uint32(0)
	if str > base {
		reclaimed += uint32(str - base)
	}
	if dex > base {
		reclaimed += uint32(dex - base)
	}
	if in_ > base {
		reclaimed += uint32(in_ - base)
	}
	if luk > base {
		reclaimed += uint32(luk - base)
	}

	result := rebalanceResult{Str: base, Dex: base, Int: base, Luk: base}

	cost := uint32(0)
	for _, t := range targets {
		if t.Floor < base {
			return rebalanceResult{}, fmt.Errorf("rebalance target floor %d is below base %d", t.Floor, base)
		}
		cost += uint32(t.Floor - base)
		switch t.Stat {
		case statStrength:
			result.Str = t.Floor
		case statDexterity:
			result.Dex = t.Floor
		case statIntelligence:
			result.Int = t.Floor
		case statLuck:
			result.Luk = t.Floor
		default:
			return rebalanceResult{}, fmt.Errorf("unknown rebalance stat %q", t.Stat)
		}
	}

	available := uint32(unallocated) + reclaimed
	if cost > available {
		return rebalanceResult{}, fmt.Errorf("insufficient AP for rebalance: need %d, have %d", cost, available)
	}
	result.Unallocated = uint16(available - cost)
	return result, nil
}
