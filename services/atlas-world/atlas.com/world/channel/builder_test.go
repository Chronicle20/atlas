package channel_test

import (
	"atlas-world/channel"
	"testing"
	"time"

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
	createdAt := time.Now()

	model, err := channel.NewModelBuilder().
		SetId(id).
		SetWorldId(1).
		SetChannelId(2).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetCurrentCapacity(50).
		SetMaxCapacity(100).
		SetCreatedAt(createdAt).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.WorldId() != 1 {
		t.Errorf("model.WorldId() = %d, want 1", model.WorldId())
	}
	if model.ChannelId() != 2 {
		t.Errorf("model.ChannelId() = %d, want 2", model.ChannelId())
	}
	if model.IpAddress() != "192.168.1.1" {
		t.Errorf("model.IpAddress() = %s, want 192.168.1.1", model.IpAddress())
	}
	if model.Port() != 8080 {
		t.Errorf("model.Port() = %d, want 8080", model.Port())
	}
	if model.CurrentCapacity() != 50 {
		t.Errorf("model.CurrentCapacity() = %d, want 50", model.CurrentCapacity())
	}
	if model.MaxCapacity() != 100 {
		t.Errorf("model.MaxCapacity() = %d, want 100", model.MaxCapacity())
	}
	if !model.CreatedAt().Equal(createdAt) {
		t.Errorf("model.CreatedAt() = %v, want %v", model.CreatedAt(), createdAt)
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrMissingId {
		t.Errorf("Build() error = %v, want ErrMissingId", err)
	}
}

func TestBuild_ZeroUUIDId(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.Nil).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrMissingId {
		t.Errorf("Build() error = %v, want ErrMissingId", err)
	}
}

func TestBuild_EmptyIpAddress(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrInvalidIpAddress {
		t.Errorf("Build() error = %v, want ErrInvalidIpAddress", err)
	}
}

func TestBuild_MissingIpAddress(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrInvalidIpAddress {
		t.Errorf("Build() error = %v, want ErrInvalidIpAddress", err)
	}
}

func TestBuild_PortZero(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(0).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrInvalidPort {
		t.Errorf("Build() error = %v, want ErrInvalidPort", err)
	}
}

func TestBuild_PortNegative(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(-1).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrInvalidPort {
		t.Errorf("Build() error = %v, want ErrInvalidPort", err)
	}
}

func TestBuild_PortTooHigh(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(65536).
		SetMaxCapacity(100).
		Build()

	if err != channel.ErrInvalidPort {
		t.Errorf("Build() error = %v, want ErrInvalidPort", err)
	}
}

func TestBuild_PortBoundaryLow(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(1).
		SetMaxCapacity(100).
		Build()

	if err != nil {
		t.Errorf("Build() with port=1 should succeed, got error: %v", err)
	}
}

func TestBuild_PortBoundaryHigh(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(65535).
		SetMaxCapacity(100).
		Build()

	if err != nil {
		t.Errorf("Build() with port=65535 should succeed, got error: %v", err)
	}
}

func TestBuild_MaxCapacityZero(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(0).
		Build()

	if err != channel.ErrInvalidCapacity {
		t.Errorf("Build() error = %v, want ErrInvalidCapacity", err)
	}
}

func TestBuild_MaxCapacityOne(t *testing.T) {
	_, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(1).
		Build()

	if err != nil {
		t.Errorf("Build() with maxCapacity=1 should succeed, got error: %v", err)
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	createdAt := time.Now()

	original, err := channel.NewModelBuilder().
		SetId(id).
		SetWorldId(1).
		SetChannelId(2).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetCurrentCapacity(50).
		SetMaxCapacity(100).
		SetCreatedAt(createdAt).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := channel.CloneModel(original).
		SetCurrentCapacity(75).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.CurrentCapacity() != 50 {
		t.Errorf("original.CurrentCapacity() = %d, want 50", original.CurrentCapacity())
	}

	// Cloned should have new capacity but same other values
	if cloned.Id() != id {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.WorldId() != 1 {
		t.Errorf("cloned.WorldId() = %d, want 1", cloned.WorldId())
	}
	if cloned.ChannelId() != 2 {
		t.Errorf("cloned.ChannelId() = %d, want 2", cloned.ChannelId())
	}
	if cloned.IpAddress() != "192.168.1.1" {
		t.Errorf("cloned.IpAddress() = %s, want 192.168.1.1", cloned.IpAddress())
	}
	if cloned.Port() != 8080 {
		t.Errorf("cloned.Port() = %d, want 8080", cloned.Port())
	}
	if cloned.CurrentCapacity() != 75 {
		t.Errorf("cloned.CurrentCapacity() = %d, want 75", cloned.CurrentCapacity())
	}
	if cloned.MaxCapacity() != 100 {
		t.Errorf("cloned.MaxCapacity() = %d, want 100", cloned.MaxCapacity())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		MustBuild()

	if model.Port() != 8080 {
		t.Errorf("model.Port() = %d, want 8080", model.Port())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	channel.NewModelBuilder().
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		MustBuild() // Missing ID, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	id := uuid.New()

	model, err := channel.NewModelBuilder().
		SetId(id).
		SetWorldId(1).
		SetChannelId(2).
		SetIpAddress("10.0.0.1").
		SetPort(9000).
		SetCurrentCapacity(25).
		SetMaxCapacity(50).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.WorldId() != 1 {
		t.Errorf("model.WorldId() = %d, want 1", model.WorldId())
	}
	if model.ChannelId() != 2 {
		t.Errorf("model.ChannelId() = %d, want 2", model.ChannelId())
	}
}

func TestBuild_DefaultCreatedAt(t *testing.T) {
	beforeBuild := time.Now()

	model, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	afterBuild := time.Now()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// CreatedAt should be between beforeBuild and afterBuild
	if model.CreatedAt().Before(beforeBuild) || model.CreatedAt().After(afterBuild) {
		t.Errorf("model.CreatedAt() = %v, expected between %v and %v", model.CreatedAt(), beforeBuild, afterBuild)
	}
}

func TestBuild_WorldIdZero(t *testing.T) {
	// WorldId of 0 should be valid
	model, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(0).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != nil {
		t.Fatalf("Build() with worldId=0 should succeed, got error: %v", err)
	}
	if model.WorldId() != 0 {
		t.Errorf("model.WorldId() = %d, want 0", model.WorldId())
	}
}

func TestBuild_ChannelIdZero(t *testing.T) {
	// ChannelId of 0 should be valid
	model, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetChannelId(0).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	if err != nil {
		t.Fatalf("Build() with channelId=0 should succeed, got error: %v", err)
	}
	if model.ChannelId() != 0 {
		t.Errorf("model.ChannelId() = %d, want 0", model.ChannelId())
	}
}

func TestBuild_CurrentCapacityZero(t *testing.T) {
	// CurrentCapacity of 0 should be valid (no users connected)
	model, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetCurrentCapacity(0).
		SetMaxCapacity(100).
		Build()

	if err != nil {
		t.Fatalf("Build() with currentCapacity=0 should succeed, got error: %v", err)
	}
	if model.CurrentCapacity() != 0 {
		t.Errorf("model.CurrentCapacity() = %d, want 0", model.CurrentCapacity())
	}
}
