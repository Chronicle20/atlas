package quest_test

import (
	"atlas-quest/quest"
	"atlas-quest/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// setupProcessorWithEmitter is a setup variant that exposes the mock event
// emitter so tests can assert on EmitProgressUpdated invocations — the key
// observable for verifying the skip-when-already-met behavior.
func setupProcessorWithEmitter(t *testing.T) (quest.Processor, *test.MockDataProcessor, *test.MockEventEmitter, func()) {
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockEmitter := test.NewMockEventEmitter()

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, mockEmitter)

	return processor, mockData, mockEmitter, func() { test.CleanupTestDB(db) }
}

func capTestField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
}

// TestSetProgress_SkipsWriteAndEmitWhenMobRequirementAlreadyMet verifies that
// after a mob's kill requirement is satisfied, further SetProgress calls for
// the same infoNumber don't write to the DB or emit a progress-updated
// event. This is the root-cause fix for duplicate "quest complete" packets
// observed against quest 1042 in production: autoComplete=false quests stay
// in StateStarted after the cap is reached, and every overflow kill was
// re-emitting a progress-updated Kafka event.
func TestSetProgress_SkipsWriteAndEmitWhenMobRequirementAlreadyMet(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 1042
	const characterId uint32 = 12
	const mobId uint32 = 130101
	const required uint32 = 5

	def := test.CreateQuestWithMobRequirement(questId, mobId, required)
	def.AutoComplete = false
	mockData.AddQuestDefinition(questId, def)
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil)

	// Bring progress up to the cap exactly.
	if err := processor.SetProgress(uuid.Nil, characterId, questId, mobId, "005"); err != nil {
		t.Fatalf("SetProgress (reach cap) error: %v", err)
	}

	emitsAfterReachingCap := len(mockEmitter.ProgressEvents)

	// Overflow attempts: these are what the monster consumer generates when
	// the player keeps killing. Each MUST be skipped.
	for _, overflow := range []string{"006", "007", "999"} {
		if err := processor.SetProgress(uuid.Nil, characterId, questId, mobId, overflow); err != nil {
			t.Fatalf("SetProgress (overflow=%s) error: %v (skip should succeed silently)", overflow, err)
		}
	}

	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, _ := fetched.GetProgress(mobId)
	if progress.Progress() != "005" {
		t.Errorf("progress after overflow attempts = %s, want \"005\" (capped)", progress.Progress())
	}

	if len(mockEmitter.ProgressEvents) != emitsAfterReachingCap {
		t.Errorf("ProgressEvents count after overflow attempts = %d, want %d (no new emits)", len(mockEmitter.ProgressEvents), emitsAfterReachingCap)
	}
}

// TestSetProgress_SkipsEmitWhenMapVisitAlreadyRecorded verifies the same
// guard protects field-enter progress. Re-entering a tracked map previously
// re-emitted a progress-updated event on every entry (idempotent DB write,
// but atlas-channel still forwarded a quest-record-update packet to the
// client each time).
func TestSetProgress_SkipsEmitWhenMapVisitAlreadyRecorded(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 2042
	const characterId uint32 = 12
	const mapId uint32 = 100000000

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMapRequirement(questId, []uint32{mapId}))
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil)

	// First visit marks the map as entered.
	if err := processor.SetProgress(uuid.Nil, characterId, questId, mapId, "1"); err != nil {
		t.Fatalf("SetProgress (first visit) error: %v", err)
	}
	emitsAfterFirstVisit := len(mockEmitter.ProgressEvents)

	// Subsequent re-entries must not re-emit.
	for i := 0; i < 3; i++ {
		if err := processor.SetProgress(uuid.Nil, characterId, questId, mapId, "1"); err != nil {
			t.Fatalf("SetProgress (re-entry %d) error: %v", i, err)
		}
	}

	if len(mockEmitter.ProgressEvents) != emitsAfterFirstVisit {
		t.Errorf("ProgressEvents count after re-entries = %d, want %d (no new emits)", len(mockEmitter.ProgressEvents), emitsAfterFirstVisit)
	}
}

// TestSetProgress_StillWritesAndEmitsBelowCap is a guard against over-eager
// skipping: the cap must only apply once the requirement is met. Writes
// below the cap still need to land in the DB and emit a progress event.
func TestSetProgress_StillWritesAndEmitsBelowCap(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 3042
	const characterId uint32 = 12
	const mobId uint32 = 130101

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil)

	emitsBefore := len(mockEmitter.ProgressEvents)

	for _, v := range []string{"001", "002", "003"} {
		if err := processor.SetProgress(uuid.Nil, characterId, questId, mobId, v); err != nil {
			t.Fatalf("SetProgress(%s) error: %v", v, err)
		}
	}

	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, _ := fetched.GetProgress(mobId)
	if progress.Progress() != "003" {
		t.Errorf("progress = %s, want \"003\"", progress.Progress())
	}

	if got := len(mockEmitter.ProgressEvents) - emitsBefore; got != 3 {
		t.Errorf("new ProgressEvents = %d, want 3", got)
	}
}

// TestSetProgress_DoesNotSkipForUnrelatedInfoNumber verifies that the cap
// doesn't misfire for an infoNumber that doesn't map to any end-requirement
// on this quest. Such calls should fall through to the normal write path.
func TestSetProgress_DoesNotSkipForUnrelatedInfoNumber(t *testing.T) {
	processor, mockData, mockEmitter, cleanup := setupProcessorWithEmitter(t)
	defer cleanup()

	const questId uint32 = 4042
	const characterId uint32 = 12
	const trackedMob uint32 = 130101
	const unrelatedInfoNumber uint32 = 999999

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, trackedMob, 5))
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, capTestField(), true, nil)

	emitsBefore := len(mockEmitter.ProgressEvents)

	if err := processor.SetProgress(uuid.Nil, characterId, questId, unrelatedInfoNumber, "1"); err != nil {
		t.Fatalf("SetProgress error: %v", err)
	}

	if got := len(mockEmitter.ProgressEvents) - emitsBefore; got != 1 {
		t.Errorf("new ProgressEvents = %d, want 1 (unrelated infoNumber must still emit)", got)
	}
}
