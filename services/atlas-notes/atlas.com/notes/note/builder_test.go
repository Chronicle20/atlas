package note_test

import (
	"atlas-notes/note"
	"testing"
	"time"

	"github.com/google/uuid"
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

// TestBuilder_SetGiftNote proves the builder threads GiftNote into the built Model.
func TestBuilder_SetGiftNote(t *testing.T) {
	m, err := note.NewBuilder().
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("thanks!").
		SetGiftNote(true).
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !m.GiftNote() {
		t.Fatalf("Expected GiftNote true, got false")
	}
}

// TestMakeEntityRoundTrip_GiftNote proves GiftNote survives a Model -> Entity -> Model round trip
// through MakeEntity/Make, so the server-only marker is not lost on persistence.
func TestMakeEntityRoundTrip_GiftNote(t *testing.T) {
	timestamp := time.Now()
	m, err := note.NewBuilder().
		SetId(1).
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("thanks!").
		SetFlag(0).
		SetTimestamp(timestamp).
		SetGiftNote(true).
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	e := note.MakeEntity(uuid.New(), m)
	if !e.GiftNote {
		t.Fatalf("Expected entity GiftNote true, got false")
	}

	back, err := note.Make(e)
	if err != nil {
		t.Fatalf("Expected no error from Make, got: %v", err)
	}
	if !back.GiftNote() {
		t.Fatalf("Expected round-tripped model GiftNote true, got false")
	}

	// A plain note (GiftNote unset) must not pick up a stray true on round trip.
	m2, err := note.NewBuilder().
		SetCharacterId(1).
		SetSenderId(2).
		SetMessage("hi").
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	e2 := note.MakeEntity(uuid.New(), m2)
	if e2.GiftNote {
		t.Fatalf("Expected entity GiftNote false for plain note, got true")
	}
	back2, err := note.Make(e2)
	if err != nil {
		t.Fatalf("Expected no error from Make, got: %v", err)
	}
	if back2.GiftNote() {
		t.Fatalf("Expected round-tripped plain-note model GiftNote false, got true")
	}
}
