package character

import "testing"

func TestCharacterSlotBounds(t *testing.T) {
	if DefaultCharacterSlotsPerWorld != int16(4) {
		t.Errorf("DefaultCharacterSlotsPerWorld = %d, want 4", DefaultCharacterSlotsPerWorld)
	}
	if MaxCharacterSlotsPerWorld != int16(12) {
		t.Errorf("MaxCharacterSlotsPerWorld = %d, want 12", MaxCharacterSlotsPerWorld)
	}
}
