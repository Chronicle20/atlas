package character_test

import (
	"atlas-channel/character"
	"errors"
	"testing"
)

func TestNewModelBuilder(t *testing.T) {
	builder := character.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	model, err := character.NewModelBuilder().
		SetId(1).
		SetAccountId(100).
		SetName("TestCharacter").
		SetLevel(10).
		SetJobId(100).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetHp(100).
		SetMaxHp(100).
		SetMp(50).
		SetMaxMp(50).
		SetMeso(1000).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.AccountId() != 100 {
		t.Errorf("model.AccountId() = %d, want 100", model.AccountId())
	}
	if model.Name() != "TestCharacter" {
		t.Errorf("model.Name() = %s, want TestCharacter", model.Name())
	}
	if model.Level() != 10 {
		t.Errorf("model.Level() = %d, want 10", model.Level())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := character.NewModelBuilder().
		SetAccountId(100).
		SetName("TestCharacter").
		Build()

	if !errors.Is(err, character.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestBuild_ZeroId(t *testing.T) {
	_, err := character.NewModelBuilder().
		SetId(0).
		SetAccountId(100).
		Build()

	if !errors.Is(err, character.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestCloneModel(t *testing.T) {
	original, err := character.NewModelBuilder().
		SetId(1).
		SetAccountId(100).
		SetName("Original").
		SetLevel(50).
		SetMeso(5000).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := character.CloneModel(original).
		SetName("Cloned").
		SetLevel(60).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Name() != "Original" {
		t.Errorf("original.Name() = %s, want Original", original.Name())
	}
	if original.Level() != 50 {
		t.Errorf("original.Level() = %d, want 50", original.Level())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.Id() != 1 {
		t.Errorf("cloned.Id() = %d, want 1", cloned.Id())
	}
	if cloned.AccountId() != 100 {
		t.Errorf("cloned.AccountId() = %d, want 100", cloned.AccountId())
	}
	if cloned.Name() != "Cloned" {
		t.Errorf("cloned.Name() = %s, want Cloned", cloned.Name())
	}
	if cloned.Level() != 60 {
		t.Errorf("cloned.Level() = %d, want 60", cloned.Level())
	}
	if cloned.Meso() != 5000 {
		t.Errorf("cloned.Meso() = %d, want 5000", cloned.Meso())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := character.NewModelBuilder().
		SetId(1).
		SetAccountId(100).
		MustBuild()

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

	character.NewModelBuilder().
		SetAccountId(100).
		MustBuild() // Missing ID, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	model, err := character.NewModelBuilder().
		SetId(1).
		SetAccountId(100).
		SetWorldId(0).
		SetName("FluentTest").
		SetGender(0).
		SetSkinColor(0).
		SetFace(20000).
		SetHair(30000).
		SetLevel(1).
		SetJobId(0).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetHp(50).
		SetMaxHp(50).
		SetMp(5).
		SetMaxMp(5).
		SetAp(0).
		SetSp("0").
		SetExperience(0).
		SetFame(0).
		SetMeso(0).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Face() != 20000 {
		t.Errorf("model.Face() = %d, want 20000", model.Face())
	}
	if model.Hair() != 30000 {
		t.Errorf("model.Hair() = %d, want 30000", model.Hair())
	}
}
