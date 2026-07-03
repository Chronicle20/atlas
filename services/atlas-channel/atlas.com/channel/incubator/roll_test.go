package incubator

import "testing"

func rewards() []Reward {
	return []Reward{
		{itemId: 1, quantity: 1, weight: 40},
		{itemId: 2, quantity: 1, weight: 10},
		{itemId: 3, quantity: 1, weight: 50},
	}
}

func TestPickWeightedBoundaries(t *testing.T) {
	cases := []struct {
		roll uint32
		want uint32
	}{
		{0, 1}, {39, 1}, // first bucket [0,40)
		{40, 2}, {49, 2}, // second bucket [40,50)
		{50, 3}, {99, 3}, // third bucket [50,100)
	}
	for _, c := range cases {
		r, ok := PickWeighted(rewards(), func(total uint32) uint32 {
			if total != 100 {
				t.Fatalf("total = %d, want 100", total)
			}
			return c.roll
		})
		if !ok || r.ItemId() != c.want {
			t.Errorf("roll %d -> item %d, want %d", c.roll, r.ItemId(), c.want)
		}
	}
}

func TestPickWeightedEmptyAndZeroWeight(t *testing.T) {
	if _, ok := PickWeighted(nil, func(uint32) uint32 { return 0 }); ok {
		t.Error("empty pool must not pick")
	}
	if _, ok := PickWeighted([]Reward{{itemId: 1, weight: 0}}, func(uint32) uint32 { return 0 }); ok {
		t.Error("zero-weight pool must not pick")
	}
}
