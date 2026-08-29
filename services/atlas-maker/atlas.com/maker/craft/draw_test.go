package craft

import (
	"atlas-maker/recipe"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func rewardsWithWeights(weights ...uint32) []recipe.Reward {
	rewards := make([]recipe.Reward, 0, len(weights))
	for i, w := range weights {
		rewards = append(rewards, recipe.Reward{
			ItemId:  item.Id(1000000 + uint32(i)),
			ItemNum: 1,
			Prob:    w,
		})
	}
	return rewards
}

func TestSelectWeightedIndexAcrossCumulativeRanges(t *testing.T) {
	pool := rewardsWithWeights(70, 25, 5)

	tests := []struct {
		name     string
		roll     uint32
		expected int
	}{
		{"first weight lower bound", 0, 0},
		{"first weight upper bound", 69, 0},
		{"second weight lower bound", 70, 1},
		{"second weight upper bound", 94, 1},
		{"third weight lower bound", 95, 2},
		{"third weight upper bound", 99, 2},
		{"roll at total falls to last", 100, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectWeightedIndex(pool, tt.roll)
			if got != tt.expected {
				t.Fatalf("selectWeightedIndex(%v, %d) = %d, want %d", pool, tt.roll, got, tt.expected)
			}
		})
	}
}

func TestTotalWeight(t *testing.T) {
	pool := rewardsWithWeights(70, 25, 5)
	if got := totalWeight(pool); got != 100 {
		t.Fatalf("totalWeight(%v) = %d, want 100", pool, got)
	}

	if got := totalWeight(nil); got != 0 {
		t.Fatalf("totalWeight(nil) = %d, want 0", got)
	}
}

func TestSelectWeightedIndexSingleEntry(t *testing.T) {
	pool := rewardsWithWeights(1)
	if got := selectWeightedIndex(pool, 0); got != 0 {
		t.Fatalf("selectWeightedIndex(%v, 0) = %d, want 0", pool, got)
	}
}

func TestSelectWeightedIndexZeroWeightEntriesAreNeverSelected(t *testing.T) {
	pool := rewardsWithWeights(50, 0, 50)
	for roll := uint32(0); roll < 100; roll++ {
		if got := selectWeightedIndex(pool, roll); got == 1 {
			t.Fatalf("selectWeightedIndex(%v, %d) = 1, expected the zero-weight entry to never be selected", pool, roll)
		}
	}
}

func TestDrawReturnsAnEntryFromThePool(t *testing.T) {
	pool := rewardsWithWeights(70, 25, 5)

	for i := 0; i < 100; i++ {
		got, err := Draw(pool)
		if err != nil {
			t.Fatalf("Draw returned unexpected error: %v", err)
		}

		found := false
		for _, r := range pool {
			if r == got {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Draw returned %v, which is not one of the pool entries %v", got, pool)
		}
	}
}

func TestDrawEmptyPoolReturnsError(t *testing.T) {
	_, err := Draw(nil)
	if err == nil {
		t.Fatal("Draw(nil) returned nil error, want a distinguishable error")
	}
}

func TestDrawAllZeroWeightsSelectsUniformly(t *testing.T) {
	pool := rewardsWithWeights(0, 0, 0)

	got, err := Draw(pool)
	if err != nil {
		t.Fatalf("Draw returned unexpected error: %v", err)
	}

	found := false
	for _, r := range pool {
		if r == got {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Draw returned %v, which is not one of the pool entries %v", got, pool)
	}
}
