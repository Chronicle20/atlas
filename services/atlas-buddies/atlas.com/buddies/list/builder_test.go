package list

import (
	"atlas-buddies/buddy"
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuild(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)

	m, err := NewBuilder(tenantId, characterId).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.tenantId != tenantId {
		t.Errorf("expected tenantId %v, got %v", tenantId, m.tenantId)
	}
	if m.characterId != characterId {
		t.Errorf("expected characterId %d, got %d", characterId, m.characterId)
	}
	if m.capacity != 20 {
		t.Errorf("expected default capacity 20, got %d", m.capacity)
	}
	if len(m.buddies) != 0 {
		t.Errorf("expected empty buddies slice, got %d buddies", len(m.buddies))
	}
}

func TestBuilderWithCustomCapacity(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	customCapacity := byte(50)

	m, err := NewBuilder(tenantId, characterId).
		SetCapacity(customCapacity).
		Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.capacity != customCapacity {
		t.Errorf("expected capacity %d, got %d", customCapacity, m.capacity)
	}
}

func TestBuilderWithId(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	id := uuid.New()

	m, err := NewBuilder(tenantId, characterId).
		SetId(id).
		Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.id != id {
		t.Errorf("expected id %v, got %v", id, m.id)
	}
}

func TestBuilderWithBuddies(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	listId := uuid.New()

	buddyModel, _ := buddy.NewBuilder(listId, 99999).
		SetCharacterName("TestBuddy").
		Build()
	buddies := []buddy.Model{buddyModel}

	m, err := NewBuilder(tenantId, characterId).
		SetBuddies(buddies).
		Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(m.buddies) != 1 {
		t.Errorf("expected 1 buddy, got %d", len(m.buddies))
	}
}

func TestBuilderValidationNilTenantId(t *testing.T) {
	_, err := NewBuilder(uuid.Nil, 12345).Build()
	if err == nil {
		t.Error("expected error for nil tenantId")
	}
	if err.Error() != "tenantId is required" {
		t.Errorf("expected 'tenantId is required' error, got %v", err)
	}
}

func TestBuilderValidationZeroCharacterId(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 0).Build()
	if err == nil {
		t.Error("expected error for zero characterId")
	}
	if err.Error() != "characterId is required" {
		t.Errorf("expected 'characterId is required' error, got %v", err)
	}
}

func TestBuilderValidationZeroCapacity(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 12345).
		SetCapacity(0).
		Build()
	if err == nil {
		t.Error("expected error for zero capacity")
	}
	if err.Error() != "capacity must be greater than 0" {
		t.Errorf("expected 'capacity must be greater than 0' error, got %v", err)
	}
}
