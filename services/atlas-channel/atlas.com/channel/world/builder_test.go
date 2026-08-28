package world_test

import (
	"atlas-channel/world"
	"testing"

	worldId "github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestNewBuilder(t *testing.T) {
	builder := world.NewBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	model, err := world.NewBuilder().
		SetId(worldId.Id(0)).
		SetName("Scania").
		SetState(world.StateNormal).
		SetMessage("Welcome").
		SetEventMessage("Event!").
		SetRecommendedMessage("Recommended").
		SetCapacityStatus(world.StatusNormal).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != worldId.Id(0) {
		t.Errorf("model.Id() = %d, want %d", model.Id(), worldId.Id(0))
	}
	if model.Name() != "Scania" {
		t.Errorf("model.Name() = %s, want Scania", model.Name())
	}
	if model.State() != world.StateNormal {
		t.Errorf("model.State() = %d, want %d", model.State(), world.StateNormal)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := world.NewBuilder().
		SetId(worldId.Id(0)).
		SetName("Scania").
		MustBuild()

	if model.Id() != worldId.Id(0) {
		t.Errorf("model.Id() = %d, want %d", model.Id(), worldId.Id(0))
	}
}

func TestCloneModel(t *testing.T) {
	original, _ := world.NewBuilder().
		SetId(worldId.Id(0)).
		SetName("Scania").
		SetState(world.StateNormal).
		Build()

	cloned, err := world.CloneModel(original).
		SetState(world.StateEvent).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.State() != world.StateNormal {
		t.Errorf("original.State() = %d, want %d", original.State(), world.StateNormal)
	}

	// Cloned should have new state
	if cloned.State() != world.StateEvent {
		t.Errorf("cloned.State() = %d, want %d", cloned.State(), world.StateEvent)
	}
	// But preserve other fields
	if cloned.Name() != "Scania" {
		t.Errorf("cloned.Name() = %s, want Scania", cloned.Name())
	}
}

func TestRecommended(t *testing.T) {
	notRecommended := world.NewBuilder().
		SetId(worldId.Id(0)).
		MustBuild()

	if notRecommended.Recommended() {
		t.Error("Expected Recommended() to be false when recommendedMessage is empty")
	}

	recommended := world.NewBuilder().
		SetId(worldId.Id(0)).
		SetRecommendedMessage("Try this world!").
		MustBuild()

	if !recommended.Recommended() {
		t.Error("Expected Recommended() to be true when recommendedMessage is set")
	}
}
