package key

import (
	"testing"
)

func TestModel_Accessors(t *testing.T) {
	m, _ := NewModelBuilder().
		SetCharacterId(12345).
		SetKey(18).
		SetType(4).
		SetAction(100).
		Build()

	if m.CharacterId() != 12345 {
		t.Errorf("CharacterId() = %d, want 12345", m.CharacterId())
	}
	if m.Key() != 18 {
		t.Errorf("Key() = %d, want 18", m.Key())
	}
	if m.Type() != 4 {
		t.Errorf("Type() = %d, want 4", m.Type())
	}
	if m.Action() != 100 {
		t.Errorf("Action() = %d, want 100", m.Action())
	}
}

func TestModel_ZeroValues(t *testing.T) {
	// Model with zero values (except characterId which is required)
	m, _ := NewModelBuilder().
		SetCharacterId(1).
		Build()

	if m.Key() != 0 {
		t.Errorf("Key() = %d, want 0", m.Key())
	}
	if m.Type() != 0 {
		t.Errorf("Type() = %d, want 0", m.Type())
	}
	if m.Action() != 0 {
		t.Errorf("Action() = %d, want 0", m.Action())
	}
}
