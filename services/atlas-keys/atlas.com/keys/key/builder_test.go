package key

import (
	"testing"
)

func TestModelBuilder_Build_Success(t *testing.T) {
	m, err := NewModelBuilder().
		SetCharacterId(12345).
		SetKey(18).
		SetType(4).
		SetAction(0).
		Build()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m.CharacterId() != 12345 {
		t.Errorf("expected characterId 12345, got %d", m.CharacterId())
	}
	if m.Key() != 18 {
		t.Errorf("expected key 18, got %d", m.Key())
	}
	if m.Type() != 4 {
		t.Errorf("expected type 4, got %d", m.Type())
	}
	if m.Action() != 0 {
		t.Errorf("expected action 0, got %d", m.Action())
	}
}

func TestModelBuilder_Build_FailsWithZeroCharacterId(t *testing.T) {
	_, err := NewModelBuilder().
		SetKey(18).
		SetType(4).
		SetAction(0).
		Build()

	if err == nil {
		t.Fatal("expected error for zero characterId, got nil")
	}
	if err.Error() != "characterId is required" {
		t.Errorf("expected 'characterId is required', got %q", err.Error())
	}
}

func TestCloneModelBuilder_CopiesAllFields(t *testing.T) {
	original, _ := NewModelBuilder().
		SetCharacterId(99999).
		SetKey(65).
		SetType(6).
		SetAction(106).
		Build()

	cloned := CloneModelBuilder(original).MustBuild()

	if cloned.CharacterId() != original.CharacterId() {
		t.Errorf("characterId mismatch: got %d, want %d", cloned.CharacterId(), original.CharacterId())
	}
	if cloned.Key() != original.Key() {
		t.Errorf("key mismatch: got %d, want %d", cloned.Key(), original.Key())
	}
	if cloned.Type() != original.Type() {
		t.Errorf("type mismatch: got %d, want %d", cloned.Type(), original.Type())
	}
	if cloned.Action() != original.Action() {
		t.Errorf("action mismatch: got %d, want %d", cloned.Action(), original.Action())
	}
}

func TestModelBuilder_MustBuild_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid input, got none")
		}
	}()

	NewModelBuilder().MustBuild()
}

func TestModelBuilder_FluentSetters_ReturnBuilder(t *testing.T) {
	b := NewModelBuilder()

	if b.SetCharacterId(1) != b {
		t.Error("SetCharacterId should return the same builder")
	}
	if b.SetKey(1) != b {
		t.Error("SetKey should return the same builder")
	}
	if b.SetType(1) != b {
		t.Error("SetType should return the same builder")
	}
	if b.SetAction(1) != b {
		t.Error("SetAction should return the same builder")
	}
}

func TestModelBuilder_Accessors(t *testing.T) {
	b := NewModelBuilder().
		SetCharacterId(111).
		SetKey(222).
		SetType(5).
		SetAction(333)

	if b.CharacterId() != 111 {
		t.Errorf("expected characterId 111, got %d", b.CharacterId())
	}
	if b.Key() != 222 {
		t.Errorf("expected key 222, got %d", b.Key())
	}
	if b.Type() != 5 {
		t.Errorf("expected type 5, got %d", b.Type())
	}
	if b.Action() != 333 {
		t.Errorf("expected action 333, got %d", b.Action())
	}
}
