package item

import (
	"testing"
)

// TestMinFreeSlot1 tests minFreeSlot with existing slots 0, 1, 4, 7, 8.
func TestMinFreeSlot1(t *testing.T) {
	items := []Model{
		Model{slot: 0},
		Model{slot: 1},
		Model{slot: 4},
		Model{slot: 7},
		Model{slot: 8},
	}
	result := minFreeSlot(items)
	if result != 2 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 2, result)
	}
}

// TestMinFreeSlot2 tests minFreeSlot with existing slots 1, 2, 4, 7, 8.
func TestMinFreeSlot2(t *testing.T) {
	items := []Model{
		Model{slot: 1},
		Model{slot: 2},
		Model{slot: 4},
		Model{slot: 7},
		Model{slot: 8},
	}
	result := minFreeSlot(items)
	if result != 3 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 3, result)
	}
}

// TestMinFreeSlot3 tests minFreeSlot with existing slots 0, 1, 2, 3, 4.
func TestMinFreeSlot3(t *testing.T) {
	items := []Model{
		Model{slot: 0},
		Model{slot: 1},
		Model{slot: 2},
		Model{slot: 3},
		Model{slot: 4},
	}
	result := minFreeSlot(items)
	if result != 5 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 5, result)
	}
}

// TestMinFreeSlot5 tests minFreeSlot with existing slots -7, 1, 2, 3
func TestMinFreeSlot5(t *testing.T) {
	items := []Model{
		Model{slot: -7},
		Model{slot: 1},
		Model{slot: 2},
		Model{slot: 3},
	}
	result := minFreeSlot(items)
	if result != 4 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 4, result)
	}
}
