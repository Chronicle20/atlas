package reactor_test

import (
	"atlas-channel/reactor"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

func TestNewModelBuilder(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	builder := reactor.NewModelBuilder(f, 100, "testReactor")
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	model, err := reactor.NewModelBuilder(f, 100, "testReactor").
		SetId(1).
		SetState(1).
		SetEventState(2).
		SetPosition(100, 200).
		SetDelay(1000).
		SetDirection(1).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.WorldId() != world.Id(0) {
		t.Errorf("model.WorldId() = %d, want 0", model.WorldId())
	}
	if model.Classification() != 100 {
		t.Errorf("model.Classification() = %d, want 100", model.Classification())
	}
}

func TestBuild_MissingId(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	_, err := reactor.NewModelBuilder(f, 100, "testReactor").
		Build()

	if !errors.Is(err, reactor.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	model := reactor.NewModelBuilder(f, 100, "testReactor").
		SetId(1).
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

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	reactor.NewModelBuilder(f, 100, "testReactor").MustBuild()
}

func TestCloneModel(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	original, _ := reactor.NewModelBuilder(f, 100, "testReactor").
		SetId(1).
		SetPosition(100, 200).
		Build()

	cloned, err := reactor.CloneModel(original).
		SetPosition(300, 400).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.X() != 100 {
		t.Errorf("original.X() = %d, want 100", original.X())
	}

	// Cloned should have new position
	if cloned.X() != 300 {
		t.Errorf("cloned.X() = %d, want 300", cloned.X())
	}
	// But preserve other fields
	if cloned.Name() != "testReactor" {
		t.Errorf("cloned.Name() = %s, want testReactor", cloned.Name())
	}
}
