package world_test

import (
	"atlas-login/world"
	"testing"
)

func TestBuilder_Build(t *testing.T) {
	m := world.NewBuilder().
		SetId(1).
		SetName("Scania").
		SetState(world.StateNormal).
		SetMessage("Welcome").
		SetEventMessage("Event!").
		SetRecommendedMessage("Try this world!").
		SetCapacityStatus(world.StatusNormal).
		Build()

	if m.Id() != 1 {
		t.Errorf("Id() = %d, want 1", m.Id())
	}
	if m.Name() != "Scania" {
		t.Errorf("Name() = %s, want 'Scania'", m.Name())
	}
	if m.State() != world.StateNormal {
		t.Errorf("State() = %d, want %d", m.State(), world.StateNormal)
	}
	if m.Message() != "Welcome" {
		t.Errorf("Message() = %s, want 'Welcome'", m.Message())
	}
	if m.EventMessage() != "Event!" {
		t.Errorf("EventMessage() = %s, want 'Event!'", m.EventMessage())
	}
	if m.RecommendedMessage() != "Try this world!" {
		t.Errorf("RecommendedMessage() = %s, want 'Try this world!'", m.RecommendedMessage())
	}
	if !m.Recommended() {
		t.Error("Recommended() = false, want true")
	}
	if m.CapacityStatus() != world.StatusNormal {
		t.Errorf("CapacityStatus() = %d, want %d", m.CapacityStatus(), world.StatusNormal)
	}
}

func TestModel_ToBuilder(t *testing.T) {
	original := world.NewBuilder().
		SetId(1).
		SetName("Scania").
		SetState(world.StateNormal).
		SetCapacityStatus(world.StatusNormal).
		Build()

	// Clone and modify capacity status
	cloned := original.ToBuilder().
		SetCapacityStatus(world.StatusHighlyPopulated).
		Build()

	// Original should be unchanged
	if original.CapacityStatus() != world.StatusNormal {
		t.Errorf("Original CapacityStatus() = %d, want %d", original.CapacityStatus(), world.StatusNormal)
	}

	// Cloned should have new status
	if cloned.CapacityStatus() != world.StatusHighlyPopulated {
		t.Errorf("Cloned CapacityStatus() = %d, want %d", cloned.CapacityStatus(), world.StatusHighlyPopulated)
	}

	// Other fields should be preserved
	if cloned.Id() != 1 {
		t.Errorf("Cloned Id() = %d, want 1", cloned.Id())
	}
	if cloned.Name() != "Scania" {
		t.Errorf("Cloned Name() = %s, want 'Scania'", cloned.Name())
	}
}

func TestNewBuilder_DefaultValues(t *testing.T) {
	m := world.NewBuilder().Build()

	if m.Id() != 0 {
		t.Errorf("Default Id() = %d, want 0", m.Id())
	}
	if m.Name() != "" {
		t.Errorf("Default Name() = %s, want ''", m.Name())
	}
	if m.State() != 0 {
		t.Errorf("Default State() = %d, want 0", m.State())
	}
	if m.CapacityStatus() != 0 {
		t.Errorf("Default CapacityStatus() = %d, want 0", m.CapacityStatus())
	}
	if m.Recommended() {
		t.Error("Default Recommended() = true, want false")
	}
}

func TestWorld_Recommended(t *testing.T) {
	// World with recommended message is recommended
	recommended := world.NewBuilder().
		SetRecommendedMessage("Come join us!").
		Build()

	if !recommended.Recommended() {
		t.Error("World with recommended message should be Recommended()")
	}

	// World without recommended message is not recommended
	notRecommended := world.NewBuilder().Build()

	if notRecommended.Recommended() {
		t.Error("World without recommended message should not be Recommended()")
	}
}
