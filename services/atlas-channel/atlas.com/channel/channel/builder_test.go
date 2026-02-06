package channel_test

import (
	"atlas-channel/channel"
	"errors"
	"testing"

	channelId "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := channel.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	model, err := channel.NewModelBuilder().
		SetId(id).
		SetWorldId(world.Id(0)).
		SetChannelId(channelId.Id(1)).
		SetIpAddress("127.0.0.1").
		SetPort(8080).
		SetCurrentCapacity(100).
		SetMaxCapacity(1000).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.WorldId() != world.Id(0) {
		t.Errorf("model.WorldId() = %d, want 0", model.WorldId())
	}
	if model.IpAddress() != "127.0.0.1" {
		t.Errorf("model.IpAddress() = %s, want 127.0.0.1", model.IpAddress())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetWorldId(world.Id(0)).
		Build()

	if !errors.Is(err, channel.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	id := uuid.New()
	model := channel.NewModelBuilder().
		SetId(id).
		MustBuild()

	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	channel.NewModelBuilder().MustBuild()
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	original, _ := channel.NewModelBuilder().
		SetId(id).
		SetWorldId(world.Id(0)).
		SetPort(8080).
		Build()

	cloned, err := channel.CloneModel(original).
		SetPort(9090).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Port() != 8080 {
		t.Errorf("original.Port() = %d, want 8080", original.Port())
	}

	// Cloned should have new port
	if cloned.Port() != 9090 {
		t.Errorf("cloned.Port() = %d, want 9090", cloned.Port())
	}
	// But preserve other fields
	if cloned.WorldId() != world.Id(0) {
		t.Errorf("cloned.WorldId() = %d, want 0", cloned.WorldId())
	}
}
