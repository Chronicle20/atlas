package consumable

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
)

// selectMorph is the pure selection function: given the weighted morph table
// and a roll in [0, sum(weights)), return the selected morph id. Morph ids are
// walked in ascending order — Go map iteration order is randomized, so sorting
// is what makes selection a deterministic function of the roll. Returns false
// when the table is empty or all weights are zero.
func selectMorph(morphs map[uint32]uint32, roll uint32) (uint32, bool) {
	ids := make([]uint32, 0, len(morphs))
	for id := range morphs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var cumulative uint32
	for _, id := range ids {
		cumulative += morphs[id]
		if roll < cumulative {
			return id, true
		}
	}
	return 0, false
}

// rollMorph performs one clean weight-based pick over the morph table using a
// CSPRNG (the task-131 reward.go seam). It sums the weights, errors on a zero
// total (defense in depth; the caller treats this as "skip the morph statup"),
// draws one integer in [0, total), and delegates to selectMorph.
func rollMorph(morphs map[uint32]uint32) (uint32, error) {
	var total uint32
	for _, w := range morphs {
		total += w
	}
	if total == 0 {
		return 0, errors.New("morph table has zero total weight")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return 0, err
	}

	id, ok := selectMorph(morphs, uint32(n.Int64()))
	if !ok {
		// Unreachable given total > 0, but never return a fabricated morph id.
		return 0, errors.New("morph selection failed")
	}
	return id, nil
}
