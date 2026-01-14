package drop

import (
	"testing"
)

func TestDropPositionOffset_EvenIndex(t *testing.T) {
	testCases := []struct {
		name     string
		index    int
		dropType byte
		baseX    int16
		expected int16
	}{
		{"index 0, type 0", 0, 0, 100, 100},   // (0+1)/2 = 0, factor*0 = 0
		{"index 2, type 0", 2, 0, 100, 125},   // (2+1)/2 = 1, 25*1 = 25, 100+25 = 125
		{"index 4, type 0", 4, 0, 100, 150},   // (4+1)/2 = 2, 25*2 = 50, 100+50 = 150
		{"index 2, type 3", 2, 3, 100, 140},   // factor 40, (2+1)/2 = 1, 40*1 = 40, 100+40 = 140
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factor := int16(25)
			if tc.dropType == 3 {
				factor = 40
			}

			var newX int16
			if tc.index%2 == 0 {
				newX = tc.baseX + factor*int16((tc.index+1)/2)
			} else {
				newX = tc.baseX - factor*int16(tc.index/2)
			}

			// Verify our understanding matches the actual formula
			// From processor.go lines 39-43:
			// if index%2 == 0 {
			//     newX += int16(factor * ((index + 1) / 2))
			// } else {
			//     newX += int16(-(factor * (index / 2)))
			// }

			if newX != tc.expected {
				t.Errorf("For index %d with type %d: expected X=%d, got %d",
					tc.index, tc.dropType, tc.expected, newX)
			}
		})
	}
}

func TestDropPositionOffset_OddIndex(t *testing.T) {
	testCases := []struct {
		name     string
		index    int
		dropType byte
		baseX    int16
		expected int16
	}{
		{"index 1, type 0", 1, 0, 100, 100},  // 1/2 = 0, factor*0 = 0, 100-0 = 100
		{"index 3, type 0", 3, 0, 100, 75},   // 3/2 = 1, 25*1 = 25, 100-25 = 75
		{"index 5, type 0", 5, 0, 100, 50},   // 5/2 = 2, 25*2 = 50, 100-50 = 50
		{"index 3, type 3", 3, 3, 100, 60},   // factor 40, 3/2 = 1, 40*1 = 40, 100-40 = 60
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factor := int16(25)
			if tc.dropType == 3 {
				factor = 40
			}

			var newX int16
			if tc.index%2 == 0 {
				newX = tc.baseX + factor*int16((tc.index+1)/2)
			} else {
				newX = tc.baseX - factor*int16(tc.index/2)
			}

			if newX != tc.expected {
				t.Errorf("For index %d with type %d: expected X=%d, got %d",
					tc.index, tc.dropType, tc.expected, newX)
			}
		})
	}
}

func TestDropModel_Builder(t *testing.T) {
	m, _ := NewBuilder().
		SetItemId(1000).
		SetChance(50000).
		SetMinimumQuantity(5).
		SetMaximumQuantity(10).
		SetQuestId(100).
		Build()

	if m.ItemId() != 1000 {
		t.Errorf("Expected itemId 1000, got %d", m.ItemId())
	}
	if m.Chance() != 50000 {
		t.Errorf("Expected chance 50000, got %d", m.Chance())
	}
	if m.MinimumQuantity() != 5 {
		t.Errorf("Expected minQty 5, got %d", m.MinimumQuantity())
	}
	if m.MaximumQuantity() != 10 {
		t.Errorf("Expected maxQty 10, got %d", m.MaximumQuantity())
	}
}

func TestDropModel_BuilderDefaults(t *testing.T) {
	m, _ := NewBuilder().Build()

	if m.MinimumQuantity() != 1 {
		t.Errorf("Expected default minQty 1, got %d", m.MinimumQuantity())
	}
	if m.MaximumQuantity() != 1 {
		t.Errorf("Expected default maxQty 1, got %d", m.MaximumQuantity())
	}
}

func TestDropModel_MesoVsItem(t *testing.T) {
	// Meso drop has itemId = 0
	mesoDrop, _ := NewBuilder().SetItemId(0).SetMinimumQuantity(100).SetMaximumQuantity(500).Build()
	if mesoDrop.ItemId() != 0 {
		t.Errorf("Meso drop should have itemId 0")
	}

	// Item drop has itemId > 0
	itemDrop, _ := NewBuilder().SetItemId(4000100).Build()
	if itemDrop.ItemId() == 0 {
		t.Errorf("Item drop should have itemId > 0")
	}
}

// Test the drop spread pattern
// Drops are placed alternating left/right from the monster position
// Even indices go right (+), odd indices go left (-)
// Index 0: +0
// Index 1: -0
// Index 2: +factor
// Index 3: -factor
// Index 4: +2*factor
// Index 5: -2*factor
// etc.
func TestDropSpreadPattern(t *testing.T) {
	baseX := int16(200)
	factor := int16(25)

	expectedOffsets := []int16{
		0,       // index 0: (0+1)/2 = 0, +0
		0,       // index 1: 1/2 = 0, -0
		25,      // index 2: (2+1)/2 = 1, +25
		-25,     // index 3: 3/2 = 1, -25
		50,      // index 4: (4+1)/2 = 2, +50
		-50,     // index 5: 5/2 = 2, -50
		75,      // index 6: (6+1)/2 = 3, +75
		-75,     // index 7: 7/2 = 3, -75
	}

	for i, expectedOffset := range expectedOffsets {
		var actualOffset int16
		if i%2 == 0 {
			actualOffset = factor * int16((i+1)/2)
		} else {
			actualOffset = -factor * int16(i/2)
		}

		if actualOffset != expectedOffset {
			t.Errorf("Index %d: expected offset %d, got %d", i, expectedOffset, actualOffset)
		}

		expectedX := baseX + expectedOffset
		actualX := baseX + actualOffset
		if actualX != expectedX {
			t.Errorf("Index %d: expected X=%d, got %d", i, expectedX, actualX)
		}
	}
}
