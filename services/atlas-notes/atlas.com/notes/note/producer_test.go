package note

import (
	"encoding/json"
	"testing"
	"time"

	note2 "atlas-notes/kafka/message/note"

	"github.com/google/uuid"
)

func TestCreateNoteStatusEventProviderCarriesTransactionId(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateNoteStatusEventProvider(txn, 200, 7, 100, "hello", 1, time.Now())()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}
	var e note2.StatusEvent[note2.StatusEventCreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", e.TransactionId, txn)
	}
	if e.Type != note2.StatusEventTypeCreated {
		t.Errorf("type: got %s, want %s", e.Type, note2.StatusEventTypeCreated)
	}
	if e.CharacterId != 200 || e.Body.NoteId != 7 || e.Body.SenderId != 100 {
		t.Errorf("body mismatch: %+v", e)
	}
}

func TestCreateFailedStatusEventProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateFailedStatusEventProvider(txn, 200, 100, "db down")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var e note2.StatusEvent[note2.StatusEventCreateFailedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", e.TransactionId, txn)
	}
	if e.Type != note2.StatusEventTypeCreateFailed {
		t.Errorf("type: got %s, want %s", e.Type, note2.StatusEventTypeCreateFailed)
	}
	if e.Body.SenderId != 100 || e.Body.Reason != "db down" {
		t.Errorf("body mismatch: %+v", e.Body)
	}
}
