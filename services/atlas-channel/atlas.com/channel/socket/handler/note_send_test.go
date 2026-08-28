package handler

import (
	"atlas-channel/saga"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBuildNoteSendSaga pins the FR-5 invariant: destroy-first, exactly two
// steps, correct payloads, flag 1.
func TestBuildNoteSendSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildNoteSendSaga(txn, now, 100, 5090000, 200, "hello")

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.NoteSend {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.NoteSend)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps: got %d, want 2", len(s.Steps))
	}

	if s.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action: got %s, want %s (destroy-first is mandatory)", s.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := s.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type: %T", s.Steps[0].Payload)
	}
	if dp.CharacterId != 100 || dp.TemplateId != 5090000 || dp.Quantity != 1 || dp.RemoveAll {
		t.Errorf("destroy payload mismatch: %+v", dp)
	}

	if s.Steps[1].Action != saga.CreateNote {
		t.Errorf("step 2 action: got %s, want %s", s.Steps[1].Action, saga.CreateNote)
	}
	np, ok := s.Steps[1].Payload.(saga.CreateNotePayload)
	if !ok {
		t.Fatalf("step 2 payload type: %T", s.Steps[1].Payload)
	}
	// Flag must be 0 (plain note). Non-zero memo flags select the client's
	// reward/gift render templates (the "gained fame" line) — see note_send.go.
	if np.SenderId != 100 || np.ReceiverId != 200 || np.Message != "hello" || np.Flag != 0 {
		t.Errorf("create-note payload mismatch: %+v", np)
	}
	if np.GiftNote {
		t.Errorf("create-note payload: GiftNote = true, want false (ordinary player note keeps the discard fame rule)")
	}
}
