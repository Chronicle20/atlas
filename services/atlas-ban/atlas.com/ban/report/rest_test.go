package report

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransformRoundTrip(t *testing.T) {
	chatLog := "log"
	m, err := NewBuilder(uuid.New(), KindClaim, 1).
		SetId(uuid.New()).
		SetReporterName("R").
		SetAccusedId(2).
		SetAccusedName("A").
		SetReasonType(3).
		SetDescription("d").
		SetChatLog(&chatLog).
		SetServerTranscript([]TranscriptLine{{Timestamp: 5, SenderId: 1, SenderName: "R", ChatType: "GENERAL", Text: "hi"}}).
		SetStatus(StatusReviewed).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.GetName() != "reports" {
		t.Errorf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != m.Id().String() {
		t.Errorf("id: %s", rm.GetID())
	}
	if rm.Kind != "claim" || rm.Status != "reviewed" || rm.ChatLog == nil || len(rm.ServerTranscript) != 1 {
		t.Errorf("attributes mismatch: %+v", rm)
	}
}

func TestRestModelSetIDRejectsGarbage(t *testing.T) {
	rm := &RestModel{}
	if err := rm.SetID("not-a-uuid"); err == nil {
		t.Fatal("expected error")
	}
	id := uuid.New()
	if err := rm.SetID(id.String()); err != nil || rm.Id != id {
		t.Fatalf("SetID: %v", err)
	}
}
