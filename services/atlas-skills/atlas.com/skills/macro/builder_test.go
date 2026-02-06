package macro_test

import (
	"atlas-skills/macro"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/skill"
)

func TestNewModelBuilder(t *testing.T) {
	builder := macro.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	model, err := macro.NewModelBuilder().
		SetId(1).
		SetName("Attack Combo").
		SetShout(true).
		SetSkillId1(skill.Id(1001001)).
		SetSkillId2(skill.Id(1001002)).
		SetSkillId3(skill.Id(1001003)).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.Name() != "Attack Combo" {
		t.Errorf("model.Name() = %s, want \"Attack Combo\"", model.Name())
	}
	if model.Shout() != true {
		t.Errorf("model.Shout() = %v, want true", model.Shout())
	}
	if model.SkillId1() != skill.Id(1001001) {
		t.Errorf("model.SkillId1() = %d, want 1001001", model.SkillId1())
	}
	if model.SkillId2() != skill.Id(1001002) {
		t.Errorf("model.SkillId2() = %d, want 1001002", model.SkillId2())
	}
	if model.SkillId3() != skill.Id(1001003) {
		t.Errorf("model.SkillId3() = %d, want 1001003", model.SkillId3())
	}
}

func TestBuild_MissingName(t *testing.T) {
	_, err := macro.NewModelBuilder().
		SetId(1).
		Build()

	if !errors.Is(err, macro.ErrMissingName) {
		t.Errorf("Build() error = %v, want ErrMissingName", err)
	}
}

func TestBuild_Success(t *testing.T) {
	model, err := macro.NewModelBuilder().
		SetId(0).
		SetName("Test Macro").
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Name() != "Test Macro" {
		t.Errorf("model.Name() = %s, want \"Test Macro\"", model.Name())
	}
}

func TestCloneModel(t *testing.T) {
	original, err := macro.NewModelBuilder().
		SetId(1).
		SetName("Original").
		SetShout(true).
		SetSkillId1(skill.Id(1001001)).
		SetSkillId2(skill.Id(1001002)).
		SetSkillId3(skill.Id(1001003)).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := macro.CloneModel(original).
		SetName("Cloned").
		SetShout(false).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Name() != "Original" {
		t.Errorf("original.Name() = %s, want \"Original\"", original.Name())
	}
	if original.Shout() != true {
		t.Errorf("original.Shout() = %v, want true", original.Shout())
	}

	// Cloned should have new values but same skill IDs
	if cloned.Id() != 1 {
		t.Errorf("cloned.Id() = %d, want 1", cloned.Id())
	}
	if cloned.Name() != "Cloned" {
		t.Errorf("cloned.Name() = %s, want \"Cloned\"", cloned.Name())
	}
	if cloned.Shout() != false {
		t.Errorf("cloned.Shout() = %v, want false", cloned.Shout())
	}
	if cloned.SkillId1() != skill.Id(1001001) {
		t.Errorf("cloned.SkillId1() = %d, want 1001001", cloned.SkillId1())
	}
}

func TestBuilderFluentChaining(t *testing.T) {
	model, err := macro.NewModelBuilder().
		SetId(2).
		SetName("Buff Combo").
		SetShout(false).
		SetSkillId1(skill.Id(2001001)).
		SetSkillId2(skill.Id(2001002)).
		SetSkillId3(skill.Id(2001003)).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 2 {
		t.Errorf("model.Id() = %d, want 2", model.Id())
	}
	if model.Name() != "Buff Combo" {
		t.Errorf("model.Name() = %s, want \"Buff Combo\"", model.Name())
	}
}

func TestSetSkillIds(t *testing.T) {
	// Test setting individual skill IDs
	model, err := macro.NewModelBuilder().
		SetId(3).
		SetName("Test").
		SetSkillId1(skill.Id(100)).
		SetSkillId2(skill.Id(200)).
		SetSkillId3(skill.Id(300)).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.SkillId1() != skill.Id(100) {
		t.Errorf("model.SkillId1() = %d, want 100", model.SkillId1())
	}
	if model.SkillId2() != skill.Id(200) {
		t.Errorf("model.SkillId2() = %d, want 200", model.SkillId2())
	}
	if model.SkillId3() != skill.Id(300) {
		t.Errorf("model.SkillId3() = %d, want 300", model.SkillId3())
	}
}
