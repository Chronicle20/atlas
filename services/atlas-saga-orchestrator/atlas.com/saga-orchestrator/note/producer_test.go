package note

import (
	"encoding/json"
	"testing"

	note2 "atlas-saga-orchestrator/kafka/message/note"

	"github.com/google/uuid"
)

func TestCreateNoteCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateNoteCommandProvider(txn, 200, 100, "hello", 1, true)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}
	var c note2.Command[note2.CommandCreateBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", c.TransactionId, txn)
	}
	if c.Type != note2.CommandTypeCreate {
		t.Errorf("type: got %s, want %s", c.Type, note2.CommandTypeCreate)
	}
	if c.CharacterId != 200 {
		t.Errorf("characterId (receiver): got %d, want 200", c.CharacterId)
	}
	if c.Body.SenderId != 100 || c.Body.Message != "hello" || c.Body.Flag != 1 || !c.Body.GiftNote {
		t.Errorf("body mismatch: %+v", c.Body)
	}
}
