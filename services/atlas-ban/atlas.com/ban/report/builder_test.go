package report

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuildsValidReport(t *testing.T) {
	tenantId := uuid.New()
	chatLog := "alice: hi"
	m, err := NewBuilder(tenantId, KindClaim, 1).
		SetReporterName("Reporter").
		SetAccusedId(2).
		SetAccusedName("Accused").
		SetReasonType(3).
		SetDescription("harassment").
		SetChatLog(&chatLog).
		SetServerTranscript([]TranscriptLine{{Timestamp: 1, SenderId: 1, SenderName: "Reporter", ChatType: "GENERAL", Text: "hi"}}).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.TenantId() != tenantId || m.Kind() != KindClaim || m.ReporterId() != 1 {
		t.Errorf("core fields mismatch: %+v", m)
	}
	if m.Status() != StatusOpen {
		t.Errorf("default status: got %s want %s", m.Status(), StatusOpen)
	}
	if m.ChatLog() == nil || *m.ChatLog() != chatLog {
		t.Errorf("chat log mismatch")
	}
	if len(m.ServerTranscript()) != 1 || m.ServerTranscript()[0].Text != "hi" {
		t.Errorf("transcript mismatch: %+v", m.ServerTranscript())
	}
}

func TestBuilderRejectsInvalidKind(t *testing.T) {
	_, err := NewBuilder(uuid.New(), Kind("bogus"), 1).SetAccusedName("x").Build()
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestBuilderRejectsInvalidStatus(t *testing.T) {
	_, err := NewBuilder(uuid.New(), KindSue, 1).SetAccusedName("x").SetStatus(Status("bogus")).Build()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusReviewed, StatusActioned} {
		if !s.Valid() {
			t.Errorf("expected %s valid", s)
		}
	}
	if Status("closed").Valid() {
		t.Error("expected 'closed' invalid")
	}
}
