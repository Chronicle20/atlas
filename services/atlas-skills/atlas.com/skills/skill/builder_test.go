package skill_test

import (
	"atlas-skills/skill"
	"errors"
	"testing"
	"time"
)

func TestNewModelBuilder(t *testing.T) {
	builder := skill.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)
	cooldown := time.Now().Add(30 * time.Second)

	model, err := skill.NewModelBuilder().
		SetId(1001001).
		SetLevel(10).
		SetMasterLevel(20).
		SetExpiration(expiration).
		SetCooldownExpiresAt(cooldown).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1001001 {
		t.Errorf("model.Id() = %d, want 1001001", model.Id())
	}
	if model.Level() != 10 {
		t.Errorf("model.Level() = %d, want 10", model.Level())
	}
	if model.MasterLevel() != 20 {
		t.Errorf("model.MasterLevel() = %d, want 20", model.MasterLevel())
	}
	if !model.Expiration().Equal(expiration) {
		t.Errorf("model.Expiration() = %v, want %v", model.Expiration(), expiration)
	}
	if !model.CooldownExpiresAt().Equal(cooldown) {
		t.Errorf("model.CooldownExpiresAt() = %v, want %v", model.CooldownExpiresAt(), cooldown)
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := skill.NewModelBuilder().
		SetLevel(5).
		Build()

	if !errors.Is(err, skill.ErrMissingId) {
		t.Errorf("Build() error = %v, want ErrMissingId", err)
	}
}

func TestBuild_Success(t *testing.T) {
	model, err := skill.NewModelBuilder().
		SetId(1001001).
		SetLevel(5).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1001001 {
		t.Errorf("model.Id() = %d, want 1001001", model.Id())
	}
}

func TestCloneModel(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)

	original, err := skill.NewModelBuilder().
		SetId(1001001).
		SetLevel(10).
		SetMasterLevel(20).
		SetExpiration(expiration).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := skill.CloneModel(original).
		SetLevel(15).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Level() != 10 {
		t.Errorf("original.Level() = %d, want 10", original.Level())
	}

	// Cloned should have new level but same other values
	if cloned.Id() != 1001001 {
		t.Errorf("cloned.Id() = %d, want 1001001", cloned.Id())
	}
	if cloned.Level() != 15 {
		t.Errorf("cloned.Level() = %d, want 15", cloned.Level())
	}
	if cloned.MasterLevel() != 20 {
		t.Errorf("cloned.MasterLevel() = %d, want 20", cloned.MasterLevel())
	}
	if !cloned.Expiration().Equal(expiration) {
		t.Errorf("cloned.Expiration() = %v, want %v", cloned.Expiration(), expiration)
	}
}

func TestSetCooldownExpiresAt(t *testing.T) {
	cooldown := time.Now().Add(60 * time.Second)

	model, err := skill.NewModelBuilder().
		SetId(1001001).
		SetCooldownExpiresAt(cooldown).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if !model.CooldownExpiresAt().Equal(cooldown) {
		t.Errorf("model.CooldownExpiresAt() = %v, want %v", model.CooldownExpiresAt(), cooldown)
	}
}

func TestBuilderFluentChaining(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)

	model, err := skill.NewModelBuilder().
		SetId(2001001).
		SetLevel(1).
		SetMasterLevel(30).
		SetExpiration(expiration).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 2001001 {
		t.Errorf("model.Id() = %d, want 2001001", model.Id())
	}
	if model.Level() != 1 {
		t.Errorf("model.Level() = %d, want 1", model.Level())
	}
	if model.MasterLevel() != 30 {
		t.Errorf("model.MasterLevel() = %d, want 30", model.MasterLevel())
	}
}
