package character

import "testing"

const (
	testTopID     = 1040002 // ClassificationTop
	testBottomID  = 1060002 // ClassificationBottom
	testOverallID = 1050002 // ClassificationOverall
)

func TestApplyDefaultClothing(t *testing.T) {
	t.Run("overall-suppresses-both", func(t *testing.T) {
		eq := map[int]int{topSlot: testOverallID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != testOverallID {
			t.Errorf("overall overwritten: %d", eq[topSlot])
		}
		if _, ok := eq[bottomSlot]; ok {
			t.Errorf("overall must suppress default pants; got %d", eq[bottomSlot])
		}
	})

	t.Run("real-top-empty-bottom-injects-pants", func(t *testing.T) {
		eq := map[int]int{topSlot: testTopID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != testTopID {
			t.Errorf("real top overwritten: %d", eq[topSlot])
		}
		if eq[bottomSlot] != DefaultPantsMale {
			t.Errorf("bottom = %d; want %d", eq[bottomSlot], DefaultPantsMale)
		}
	})

	t.Run("empty-top-real-bottom-injects-coat", func(t *testing.T) {
		eq := map[int]int{bottomSlot: testBottomID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != DefaultCoatMale {
			t.Errorf("top = %d; want %d", eq[topSlot], DefaultCoatMale)
		}
		if eq[bottomSlot] != testBottomID {
			t.Errorf("real bottom overwritten: %d", eq[bottomSlot])
		}
	})

	t.Run("both-empty-injects-both-male", func(t *testing.T) {
		eq := map[int]int{}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != DefaultCoatMale || eq[bottomSlot] != DefaultPantsMale {
			t.Errorf("male both = (%d,%d); want (%d,%d)", eq[topSlot], eq[bottomSlot], DefaultCoatMale, DefaultPantsMale)
		}
	})

	t.Run("both-empty-injects-both-female", func(t *testing.T) {
		eq := map[int]int{}
		applyDefaultClothing(eq, GenderFemale)
		if eq[topSlot] != DefaultCoatFemale || eq[bottomSlot] != DefaultPantsFemale {
			t.Errorf("female both = (%d,%d); want (%d,%d)", eq[topSlot], eq[bottomSlot], DefaultCoatFemale, DefaultPantsFemale)
		}
	})
}
