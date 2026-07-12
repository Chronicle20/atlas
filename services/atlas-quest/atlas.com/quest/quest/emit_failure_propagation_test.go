package quest_test

import (
	"atlas-quest/test"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// TestStart_PropagatesEnqueueFailure verifies the task-114 fix: when the
// outbox enqueue for the quest-started event fails, Start returns that
// error (instead of swallowing it and returning success) and no event is
// recorded. Before the fix, the emit error was logged and dropped, letting
// the transaction commit with no corresponding event enqueued.
func TestStart_PropagatesEnqueueFailure(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 5042
	const characterId uint32 = 12

	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))
	injectedErr := errors.New("simulated outbox enqueue failure")
	mockEmitter.StartedErr = injectedErr

	_, _, err := processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil)
	if err == nil {
		t.Fatalf("Start() error = nil, want propagated enqueue error")
	}
	if !errors.Is(err, injectedErr) {
		t.Errorf("Start() error = %v, want it to wrap %v", err, injectedErr)
	}
	if len(mockEmitter.StartedEvents) != 0 {
		t.Errorf("StartedEvents = %d, want 0 (enqueue failed, nothing should be recorded)", len(mockEmitter.StartedEvents))
	}
}

// TestComplete_PropagatesEnqueueFailure mirrors TestStart_PropagatesEnqueueFailure
// for the quest-completed status event.
func TestComplete_PropagatesEnqueueFailure(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 5043
	const characterId uint32 = 12

	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))
	if _, _, err := processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil); err != nil {
		t.Fatalf("Start() setup error: %v", err)
	}

	injectedErr := errors.New("simulated outbox enqueue failure")
	mockEmitter.CompletedErr = injectedErr

	_, err := processor.Complete(uuid.Nil, characterId, questId, capTestField(), true, nil)
	if err == nil {
		t.Fatalf("Complete() error = nil, want propagated enqueue error")
	}
	if !errors.Is(err, injectedErr) {
		t.Errorf("Complete() error = %v, want it to wrap %v", err, injectedErr)
	}
	if len(mockEmitter.CompletedEvents) != 0 {
		t.Errorf("CompletedEvents = %d, want 0 (enqueue failed, nothing should be recorded)", len(mockEmitter.CompletedEvents))
	}
}

// TestForfeit_PropagatesEnqueueFailure mirrors TestStart_PropagatesEnqueueFailure
// for the quest-forfeited status event.
func TestForfeit_PropagatesEnqueueFailure(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 5044
	const characterId uint32 = 12

	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))
	if _, _, err := processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil); err != nil {
		t.Fatalf("Start() setup error: %v", err)
	}

	injectedErr := errors.New("simulated outbox enqueue failure")
	mockEmitter.ForfeitedErr = injectedErr

	err := processor.Forfeit(uuid.Nil, characterId, questId)
	if err == nil {
		t.Fatalf("Forfeit() error = nil, want propagated enqueue error")
	}
	if !errors.Is(err, injectedErr) {
		t.Errorf("Forfeit() error = %v, want it to wrap %v", err, injectedErr)
	}
	if len(mockEmitter.ForfeitedEvents) != 0 {
		t.Errorf("ForfeitedEvents = %d, want 0 (enqueue failed, nothing should be recorded)", len(mockEmitter.ForfeitedEvents))
	}
}

// TestSetProgress_PropagatesEnqueueFailure covers the main (non-fallback)
// EmitProgressUpdated call site.
func TestSetProgress_PropagatesEnqueueFailure(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 5045
	const characterId uint32 = 12
	const mobId uint32 = 130101

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 5))
	if _, _, err := processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil); err != nil {
		t.Fatalf("Start() setup error: %v", err)
	}

	injectedErr := errors.New("simulated outbox enqueue failure")
	mockEmitter.ProgressErr = injectedErr

	err := processor.SetProgress(uuid.Nil, characterId, questId, mobId, "001")
	if err == nil {
		t.Fatalf("SetProgress() error = nil, want propagated enqueue error")
	}
	if !errors.Is(err, injectedErr) {
		t.Errorf("SetProgress() error = %v, want it to wrap %v", err, injectedErr)
	}
	if len(mockEmitter.ProgressEvents) != 0 {
		t.Errorf("ProgressEvents = %d, want 0 (enqueue failed, nothing should be recorded)", len(mockEmitter.ProgressEvents))
	}
}
