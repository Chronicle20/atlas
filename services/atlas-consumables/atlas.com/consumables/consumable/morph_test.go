package consumable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T2: exhaustive weighting — enumerate every roll in [0, total) and assert the
// per-morph selection count equals its weight exactly. Stronger than any
// seeded statistical test (design §3.2). Table is shaped like the real
// 2211000/2212000 data (entries summing to 100); values are synthetic fixtures.
func TestSelectMorph_ExhaustiveWeighting(t *testing.T) {
	morphs := map[uint32]uint32{10: 20, 11: 30, 12: 50}
	counts := make(map[uint32]int)
	for roll := uint32(0); roll < 100; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		counts[id]++
	}
	assert.Equal(t, map[uint32]int{10: 20, 11: 30, 12: 50}, counts)
}

// T2b: FR-5 — no assumption that weights sum to 100.
func TestSelectMorph_WeightsNotSummingTo100(t *testing.T) {
	morphs := map[uint32]uint32{1: 3, 2: 1}
	counts := make(map[uint32]int)
	for roll := uint32(0); roll < 4; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		counts[id]++
	}
	assert.Equal(t, map[uint32]int{1: 3, 2: 1}, counts)
}

// T3: degenerate tables.
func TestSelectMorph_EmptyTable(t *testing.T) {
	_, ok := selectMorph(map[uint32]uint32{}, 0)
	assert.False(t, ok)
}

func TestSelectMorph_AllZeroWeights(t *testing.T) {
	_, ok := selectMorph(map[uint32]uint32{5: 0, 6: 0}, 0)
	assert.False(t, ok)
}

func TestSelectMorph_ZeroWeightEntrySkipped(t *testing.T) {
	morphs := map[uint32]uint32{1: 0, 2: 5}
	for roll := uint32(0); roll < 5; roll++ {
		id, ok := selectMorph(morphs, roll)
		assert.True(t, ok, "roll %d", roll)
		assert.Equal(t, uint32(2), id, "roll %d", roll)
	}
}

// T4: roll seam — zero-total errors; valid results are always table keys.
// No distribution assertion here; TestSelectMorph_ExhaustiveWeighting owns weighting.
func TestRollMorph_ZeroTotalErrors(t *testing.T) {
	_, err := rollMorph(map[uint32]uint32{7: 0})
	assert.Error(t, err)
	_, err = rollMorph(map[uint32]uint32{})
	assert.Error(t, err)
}

func TestRollMorph_ResultAlwaysTableKey(t *testing.T) {
	morphs := map[uint32]uint32{10: 20, 11: 30, 12: 50}
	for i := 0; i < 200; i++ {
		id, err := rollMorph(morphs)
		assert.NoError(t, err)
		_, present := morphs[id]
		assert.True(t, present, "rolled id %d not in table", id)
	}
}
