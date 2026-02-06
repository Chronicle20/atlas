package character

import (
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestBuilderBuildSuccess(t *testing.T) {
	tenantId := uuid.New()
	b := NewBuilder().
		SetTenantId(tenantId).
		SetId(1).
		SetName("TestCharacter").
		SetWorldId(world.Id(0)).
		SetChannelId(channel.Id(1)).
		SetMessengerId(100).
		SetOnline(true)

	m, err := b.Build()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.Id() != 1 {
		t.Fatalf("expected id=1, got %d", m.Id())
	}
	if m.Name() != "TestCharacter" {
		t.Fatalf("expected name=TestCharacter, got %s", m.Name())
	}
	if m.WorldId() != world.Id(0) {
		t.Fatalf("expected worldId=0, got %d", m.WorldId())
	}
	if m.ChannelId() != channel.Id(1) {
		t.Fatalf("expected channelId=1, got %d", m.ChannelId())
	}
	if m.MessengerId() != 100 {
		t.Fatalf("expected messengerId=100, got %d", m.MessengerId())
	}
	if !m.Online() {
		t.Fatal("expected online=true, got false")
	}
}

func TestBuilderBuildMissingId(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetName("TestCharacter")

	_, err := b.Build()
	if !errors.Is(err, ErrMissingId) {
		t.Fatalf("expected ErrMissingId, got %v", err)
	}
}

func TestBuilderBuildMissingName(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1)

	_, err := b.Build()
	if !errors.Is(err, ErrMissingName) {
		t.Fatalf("expected ErrMissingName, got %v", err)
	}
}

func TestBuilderBuildMissingBoth(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New())

	_, err := b.Build()
	// Should fail on ID check first
	if !errors.Is(err, ErrMissingId) {
		t.Fatalf("expected ErrMissingId (checked first), got %v", err)
	}
}

func TestBuilderBuildMinimalRequired(t *testing.T) {
	// Only ID and Name are required
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		SetName("MinimalChar")

	m, err := b.Build()
	if err != nil {
		t.Fatalf("expected success with minimal required fields, got %v", err)
	}
	if m.Id() != 1 {
		t.Fatalf("expected id=1, got %d", m.Id())
	}
	if m.Name() != "MinimalChar" {
		t.Fatalf("expected name=MinimalChar, got %s", m.Name())
	}
	// Default values for optional fields
	if m.MessengerId() != 0 {
		t.Fatalf("expected default messengerId=0, got %d", m.MessengerId())
	}
	if m.Online() {
		t.Fatal("expected default online=false, got true")
	}
}

func TestBuilderChaining(t *testing.T) {
	tenantId := uuid.New()

	// Test that all builder methods return the builder for chaining
	b := NewBuilder()

	b2 := b.SetTenantId(tenantId)
	if b != b2 {
		t.Fatal("SetTenantId should return same builder instance")
	}

	b3 := b.SetId(1)
	if b != b3 {
		t.Fatal("SetId should return same builder instance")
	}

	b4 := b.SetName("Test")
	if b != b4 {
		t.Fatal("SetName should return same builder instance")
	}

	b5 := b.SetWorldId(world.Id(0))
	if b != b5 {
		t.Fatal("SetWorldId should return same builder instance")
	}

	b6 := b.SetChannelId(channel.Id(0))
	if b != b6 {
		t.Fatal("SetChannelId should return same builder instance")
	}

	b7 := b.SetMessengerId(100)
	if b != b7 {
		t.Fatal("SetMessengerId should return same builder instance")
	}

	b8 := b.SetOnline(true)
	if b != b8 {
		t.Fatal("SetOnline should return same builder instance")
	}
}

func TestBuilderEmptyName(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		SetName("")

	_, err := b.Build()
	if !errors.Is(err, ErrMissingName) {
		t.Fatalf("expected ErrMissingName for empty string, got %v", err)
	}
}
