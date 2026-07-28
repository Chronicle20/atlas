package reward

import "testing"

// TestSelectWeightedIndex_BoundaryHonored deterministically walks every roll
// value in [0, totalWeight) for a pool of {A: weight 1, B: weight 3} and
// asserts the cumulative-weight boundary lands where expected: roll 0 maps
// to A, rolls 1-3 map to B. This is the pure helper the weighted branch of
// selectItem delegates to, so the distribution can be proven without
// stubbing crypto/rand.
func TestSelectWeightedIndex_BoundaryHonored(t *testing.T) {
	pool := []poolItem{
		{ItemId: 1000, Quantity: 1, Weight: 1}, // A: cumulative range [0,1)
		{ItemId: 2000, Quantity: 1, Weight: 3}, // B: cumulative range [1,4)
	}

	expected := map[uint32]int{
		0: 0, // roll 0 -> A
		1: 1, // roll 1 -> B
		2: 1, // roll 2 -> B
		3: 1, // roll 3 -> B
	}

	for roll, wantIdx := range expected {
		gotIdx := selectWeightedIndex(pool, roll)
		if gotIdx != wantIdx {
			t.Errorf("selectWeightedIndex(pool, %d) = %d, want %d", roll, gotIdx, wantIdx)
		}
	}
}

// TestSelectWeightedIndex_ZeroWeightItemUnreachable proves a zero-weight
// item sitting between two positive-weight items never has a roll land on
// it: its cumulative range is zero-width, so no roll in [0, totalWeight)
// can select it.
func TestSelectWeightedIndex_ZeroWeightItemUnreachable(t *testing.T) {
	pool := []poolItem{
		{ItemId: 1000, Quantity: 1, Weight: 1}, // A: cumulative range [0,1)
		{ItemId: 2000, Quantity: 1, Weight: 0}, // C: zero-width range, unreachable
		{ItemId: 3000, Quantity: 1, Weight: 3}, // B: cumulative range [1,4)
	}

	total := totalWeight(pool)
	if total != 4 {
		t.Fatalf("totalWeight(pool) = %d, want 4", total)
	}

	for roll := uint32(0); roll < total; roll++ {
		idx := selectWeightedIndex(pool, roll)
		if idx == 1 {
			t.Errorf("selectWeightedIndex(pool, %d) selected the zero-weight item at index 1", roll)
		}
	}
}

// TestSelectItem_WeightedSelection_OnlyReachableItems exercises selectItem
// end-to-end (real crypto/rand) over a weighted pool and asserts every
// result is one of the two positive-weight items, and both appear across
// enough iterations that a missing one indicates a real bug rather than
// chance (P(missing the 1/4-weighted item over 300 draws) is astronomically
// small).
func TestSelectItem_WeightedSelection_OnlyReachableItems(t *testing.T) {
	pool := []poolItem{
		{ItemId: 1000, Quantity: 1, Weight: 1},
		{ItemId: 2000, Quantity: 1, Weight: 3},
	}

	seen := map[uint32]bool{}
	for i := 0; i < 300; i++ {
		selected, err := selectItem(pool)
		if err != nil {
			t.Fatalf("selectItem() returned error: %v", err)
		}
		if selected.ItemId != 1000 && selected.ItemId != 2000 {
			t.Fatalf("selectItem() returned unreachable item id %d", selected.ItemId)
		}
		seen[selected.ItemId] = true
	}

	if !seen[1000] {
		t.Error("expected weight-1 item 1000 to appear across 300 draws, never did")
	}
	if !seen[2000] {
		t.Error("expected weight-3 item 2000 to appear across 300 draws, never did")
	}
}

// TestSelectItem_AllWeightsZero_UniformFallback proves the existing
// behavior is preserved: when every item in the pool has weight 0,
// selectItem falls back to the pre-existing uniform pick over the whole
// pool (rand.Int(len(pool))) rather than the weighted branch.
func TestSelectItem_AllWeightsZero_UniformFallback(t *testing.T) {
	pool := []poolItem{
		{ItemId: 1000, Quantity: 1, Weight: 0},
		{ItemId: 2000, Quantity: 1, Weight: 0},
		{ItemId: 3000, Quantity: 1, Weight: 0},
	}

	seen := map[uint32]bool{}
	for i := 0; i < 300; i++ {
		selected, err := selectItem(pool)
		if err != nil {
			t.Fatalf("selectItem() returned error: %v", err)
		}
		if selected.ItemId != 1000 && selected.ItemId != 2000 && selected.ItemId != 3000 {
			t.Fatalf("selectItem() returned unexpected item id %d", selected.ItemId)
		}
		seen[selected.ItemId] = true
	}

	for _, id := range []uint32{1000, 2000, 3000} {
		if !seen[id] {
			t.Errorf("expected item %d to be reachable under uniform fallback across 300 draws, never appeared", id)
		}
	}
}

// TestSelectItem_EmptyPool_Errors preserves the existing empty-pool error
// path for both the weighted and uniform branches.
func TestSelectItem_EmptyPool_Errors(t *testing.T) {
	_, err := selectItem(nil)
	if err == nil {
		t.Error("expected error selecting from an empty pool, got nil")
	}
}
