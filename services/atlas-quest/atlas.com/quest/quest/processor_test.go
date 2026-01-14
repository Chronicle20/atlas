package quest_test

import (
	"atlas-quest/quest"
	"atlas-quest/test"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupProcessor(t *testing.T) (quest.Processor, *test.MockDataProcessor, *test.MockValidationProcessor, func()) {
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockEventEmitter := test.NewMockEventEmitter()

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, mockEventEmitter)

	cleanup := func() {
		test.CleanupTestDB(db)
	}

	return processor, mockData, mockValidation, cleanup
}

func createTestField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
}

// TestNewProcessor tests processor initialization
func TestNewProcessor(t *testing.T) {
	processor, _, _, cleanup := setupProcessor(t)
	defer cleanup()

	if processor == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

// TestGetById_NotFound tests fetching a non-existent quest
func TestGetById_NotFound(t *testing.T) {
	processor, _, _, cleanup := setupProcessor(t)
	defer cleanup()

	_, err := processor.GetById(999)
	if err == nil {
		t.Error("Expected error for non-existent quest")
	}
}

// TestStart_HappyPath tests starting a new quest successfully
func TestStart_HappyPath(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	model, failedConditions, err := processor.Start(characterId, questId, createTestField(), false)

	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if len(failedConditions) > 0 {
		t.Errorf("Start() unexpected failed conditions: %v", failedConditions)
	}
	if model.CharacterId() != characterId {
		t.Errorf("model.CharacterId() = %d, want %d", model.CharacterId(), characterId)
	}
	if model.QuestId() != questId {
		t.Errorf("model.QuestId() = %d, want %d", model.QuestId(), questId)
	}
	if model.State() != quest.StateStarted {
		t.Errorf("model.State() = %d, want %d", model.State(), quest.StateStarted)
	}
}

// TestStart_WithMobRequirements tests that progress is initialized for mob requirements
func TestStart_WithMobRequirements(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1001)
	characterId := uint32(12345)
	mobId := uint32(100100)

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	model, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	// Fetch the quest to check progress was initialized
	fetched, err := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId() unexpected error: %v", err)
	}

	progress, found := fetched.GetProgress(mobId)
	if !found {
		t.Error("Expected progress entry for mob ID to be initialized")
	}
	if progress.Progress() != "000" {
		t.Errorf("progress.Progress() = %s, want \"000\"", progress.Progress())
	}

	_ = model // Use model
}

// TestStart_WithMapRequirements tests that progress is initialized for map requirements
func TestStart_WithMapRequirements(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1002)
	characterId := uint32(12345)
	mapIds := []uint32{100000000, 100000001}

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMapRequirement(questId, mapIds))

	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	fetched, err := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId() unexpected error: %v", err)
	}

	for _, mapId := range mapIds {
		progress, found := fetched.GetProgress(mapId)
		if !found {
			t.Errorf("Expected progress entry for map ID %d to be initialized", mapId)
			continue
		}
		if progress.Progress() != "0" {
			t.Errorf("progress.Progress() for map %d = %s, want \"0\"", mapId, progress.Progress())
		}
	}
}

// TestStart_AlreadyStarted tests that starting an already started quest returns error
func TestStart_AlreadyStarted(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1003)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start the quest first time
	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("First Start() unexpected error: %v", err)
	}

	// Try to start again
	_, _, err = processor.Start(characterId, questId, createTestField(), false)
	if err != quest.ErrQuestAlreadyStarted {
		t.Errorf("Second Start() error = %v, want ErrQuestAlreadyStarted", err)
	}
}

// TestStart_ValidationFailed tests that validation failures are returned properly
func TestStart_ValidationFailed(t *testing.T) {
	processor, mockData, mockValidation, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1004)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Set up validation to fail
	mockValidation.StartValidationResult = false
	mockValidation.StartFailedConditions = []string{"level_too_low", "missing_item_123"}

	_, failedConditions, err := processor.Start(characterId, questId, createTestField(), false)

	if err != quest.ErrStartRequirementsNotMet {
		t.Errorf("Start() error = %v, want ErrStartRequirementsNotMet", err)
	}
	if len(failedConditions) != 2 {
		t.Errorf("len(failedConditions) = %d, want 2", len(failedConditions))
	}
}

// TestStart_SkipValidation tests that skipValidation bypasses validation
func TestStart_SkipValidation(t *testing.T) {
	processor, mockData, mockValidation, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(1005)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Set up validation to fail
	mockValidation.StartValidationResult = false
	mockValidation.StartFailedConditions = []string{"level_too_low"}

	// Start with skipValidation = true
	model, _, err := processor.Start(characterId, questId, createTestField(), true)

	if err != nil {
		t.Fatalf("Start() with skipValidation=true unexpected error: %v", err)
	}
	if model.State() != quest.StateStarted {
		t.Errorf("model.State() = %d, want StateStarted", model.State())
	}
}

// TestComplete_HappyPath tests completing a quest successfully
func TestComplete_HappyPath(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(2000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start the quest first
	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	// Complete the quest
	nextQuestId, err := processor.Complete(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Complete() unexpected error: %v", err)
	}
	if nextQuestId != 0 {
		t.Errorf("nextQuestId = %d, want 0 (no chain)", nextQuestId)
	}

	// Verify state changed
	fetched, err := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId() unexpected error: %v", err)
	}
	if fetched.State() != quest.StateCompleted {
		t.Errorf("fetched.State() = %d, want StateCompleted", fetched.State())
	}
	if fetched.CompletedCount() != 1 {
		t.Errorf("fetched.CompletedCount() = %d, want 1", fetched.CompletedCount())
	}
}

// TestComplete_WithChain tests completing a quest that has a next quest
func TestComplete_WithChain(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(2001)
	nextQuestIdExpected := uint32(2002)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateQuestWithChain(questId, nextQuestIdExpected))

	// Start and complete
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)
	nextQuestId, err := processor.Complete(characterId, questId, createTestField(), false)

	if err != nil {
		t.Fatalf("Complete() unexpected error: %v", err)
	}
	if nextQuestId != nextQuestIdExpected {
		t.Errorf("nextQuestId = %d, want %d", nextQuestId, nextQuestIdExpected)
	}
}

// TestComplete_NotStarted tests that completing a non-started quest returns error
func TestComplete_NotStarted(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(2002)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	_, err := processor.Complete(characterId, questId, createTestField(), false)
	if err == nil {
		t.Error("Complete() expected error for non-started quest")
	}
}

// TestComplete_ValidationFailed tests that end validation failures are handled
func TestComplete_ValidationFailed(t *testing.T) {
	processor, mockData, mockValidation, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(2003)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Set up end validation to fail
	mockValidation.EndValidationResult = false
	mockValidation.EndFailedConditions = []string{"missing_item"}

	_, err := processor.Complete(characterId, questId, createTestField(), false)
	if err != quest.ErrEndRequirementsNotMet {
		t.Errorf("Complete() error = %v, want ErrEndRequirementsNotMet", err)
	}
}

// TestForfeit_HappyPath tests forfeiting a quest successfully
func TestForfeit_HappyPath(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(3000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start the quest
	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	// Forfeit the quest
	err = processor.Forfeit(characterId, questId)
	if err != nil {
		t.Fatalf("Forfeit() unexpected error: %v", err)
	}

	// Verify state changed
	fetched, err := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId() unexpected error: %v", err)
	}
	if fetched.State() != quest.StateNotStarted {
		t.Errorf("fetched.State() = %d, want StateNotStarted", fetched.State())
	}
	if fetched.ForfeitCount() != 1 {
		t.Errorf("fetched.ForfeitCount() = %d, want 1", fetched.ForfeitCount())
	}
}

// TestForfeit_ClearsProgress tests that forfeiting clears progress
func TestForfeit_ClearsProgress(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(3001)
	characterId := uint32(12345)
	mobId := uint32(100100)
	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	// Start the quest (progress initialized)
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Update progress
	_ = processor.SetProgress(characterId, questId, mobId, "005")

	// Forfeit
	_ = processor.Forfeit(characterId, questId)

	// Verify progress is cleared
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if len(fetched.Progress()) != 0 {
		t.Errorf("len(fetched.Progress()) = %d, want 0 (cleared)", len(fetched.Progress()))
	}
}

// TestForfeit_NotStarted tests that forfeiting a non-started quest returns error
func TestForfeit_NotStarted(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(3002)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	err := processor.Forfeit(characterId, questId)
	if err == nil {
		t.Error("Forfeit() expected error for non-started quest")
	}
}

// TestSetProgress_UpdateExisting tests updating existing progress
func TestSetProgress_UpdateExisting(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(4000)
	characterId := uint32(12345)
	mobId := uint32(100100)
	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	// Start the quest (progress initialized to "000")
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Update progress
	err := processor.SetProgress(characterId, questId, mobId, "005")
	if err != nil {
		t.Fatalf("SetProgress() unexpected error: %v", err)
	}

	// Verify progress updated
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(mobId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "005" {
		t.Errorf("progress.Progress() = %s, want \"005\"", progress.Progress())
	}
}

// TestSetProgress_CreateNew tests creating a new progress entry
func TestSetProgress_CreateNew(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(4001)
	characterId := uint32(12345)
	infoNumber := uint32(999999)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start the quest (no initial progress)
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Create new progress entry
	err := processor.SetProgress(characterId, questId, infoNumber, "1")
	if err != nil {
		t.Fatalf("SetProgress() unexpected error: %v", err)
	}

	// Verify progress created
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(infoNumber)
	if !found {
		t.Fatal("Expected progress entry to be created")
	}
	if progress.Progress() != "1" {
		t.Errorf("progress.Progress() = %s, want \"1\"", progress.Progress())
	}
}

// TestSetProgress_NotStarted tests that setting progress on non-started quest fails
func TestSetProgress_NotStarted(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(4002)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	err := processor.SetProgress(characterId, questId, 100, "1")
	if err == nil {
		t.Error("SetProgress() expected error for non-started quest")
	}
}

// TestGetByCharacterId_ReturnsAllQuests tests fetching all quests for a character
func TestGetByCharacterId_ReturnsAllQuests(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	questIds := []uint32{5000, 5001, 5002}

	for _, qid := range questIds {
		mockData.AddQuestDefinition(qid, test.CreateSimpleQuestDefinition(qid))
		_, _, _ = processor.Start(characterId, qid, createTestField(), false)
	}

	quests, err := processor.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("GetByCharacterId() unexpected error: %v", err)
	}
	if len(quests) != len(questIds) {
		t.Errorf("len(quests) = %d, want %d", len(quests), len(questIds))
	}
}

// TestGetByCharacterIdAndState_FiltersCorrectly tests fetching quests by state
func TestGetByCharacterIdAndState_FiltersCorrectly(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)

	// Create 3 quests: 2 started, 1 completed
	mockData.AddQuestDefinition(6000, test.CreateSimpleQuestDefinition(6000))
	mockData.AddQuestDefinition(6001, test.CreateSimpleQuestDefinition(6001))
	mockData.AddQuestDefinition(6002, test.CreateSimpleQuestDefinition(6002))

	_, _, _ = processor.Start(characterId, 6000, createTestField(), false)
	_, _, _ = processor.Start(characterId, 6001, createTestField(), false)
	_, _, _ = processor.Start(characterId, 6002, createTestField(), false)

	// Complete one quest
	_, _ = processor.Complete(characterId, 6002, createTestField(), true)

	// Get started quests
	started, err := processor.GetByCharacterIdAndState(characterId, quest.StateStarted)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndState(StateStarted) error: %v", err)
	}
	if len(started) != 2 {
		t.Errorf("len(started) = %d, want 2", len(started))
	}

	// Get completed quests
	completed, err := processor.GetByCharacterIdAndState(characterId, quest.StateCompleted)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndState(StateCompleted) error: %v", err)
	}
	if len(completed) != 1 {
		t.Errorf("len(completed) = %d, want 1", len(completed))
	}
}

// TestTenantIsolation tests that quests are isolated by tenant
func TestTenantIsolation(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockEventEmitter := test.NewMockEventEmitter()

	questId := uint32(7000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Create quest with tenant 1
	tenant1Id := uuid.New()
	ctx1 := test.CreateTestContextWithTenant(tenant1Id)
	processor1 := quest.NewProcessorWithDependencies(logger, ctx1, db, mockData, mockValidation, mockEventEmitter)
	_, _, _ = processor1.Start(characterId, questId, createTestField(), false)

	// Try to access from tenant 2
	tenant2Id := uuid.New()
	ctx2 := test.CreateTestContextWithTenant(tenant2Id)
	processor2 := quest.NewProcessorWithDependencies(logger, ctx2, db, mockData, mockValidation, mockEventEmitter)

	// Tenant 2 should not see tenant 1's quest
	_, err := processor2.GetByCharacterIdAndQuestId(characterId, questId)
	if err == nil {
		t.Error("Expected error when accessing quest from different tenant")
	}

	// Tenant 2 should be able to start their own quest
	_, _, err = processor2.Start(characterId, questId, createTestField(), false)
	if err != nil {
		t.Fatalf("Tenant 2 Start() unexpected error: %v", err)
	}
}

// TestCheckAutoComplete_Completes tests auto-completion when conditions met
func TestCheckAutoComplete_Completes(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(8000)
	characterId := uint32(12345)
	mobId := uint32(100100)

	def := test.CreateQuestWithMobRequirement(questId, mobId, 5)
	def.AutoComplete = true
	mockData.AddQuestDefinition(questId, def)

	// Start quest
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Set progress to meet requirements
	_ = processor.SetProgress(characterId, questId, mobId, "005")

	// Check auto-complete
	nextQuestId, completed, err := processor.CheckAutoComplete(characterId, questId, createTestField())
	if err != nil {
		t.Fatalf("CheckAutoComplete() error: %v", err)
	}
	if !completed {
		t.Error("Expected quest to be auto-completed")
	}
	if nextQuestId != 0 {
		t.Errorf("nextQuestId = %d, want 0", nextQuestId)
	}

	// Verify state changed
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if fetched.State() != quest.StateCompleted {
		t.Errorf("fetched.State() = %d, want StateCompleted", fetched.State())
	}
}

// TestCheckAutoComplete_NotAutoComplete tests that non-auto-complete quests are not completed
func TestCheckAutoComplete_NotAutoComplete(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(8001)
	characterId := uint32(12345)

	def := test.CreateSimpleQuestDefinition(questId)
	def.AutoComplete = false
	mockData.AddQuestDefinition(questId, def)

	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	_, completed, err := processor.CheckAutoComplete(characterId, questId, createTestField())
	if err != nil {
		t.Fatalf("CheckAutoComplete() error: %v", err)
	}
	if completed {
		t.Error("Expected quest NOT to be auto-completed (AutoComplete=false)")
	}
}

// TestCheckAutoComplete_RequirementsNotMet tests that incomplete quests are not auto-completed
func TestCheckAutoComplete_RequirementsNotMet(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(8002)
	characterId := uint32(12345)
	mobId := uint32(100100)

	def := test.CreateQuestWithMobRequirement(questId, mobId, 10) // Need 10 kills
	def.AutoComplete = true
	mockData.AddQuestDefinition(questId, def)

	_, _, _ = processor.Start(characterId, questId, createTestField(), false)

	// Progress is still 0
	_, completed, _ := processor.CheckAutoComplete(characterId, questId, createTestField())
	if completed {
		t.Error("Expected quest NOT to be auto-completed (requirements not met)")
	}
}

// TestDeleteByCharacterId tests deleting all quests for a character
func TestDeleteByCharacterId(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	questIds := []uint32{9000, 9001, 9002}

	for _, qid := range questIds {
		mockData.AddQuestDefinition(qid, test.CreateSimpleQuestDefinition(qid))
		_, _, _ = processor.Start(characterId, qid, createTestField(), false)
	}

	err := processor.DeleteByCharacterId(characterId)
	if err != nil {
		t.Fatalf("DeleteByCharacterId() error: %v", err)
	}

	quests, _ := processor.GetByCharacterId(characterId)
	if len(quests) != 0 {
		t.Errorf("len(quests) = %d, want 0 after deletion", len(quests))
	}
}

// TestRepeatableQuest_RestartAfterInterval tests restarting a repeatable quest
func TestRepeatableQuest_RestartAfterInterval(t *testing.T) {
	// This test is more of an integration test - we can't easily test time-based logic
	// without manipulating time. We'll just verify the error is returned correctly
	// when interval has NOT elapsed.

	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(10000)
	characterId := uint32(12345)

	// Interval of 60 minutes
	mockData.AddQuestDefinition(questId, test.CreateRepeatableQuest(questId, 60))

	// Start and complete
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)
	_, _ = processor.Complete(characterId, questId, createTestField(), true)

	// Try to restart immediately - should fail (interval not elapsed)
	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != quest.ErrIntervalNotElapsed {
		t.Errorf("Start() error = %v, want ErrIntervalNotElapsed", err)
	}
}

// TestNonRepeatableQuest_CannotRestart tests that non-repeatable quests cannot restart
func TestNonRepeatableQuest_CannotRestart(t *testing.T) {
	processor, mockData, _, cleanup := setupProcessor(t)
	defer cleanup()

	questId := uint32(10001)
	characterId := uint32(12345)

	// No interval = non-repeatable
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Start and complete
	_, _, _ = processor.Start(characterId, questId, createTestField(), false)
	_, _ = processor.Complete(characterId, questId, createTestField(), true)

	// Try to restart - should fail
	_, _, err := processor.Start(characterId, questId, createTestField(), false)
	if err != quest.ErrQuestAlreadyCompleted {
		t.Errorf("Start() error = %v, want ErrQuestAlreadyCompleted", err)
	}
}

// TestTimeLimitedQuest_CannotCompleteAfterExpiration tests expired quest behavior
func TestTimeLimitedQuest_CannotCompleteAfterExpiration(t *testing.T) {
	// This is difficult to test without time manipulation
	// We'll skip for now as it requires either mocking time or waiting
	t.Skip("Time-based test requires time manipulation")
}

// TestWithTransaction tests that processor works with transactions
func TestWithTransaction(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockEventEmitter := test.NewMockEventEmitter()

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, mockEventEmitter)

	questId := uint32(11000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	// Use WithTransaction
	tx := db.Begin()
	txProcessor := processor.WithTransaction(tx)

	_, _, err := txProcessor.Start(characterId, questId, createTestField(), false)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Start() in transaction error: %v", err)
	}

	tx.Commit()

	// Verify quest exists after commit
	_, err = processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Errorf("Quest not found after transaction commit: %v", err)
	}
}

// Benchmark tests
func BenchmarkStart(b *testing.B) {
	db := test.SetupTestDB(&testing.T{})
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockEventEmitter := test.NewMockEventEmitter()

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, mockEventEmitter)

	f := createTestField()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		questId := uint32(i + 100000)
		characterId := uint32(i + 1)
		mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))
		_, _, _ = processor.Start(characterId, questId, f, true)
	}
}
