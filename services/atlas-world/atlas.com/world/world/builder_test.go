package world_test

import (
	"atlas-world/channel"
	"atlas-world/world"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := world.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	channels := createTestChannels(t, 2)

	model, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		SetState(world.StateEvent).
		SetMessage("Welcome to Scania").
		SetEventMessage("Double EXP Event!").
		SetRecommendedMessage("Great for new players").
		SetCapacityStatus(world.StatusNormal).
		SetChannels(channels).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.Name() != "Scania" {
		t.Errorf("model.Name() = %s, want Scania", model.Name())
	}
	if model.State() != world.StateEvent {
		t.Errorf("model.State() = %d, want StateEvent", model.State())
	}
	if model.Message() != "Welcome to Scania" {
		t.Errorf("model.Message() = %s, want 'Welcome to Scania'", model.Message())
	}
	if model.EventMessage() != "Double EXP Event!" {
		t.Errorf("model.EventMessage() = %s, want 'Double EXP Event!'", model.EventMessage())
	}
	if model.RecommendedMessage() != "Great for new players" {
		t.Errorf("model.RecommendedMessage() = %s, want 'Great for new players'", model.RecommendedMessage())
	}
	if model.CapacityStatus() != world.StatusNormal {
		t.Errorf("model.CapacityStatus() = %d, want StatusNormal", model.CapacityStatus())
	}
	if len(model.Channels()) != 2 {
		t.Errorf("len(model.Channels()) = %d, want 2", len(model.Channels()))
	}
}

func TestBuild_MissingName(t *testing.T) {
	_, err := world.NewModelBuilder().
		SetId(1).
		Build()

	if err != world.ErrMissingName {
		t.Errorf("Build() error = %v, want ErrMissingName", err)
	}
}

func TestBuild_EmptyName(t *testing.T) {
	_, err := world.NewModelBuilder().
		SetId(1).
		SetName("").
		Build()

	if err != world.ErrMissingName {
		t.Errorf("Build() error = %v, want ErrMissingName", err)
	}
}

func TestBuild_Success(t *testing.T) {
	model, err := world.NewModelBuilder().
		SetId(0).
		SetName("Bera").
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 0 {
		t.Errorf("model.Id() = %d, want 0", model.Id())
	}
	if model.Name() != "Bera" {
		t.Errorf("model.Name() = %s, want Bera", model.Name())
	}
}

func TestBuild_IdZero(t *testing.T) {
	// Id of 0 should be valid (first world)
	model, err := world.NewModelBuilder().
		SetId(0).
		SetName("Scania").
		Build()

	if err != nil {
		t.Fatalf("Build() with id=0 should succeed, got error: %v", err)
	}
	if model.Id() != 0 {
		t.Errorf("model.Id() = %d, want 0", model.Id())
	}
}

func TestCloneModel(t *testing.T) {
	channels := createTestChannels(t, 2)

	original, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		SetState(world.StateNormal).
		SetMessage("Welcome").
		SetEventMessage("Event!").
		SetRecommendedMessage("Recommended").
		SetCapacityStatus(world.StatusNormal).
		SetChannels(channels).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := world.CloneModel(original).
		SetState(world.StateHot).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.State() != world.StateNormal {
		t.Errorf("original.State() = %d, want StateNormal", original.State())
	}

	// Cloned should have new state but same other values
	if cloned.Id() != 1 {
		t.Errorf("cloned.Id() = %d, want 1", cloned.Id())
	}
	if cloned.Name() != "Scania" {
		t.Errorf("cloned.Name() = %s, want Scania", cloned.Name())
	}
	if cloned.State() != world.StateHot {
		t.Errorf("cloned.State() = %d, want StateHot", cloned.State())
	}
	if cloned.Message() != "Welcome" {
		t.Errorf("cloned.Message() = %s, want 'Welcome'", cloned.Message())
	}
	if len(cloned.Channels()) != 2 {
		t.Errorf("len(cloned.Channels()) = %d, want 2", len(cloned.Channels()))
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		MustBuild()

	if model.Name() != "Scania" {
		t.Errorf("model.Name() = %s, want Scania", model.Name())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	world.NewModelBuilder().
		SetId(1).
		MustBuild() // Missing name, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	model, err := world.NewModelBuilder().
		SetId(2).
		SetName("Bera").
		SetState(world.StateNew).
		SetMessage("Hello World").
		SetEventMessage("Special Event").
		SetRecommendedMessage("Join us!").
		SetCapacityStatus(world.StatusHighlyPopulated).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 2 {
		t.Errorf("model.Id() = %d, want 2", model.Id())
	}
	if model.Name() != "Bera" {
		t.Errorf("model.Name() = %s, want Bera", model.Name())
	}
	if model.State() != world.StateNew {
		t.Errorf("model.State() = %d, want StateNew", model.State())
	}
	if model.CapacityStatus() != world.StatusHighlyPopulated {
		t.Errorf("model.CapacityStatus() = %d, want StatusHighlyPopulated", model.CapacityStatus())
	}
}

func TestAllStateValues(t *testing.T) {
	states := []world.State{
		world.StateNormal,
		world.StateEvent,
		world.StateNew,
		world.StateHot,
	}

	for _, state := range states {
		model, err := world.NewModelBuilder().
			SetId(1).
			SetName("TestWorld").
			SetState(state).
			Build()

		if err != nil {
			t.Errorf("Build() with state=%d unexpected error: %v", state, err)
		}
		if model.State() != state {
			t.Errorf("model.State() = %d, want %d", model.State(), state)
		}
	}
}

func TestAllStatusValues(t *testing.T) {
	statuses := []world.Status{
		world.StatusNormal,
		world.StatusHighlyPopulated,
		world.StatusFull,
	}

	for _, status := range statuses {
		model, err := world.NewModelBuilder().
			SetId(1).
			SetName("TestWorld").
			SetCapacityStatus(status).
			Build()

		if err != nil {
			t.Errorf("Build() with status=%d unexpected error: %v", status, err)
		}
		if model.CapacityStatus() != status {
			t.Errorf("model.CapacityStatus() = %d, want %d", model.CapacityStatus(), status)
		}
	}
}

func TestBuild_EmptyChannels(t *testing.T) {
	model, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		SetChannels([]channel.Model{}).
		Build()

	if err != nil {
		t.Fatalf("Build() with empty channels should succeed, got error: %v", err)
	}
	if len(model.Channels()) != 0 {
		t.Errorf("len(model.Channels()) = %d, want 0", len(model.Channels()))
	}
}

func TestBuild_NilChannels(t *testing.T) {
	model, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		Build()

	if err != nil {
		t.Fatalf("Build() without setting channels should succeed, got error: %v", err)
	}
	if model.Channels() != nil && len(model.Channels()) != 0 {
		t.Errorf("model.Channels() should be nil or empty, got length %d", len(model.Channels()))
	}
}

func TestRecommended(t *testing.T) {
	// With recommended message
	model1, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		SetRecommendedMessage("Join this world!").
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if !model1.Recommended() {
		t.Error("model.Recommended() = false, want true when RecommendedMessage is set")
	}

	// Without recommended message
	model2, err := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model2.Recommended() {
		t.Error("model.Recommended() = true, want false when RecommendedMessage is empty")
	}
}

// Helper function to create test channels
func createTestChannels(t *testing.T, count int) []channel.Model {
	t.Helper()
	channels := make([]channel.Model, count)
	for i := 0; i < count; i++ {
		ch, err := channel.NewModelBuilder().
			SetId(uuid.New()).
			SetWorldId(1).
			SetChannelId(byte(i)).
			SetIpAddress("192.168.1.1").
			SetPort(8080 + i).
			SetMaxCapacity(100).
			SetCreatedAt(time.Now()).
			Build()
		if err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}
		channels[i] = ch
	}
	return channels
}
