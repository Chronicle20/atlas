package pet

import (
	"testing"
)

func TestWeightedRoll(t *testing.T) {
	t.Run("total==0 guard returns 0 and never panics", func(t *testing.T) {
		got := weightedRoll([]uint32{0, 0, 0})
		if got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("single candidate always returns 0", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			got := weightedRoll([]uint32{100})
			if got != 0 {
				t.Fatalf("expected 0, got %d (iteration %d)", got, i)
			}
		}
	})

	t.Run("zero-weight entries are never selected", func(t *testing.T) {
		// weights {0, 5, 0}: only index 1 has weight, so every draw must return 1.
		for i := 0; i < 1000; i++ {
			got := weightedRoll([]uint32{0, 5, 0})
			if got != 1 {
				t.Fatalf("expected 1, got %d (iteration %d)", got, i)
			}
		}
	})

	t.Run("all indices in range and only positive-weight indices selected", func(t *testing.T) {
		// Weights that do NOT sum to 100 — verifies "relative not percentage" semantics.
		weights := []uint32{330, 330, 330, 9, 1}
		seen := make(map[int]bool)
		for i := 0; i < 5000; i++ {
			got := weightedRoll(weights)
			if got < 0 || got >= len(weights) {
				t.Fatalf("index %d out of range [0, %d) at iteration %d", got, len(weights), i)
			}
			if weights[got] == 0 {
				t.Fatalf("index %d has zero weight but was selected at iteration %d", got, i)
			}
			seen[got] = true
		}
		// Every positive-weight index should appear at least once over 5000 draws.
		for i, w := range weights {
			if w > 0 && !seen[i] {
				t.Fatalf("index %d (weight %d) was never selected over 5000 draws", i, w)
			}
		}
	})
}
