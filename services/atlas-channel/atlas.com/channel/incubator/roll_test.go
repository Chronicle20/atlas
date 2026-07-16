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

func TestExtract_CarriesEggId(t *testing.T) {
	r, err := Extract(RewardRestModel{Id: "x", ItemId: 2000000, Quantity: 50, Weight: 40, EggId: 4170005})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if r.EggId() != 4170005 {
		t.Fatalf("EggId() = %d, want 4170005", r.EggId())
	}
}

func TestFilterByEgg(t *testing.T) {
	all := []Reward{
		mustExtract(t, 2000000, 4170000), mustExtract(t, 2000001, 4170005), mustExtract(t, 1302000, 4170005),
	}
	got := FilterByEgg(all, 4170005)
	if len(got) != 2 {
		t.Fatalf("FilterByEgg(4170005) len = %d, want 2", len(got))
	}
}

func mustExtract(t *testing.T, itemId, eggId uint32) Reward {
	t.Helper()
	r, err := Extract(RewardRestModel{Id: "r", ItemId: itemId, Quantity: 1, Weight: 1, EggId: eggId})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return r
}
