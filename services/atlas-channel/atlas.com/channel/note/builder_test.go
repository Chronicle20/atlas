package note_test

import (
	"atlas-channel/note"
	"errors"
	"testing"
	"time"
)

func TestNewModelBuilder(t *testing.T) {
	builder := note.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	timestamp := time.Now()
	model, err := note.NewModelBuilder().
		SetId(1).
		SetCharacterId(100).
		SetSenderId(200).
		SetMessage("Hello World").
		SetTimestamp(timestamp).
		SetFlag(1).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.CharacterId() != 100 {
		t.Errorf("model.CharacterId() = %d, want 100", model.CharacterId())
	}
	if model.Message() != "Hello World" {
		t.Errorf("model.Message() = %s, want Hello World", model.Message())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := note.NewModelBuilder().
		SetCharacterId(100).
		Build()

	if !errors.Is(err, note.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := note.NewModelBuilder().SetId(1).MustBuild()

	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	note.NewModelBuilder().MustBuild()
}

func TestCloneModel(t *testing.T) {
	original, _ := note.NewModelBuilder().
		SetId(1).
		SetCharacterId(100).
		SetMessage("Original").
		Build()

	cloned, err := note.CloneModel(original).
		SetMessage("Cloned").
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Message() != "Original" {
		t.Errorf("original.Message() = %s, want Original", original.Message())
	}

	// Cloned should have new message
	if cloned.Message() != "Cloned" {
		t.Errorf("cloned.Message() = %s, want Cloned", cloned.Message())
	}
	// But preserve other fields
	if cloned.CharacterId() != 100 {
		t.Errorf("cloned.CharacterId() = %d, want 100", cloned.CharacterId())
	}
}
