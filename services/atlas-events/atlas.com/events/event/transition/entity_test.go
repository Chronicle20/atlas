package transition

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransitionRoundTrip(t *testing.T) {
	occId := uuid.New()
	at := time.Date(2026, 8, 15, 12, 5, 3, 0, time.UTC)

	m, err := NewBuilder(occId, "ATTACKING").
		SetFromStage("").
		SetOccurredAt(at).
		SetTrigger(TriggerTypeScheduledWork, "work-1").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	e, err := ToEntity(m, uuid.New())
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.OccurrenceId() != occId || back.ToStage() != "ATTACKING" ||
		back.TriggerType() != TriggerTypeScheduledWork || back.TriggerReference() != "work-1" {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if !back.OccurredAt().Equal(at) {
		t.Fatalf("occurredAt = %s, want %s", back.OccurredAt(), at)
	}
}

// FR-T1: fromStage is nullable — the creation row has no prior stage.
func TestCreationTransitionHasNoFromStage(t *testing.T) {
	m, err := NewBuilder(uuid.New(), "ACTIVE").
		SetTrigger(TriggerTypeOccurrenceCreated, "work-1").Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.FromStage() != "" {
		t.Fatalf("FromStage = %q, want empty", m.FromStage())
	}
}

// FR-T3: a transition without a trigger type is not traceable, so it is not
// constructible.
func TestBuildRequiresTriggerType(t *testing.T) {
	if _, err := NewBuilder(uuid.New(), "ACTIVE").Build(); err == nil {
		t.Fatalf("expected an error when no trigger type was set")
	}
}
