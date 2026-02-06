package messenger

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuildSuccess(t *testing.T) {
	tenantId := uuid.New()
	b := NewBuilder().
		SetTenantId(tenantId).
		SetId(1).
		AddMember(100, 0)

	m, err := b.Build()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.Id() != 1 {
		t.Fatalf("expected id=1, got %d", m.Id())
	}
	if len(m.Members()) != 1 {
		t.Fatalf("expected 1 member, got %d", len(m.Members()))
	}
	if m.Members()[0].Id() != 100 {
		t.Fatalf("expected member id=100, got %d", m.Members()[0].Id())
	}
	if m.Members()[0].Slot() != 0 {
		t.Fatalf("expected member slot=0, got %d", m.Members()[0].Slot())
	}
}

func TestBuilderBuildWithMaxMembers(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		AddMember(1, 0).
		AddMember(2, 1).
		AddMember(3, 2)

	m, err := b.Build()
	if err != nil {
		t.Fatalf("expected success with exactly MaxMembers, got %v", err)
	}
	if len(m.Members()) != MaxMembers {
		t.Fatalf("expected %d members, got %d", MaxMembers, len(m.Members()))
	}
}

func TestBuilderBuildExceedsMaxMembers(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		AddMember(1, 0).
		AddMember(2, 1).
		AddMember(3, 2).
		AddMember(4, 3) // 4 > MaxMembers=3

	_, err := b.Build()
	if !errors.Is(err, ErrAtCapacity) {
		t.Fatalf("expected ErrAtCapacity, got %v", err)
	}
}

func TestBuilderBuildWithNoMembers(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1)

	m, err := b.Build()
	if err != nil {
		t.Fatalf("expected success with no members, got %v", err)
	}
	if len(m.Members()) != 0 {
		t.Fatalf("expected 0 members, got %d", len(m.Members()))
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

	b3 := b.SetId(42)
	if b != b3 {
		t.Fatal("SetId should return same builder instance")
	}

	b4 := b.AddMember(100, 0)
	if b != b4 {
		t.Fatal("AddMember should return same builder instance")
	}
}

func TestMaxMembersConstant(t *testing.T) {
	// Verify MaxMembers constant is set correctly
	if MaxMembers != 3 {
		t.Fatalf("expected MaxMembers=3, got %d", MaxMembers)
	}
}
