package key

import (
	"testing"
)

func TestRestModel_GetName(t *testing.T) {
	r := RestModel{}
	if r.GetName() != "keys" {
		t.Errorf("expected name 'keys', got %q", r.GetName())
	}
}

func TestRestModel_GetID(t *testing.T) {
	r := RestModel{Key: 18}
	if r.GetID() != "18" {
		t.Errorf("expected ID '18', got %q", r.GetID())
	}
}

func TestRestModel_SetID(t *testing.T) {
	r := &RestModel{}
	err := r.SetID("42")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Key != 42 {
		t.Errorf("expected Key 42, got %d", r.Key)
	}
}

func TestRestModel_SetID_InvalidInput(t *testing.T) {
	r := &RestModel{}
	err := r.SetID("not-a-number")

	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}

func TestTransform(t *testing.T) {
	m, _ := NewModelBuilder().
		SetCharacterId(12345).
		SetKey(65).
		SetType(6).
		SetAction(106).
		Build()

	r, err := Transform(m)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Key != m.Key() {
		t.Errorf("key mismatch: got %d, want %d", r.Key, m.Key())
	}
	if r.Type != m.Type() {
		t.Errorf("type mismatch: got %d, want %d", r.Type, m.Type())
	}
	if r.Action != m.Action() {
		t.Errorf("action mismatch: got %d, want %d", r.Action, m.Action())
	}
}
