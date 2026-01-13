package buddy

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuild(t *testing.T) {
	listId := uuid.New()
	characterId := uint32(12345)

	m, err := NewBuilder(listId, characterId).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.listId != listId {
		t.Errorf("expected listId %v, got %v", listId, m.listId)
	}
	if m.characterId != characterId {
		t.Errorf("expected characterId %d, got %d", characterId, m.characterId)
	}
	if m.group != "Default Group" {
		t.Errorf("expected default group 'Default Group', got %s", m.group)
	}
	if m.channelId != -1 {
		t.Errorf("expected default channelId -1, got %d", m.channelId)
	}
	if m.inShop != false {
		t.Errorf("expected default inShop false, got %v", m.inShop)
	}
	if m.pending != false {
		t.Errorf("expected default pending false, got %v", m.pending)
	}
}

func TestBuilderWithAllFields(t *testing.T) {
	listId := uuid.New()
	characterId := uint32(12345)

	m, err := NewBuilder(listId, characterId).
		SetGroup("Friends").
		SetCharacterName("TestPlayer").
		SetChannelId(5).
		SetInShop(true).
		SetPending(true).
		Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.group != "Friends" {
		t.Errorf("expected group 'Friends', got %s", m.group)
	}
	if m.characterName != "TestPlayer" {
		t.Errorf("expected characterName 'TestPlayer', got %s", m.characterName)
	}
	if m.channelId != 5 {
		t.Errorf("expected channelId 5, got %d", m.channelId)
	}
	if m.inShop != true {
		t.Errorf("expected inShop true, got %v", m.inShop)
	}
	if m.pending != true {
		t.Errorf("expected pending true, got %v", m.pending)
	}
}

func TestBuilderValidationNilListId(t *testing.T) {
	_, err := NewBuilder(uuid.Nil, 12345).Build()
	if err == nil {
		t.Error("expected error for nil listId")
	}
	if err.Error() != "listId is required" {
		t.Errorf("expected 'listId is required' error, got %v", err)
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

func TestBuilderFluentChaining(t *testing.T) {
	listId := uuid.New()
	characterId := uint32(12345)

	builder := NewBuilder(listId, characterId)

	// Verify fluent chaining returns the same builder
	result := builder.
		SetGroup("Group1").
		SetCharacterName("Name1").
		SetChannelId(1).
		SetInShop(true).
		SetPending(false)

	if result != builder {
		t.Error("fluent methods should return the same builder instance")
	}
}
