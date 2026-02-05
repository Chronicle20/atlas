package monster_test

import (
	"atlas-channel/monster"
	"testing"

	"github.com/Chronicle20/atlas-constants/field"
)

func TestNewModelBuilder(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000).Build()
	builder := monster.NewModelBuilder(1, f, 100100)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	f := field.NewBuilder(0, 1, 100000).Build()
	model, err := monster.NewModelBuilder(1, f, 100100).
		SetMaxHP(1000).
		SetX(100).
		SetY(200).
		SetStance(5).
		SetFH(10).
		SetTeam(0).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.UniqueId() != 1 {
		t.Errorf("model.UniqueId() = %d, want 1", model.UniqueId())
	}
	if model.MonsterId() != 100100 {
		t.Errorf("model.MonsterId() = %d, want 100100", model.MonsterId())
	}
	if model.MaxHP() != 1000 {
		t.Errorf("model.MaxHP() = %d, want 1000", model.MaxHP())
	}
}

func TestBuild_MissingUniqueId(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000).Build()
	_, err := monster.NewModelBuilder(0, f, 100100).
		Build()

	if err != monster.ErrInvalidUniqueId {
		t.Errorf("Build() error = %v, want ErrInvalidUniqueId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	f := field.NewBuilder(0, 0, 100000).Build()
	model := monster.NewModelBuilder(1, f, 100100).MustBuild()

	if model.UniqueId() != 1 {
		t.Errorf("model.UniqueId() = %d, want 1", model.UniqueId())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	f := field.NewBuilder(0, 0, 100000).Build()
	monster.NewModelBuilder(0, f, 100100).MustBuild() // Zero unique ID, should panic
}

func TestCloneModel(t *testing.T) {
	f := field.NewBuilder(0, 1, 100000).Build()
	original, _ := monster.NewModelBuilder(1, f, 100100).
		SetMaxHP(1000).
		SetX(100).
		Build()

	cloned, err := monster.CloneModel(original).
		SetX(200).
		SetControlCharacterId(50).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Cloned should have new position
	if cloned.X() != 200 {
		t.Errorf("cloned.X() = %d, want 200", cloned.X())
	}
	// Cloned should preserve other fields
	if cloned.MaxHP() != 1000 {
		t.Errorf("cloned.MaxHP() = %d, want 1000", cloned.MaxHP())
	}
}
