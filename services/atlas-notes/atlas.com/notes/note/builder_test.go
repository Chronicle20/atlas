package note_test

import (
	"atlas-notes/note"
	"testing"
	"time"
)

func TestBuilder_Build_Valid(t *testing.T) {
	m, err := note.NewBuilder().
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("Hello").
		SetFlag(0).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if m.CharacterId() != 1 {
		t.Fatalf("Expected characterId 1, got %d", m.CharacterId())
	}
	if m.SenderId() != 2 {
		t.Fatalf("Expected senderId 2, got %d", m.SenderId())
	}
	if m.Message() != "Hello" {
		t.Fatalf("Expected message 'Hello', got '%s'", m.Message())
	}
}

func TestBuilder_Build_MissingCharacterId(t *testing.T) {
	_, err := note.NewBuilder().
		SetSenderId(2).
		SetMessage("Hello").
		Build()

	if err == nil {
		t.Fatalf("Expected error for missing characterId")
	}
	if err.Error() != "characterId is required" {
		t.Fatalf("Expected 'characterId is required' error, got: %v", err)
	}
}

func TestBuilder_Build_MissingSenderId(t *testing.T) {
	_, err := note.NewBuilder().
		SetCharacterId(1).
		SetMessage("Hello").
		Build()

	if err == nil {
		t.Fatalf("Expected error for missing senderId")
	}
	if err.Error() != "senderId is required" {
		t.Fatalf("Expected 'senderId is required' error, got: %v", err)
	}
}

func TestBuilder_Build_MissingMessage(t *testing.T) {
	_, err := note.NewBuilder().
		SetCharacterId(1).
		SetSenderId(2).
		Build()

	if err == nil {
		t.Fatalf("Expected error for missing message")
	}
	if err.Error() != "message is required" {
		t.Fatalf("Expected 'message is required' error, got: %v", err)
	}
}

func TestBuilder_Build_EmptyMessage(t *testing.T) {
	_, err := note.NewBuilder().
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("").
		Build()

	if err == nil {
		t.Fatalf("Expected error for empty message")
	}
	if err.Error() != "message is required" {
		t.Fatalf("Expected 'message is required' error, got: %v", err)
	}
}

func TestBuilder_Build_AllFields(t *testing.T) {
	timestamp := time.Now()
	m, err := note.NewBuilder().
		SetId(123).
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("Full note").
		SetFlag(5).
		SetTimestamp(timestamp).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if m.Id() != 123 {
		t.Fatalf("Expected id 123, got %d", m.Id())
	}
	if m.CharacterId() != 1 {
		t.Fatalf("Expected characterId 1, got %d", m.CharacterId())
	}
	if m.SenderId() != 2 {
		t.Fatalf("Expected senderId 2, got %d", m.SenderId())
	}
	if m.Message() != "Full note" {
		t.Fatalf("Expected message 'Full note', got '%s'", m.Message())
	}
	if m.Flag() != 5 {
		t.Fatalf("Expected flag 5, got %d", m.Flag())
	}
	if !m.Timestamp().Equal(timestamp) {
		t.Fatalf("Expected timestamp %v, got %v", timestamp, m.Timestamp())
	}
}
