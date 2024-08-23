package slottable_test

import (
	"atlas-character/asset"
	"atlas-character/slottable"
	"testing"
)

type TestModel struct {
	slot int16
}

func (m TestModel) Slot() int16 {
	return m.slot
}

func HelperFunc() {

}

// TestMinFreeSlot1 tests minFreeSlot with existing slots 0, 1, 4, 7, 8.
func TestMinFreeSlot1(t *testing.T) {
	items := []asset.Slottable{
		TestModel{slot: 0},
		TestModel{slot: 1},
		TestModel{slot: 4},
		TestModel{slot: 7},
		TestModel{slot: 8},
	}
	result := slottable.MinFreeSlot(items)
	if result != 2 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 2, result)
	}
}

// TestMinFreeSlot2 tests minFreeSlot with existing slots 1, 2, 4, 7, 8.
func TestMinFreeSlot2(t *testing.T) {
	items := []asset.Slottable{
		TestModel{slot: 1},
		TestModel{slot: 2},
		TestModel{slot: 4},
		TestModel{slot: 7},
		TestModel{slot: 8},
	}
	result := slottable.MinFreeSlot(items)
	if result != 3 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 3, result)
	}
}

// TestMinFreeSlot3 tests minFreeSlot with existing slots 0, 1, 2, 3, 4.
func TestMinFreeSlot3(t *testing.T) {
	items := []asset.Slottable{
		TestModel{slot: 0},
		TestModel{slot: 1},
		TestModel{slot: 2},
		TestModel{slot: 3},
		TestModel{slot: 4},
	}
	result := slottable.MinFreeSlot(items)
	if result != 5 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 5, result)
	}
}

// TestMinFreeSlot5 tests minFreeSlot with existing slots -7, 1, 2, 3
func TestMinFreeSlot5(t *testing.T) {
	items := []asset.Slottable{
		TestModel{slot: -7},
		TestModel{slot: 1},
		TestModel{slot: 2},
		TestModel{slot: 3},
	}
	result := slottable.MinFreeSlot(items)
	if result != 4 {
		t.Fatalf("MinFreeSlot expected=%d, got=%d", 4, result)
	}
}
