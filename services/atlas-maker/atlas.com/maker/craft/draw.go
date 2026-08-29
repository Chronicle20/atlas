package craft

import (
	"atlas-maker/recipe"
	"crypto/rand"
	"errors"
	"math/big"
)

// ErrEmptyRewardPool is returned when Draw is called against a recipe with
// no random-reward entries. It is distinct from a zero-value recipe.Reward
// so callers can tell "nothing to draw from" apart from a valid draw.
var ErrEmptyRewardPool = errors.New("recipe has no random rewards to draw from")

// totalWeight sums Prob across the reward pool. A pool where every entry is
// 0 sums to 0, which Draw treats as a request to fall back to a uniform
// pick across the pool.
func totalWeight(pool []recipe.Reward) uint32 {
	var total uint32
	for _, r := range pool {
		total += r.Prob
	}
	return total
}

// selectWeightedIndex is a pure helper: given a pool and a roll drawn from
// [0, totalWeight(pool)), it returns the index of the reward whose
// cumulative-weight range contains roll. A zero-weight reward contributes a
// zero-width range and can never be selected by any roll in range. Extracted
// as a pure function so the weighting boundary can be tested deterministically
// without stubbing crypto/rand.
func selectWeightedIndex(pool []recipe.Reward, roll uint32) int {
	var cumulative uint32
	for i, r := range pool {
		cumulative += r.Prob
		if roll < cumulative {
			return i
		}
	}
	// Unreachable when roll is drawn from [0, totalWeight(pool)); guards
	// against float/overflow surprises by returning the last item rather
	// than panicking.
	return len(pool) - 1
}

// Draw picks one reward from rewards by weight (recipe.Reward.Prob), using
// crypto/rand rather than math/rand because the draw decides item value and
// a predictable stream is exploitable (PRD §8 Randomness). When every entry
// weighs 0, Draw falls back to a uniform pick across the pool. rewards is
// read only; it is never sorted, reordered, or mutated, since callers may
// pass the recipe cache's own backing slice.
func Draw(rewards []recipe.Reward) (recipe.Reward, error) {
	if len(rewards) == 0 {
		return recipe.Reward{}, ErrEmptyRewardPool
	}

	if total := totalWeight(rewards); total > 0 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
		if err != nil {
			return recipe.Reward{}, err
		}
		return rewards[selectWeightedIndex(rewards, uint32(n.Int64()))], nil
	}

	// No reward in the pool declares a weight: fall back to a uniform pick.
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(rewards))))
	if err != nil {
		return recipe.Reward{}, err
	}
	return rewards[n.Int64()], nil
}
