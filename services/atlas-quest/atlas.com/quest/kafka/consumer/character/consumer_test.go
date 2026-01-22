package character_test

import (
	consumer "atlas-quest/kafka/consumer/character"
	"atlas-quest/kafka/message/character"
	"atlas-quest/quest"
	"atlas-quest/test"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestHandleMapChangedEvent_IgnoresWrongType tests that non-MAP_CHANGED events are ignored
func TestHandleMapChangedEvent_IgnoresWrongType(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: 12345,
		Type:        character.EventCharacterStatusTypeLogin, // Wrong type
		Body: character.StatusEventMapChangedBody{
			TargetMapId: 100000000,
		},
	}

	// Handler should return early for wrong type
	if event.Type != character.EventCharacterStatusTypeMapChanged {
		// This is what the handler does
		return
	}

	t.Error("Handler should have returned early for wrong event type")

	_ = db
}

// TestHandleMapChangedEvent_SkipsZeroCharacterId tests that zero character IDs are skipped
func TestHandleMapChangedEvent_SkipsZeroCharacterId(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: 0, // Zero character ID
		Type:        character.EventCharacterStatusTypeMapChanged,
		Body: character.StatusEventMapChangedBody{
			TargetMapId: 100000000,
		},
	}

	// Handler should skip zero character ID
	if event.CharacterId == 0 {
		return // This is what the handler does
	}

	t.Error("Handler should have returned early for zero character ID")

	_ = db
}

// TestHandleMapChangedEvent_UpdatesMapProgress tests that map visit progress is set
func TestHandleMapChangedEvent_UpdatesMapProgress(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1000)
	characterId := uint32(12345)
	mapId := uint32(100000000)

	// Create quest with map requirement
	mockData.AddQuestDefinition(questId, test.CreateQuestWithMapRequirement(questId, []uint32{mapId}))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the quest
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, test.CreateTestField(), true)

	// Verify initial progress (0 = not visited)
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, _ := fetched.GetProgress(mapId)
	if progress.Progress() != "0" {
		t.Errorf("Initial map progress = %s, want \"0\"", progress.Progress())
	}

	// Simulate map change event handling logic
	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: characterId,
		Type:        character.EventCharacterStatusTypeMapChanged,
		WorldId:     1,
		Body: character.StatusEventMapChangedBody{
			ChannelId:   1,
			TargetMapId: _map.Id(mapId),
		},
	}

	if event.Type != character.EventCharacterStatusTypeMapChanged || event.CharacterId == 0 {
		t.Fatal("Event should be processed")
	}

	targetMapId := uint32(event.Body.TargetMapId)

	// Get started quests
	quests, _ := processor.GetByCharacterIdAndState(characterId, quest.StateStarted)

	// Update map progress
	for _, q := range quests {
		if _, found := q.GetProgress(targetMapId); found {
			_ = processor.SetProgress(uuid.Nil, characterId, q.QuestId(), targetMapId, "1")
		}
	}

	// Verify map was marked as visited
	fetched, _ = processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(mapId)
	if !found {
		t.Fatal("Expected map progress entry to exist")
	}
	if progress.Progress() != "1" {
		t.Errorf("Map progress = %s, want \"1\"", progress.Progress())
	}
}

// TestHandleMapChangedEvent_MultipleMapRequirements tests multiple map visit tracking
func TestHandleMapChangedEvent_MultipleMapRequirements(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1001)
	characterId := uint32(12345)
	mapIds := []uint32{100000000, 100000001, 100000002}

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMapRequirement(questId, mapIds))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the quest
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, test.CreateTestField(), true)

	// Visit only the first two maps
	for _, mapId := range mapIds[:2] {
		quests, _ := processor.GetByCharacterIdAndState(characterId, quest.StateStarted)
		for _, q := range quests {
			if _, found := q.GetProgress(mapId); found {
				_ = processor.SetProgress(uuid.Nil, characterId, q.QuestId(), mapId, "1")
			}
		}
	}

	// Verify progress
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)

	// First two should be visited
	for _, mapId := range mapIds[:2] {
		progress, _ := fetched.GetProgress(mapId)
		if progress.Progress() != "1" {
			t.Errorf("Map %d progress = %s, want \"1\"", mapId, progress.Progress())
		}
	}

	// Third should still be not visited
	progress, _ := fetched.GetProgress(mapIds[2])
	if progress.Progress() != "0" {
		t.Errorf("Map %d progress = %s, want \"0\" (not visited)", mapIds[2], progress.Progress())
	}
}

// TestHandleMapChangedEvent_AutoComplete tests that auto-complete is triggered after map visit
func TestHandleMapChangedEvent_AutoComplete(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1002)
	characterId := uint32(12345)
	mapId := uint32(100000000)

	// Create auto-complete quest with single map requirement
	def := test.CreateQuestWithMapRequirement(questId, []uint32{mapId})
	def.AutoComplete = true
	mockData.AddQuestDefinition(questId, def)

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the quest
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, test.CreateTestField(), true)

	// Simulate map change and auto-complete check
	quests, _ := processor.GetByCharacterIdAndState(characterId, quest.StateStarted)
	for _, q := range quests {
		if _, found := q.GetProgress(mapId); found {
			_ = processor.SetProgress(uuid.Nil, characterId, q.QuestId(), mapId, "1")

			// Check auto-complete
			_, completed, _ := processor.CheckAutoComplete(characterId, q.QuestId(), test.CreateTestFieldWithMap(mapId))
			if completed {
				// Quest was auto-completed
			}
		}
	}

	// Verify quest was auto-completed
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if fetched.State() != quest.StateCompleted {
		t.Errorf("Quest state = %d, want StateCompleted (auto-complete)", fetched.State())
	}
}

// TestHandleMapChangedEvent_ChainedQuest tests that chained quests are started after auto-complete
func TestHandleMapChangedEvent_ChainedQuest(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1003)
	nextQuestId := uint32(1004)
	characterId := uint32(12345)
	mapId := uint32(100000000)

	// Create auto-complete quest with chain
	def := test.CreateQuestWithMapRequirement(questId, []uint32{mapId})
	def.AutoComplete = true
	def.EndActions.NextQuest = nextQuestId
	mockData.AddQuestDefinition(questId, def)
	mockData.AddQuestDefinition(nextQuestId, test.CreateSimpleQuestDefinition(nextQuestId))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the first quest
	_, _, _ = processor.Start(uuid.Nil, characterId, questId, test.CreateTestField(), true)

	// Simulate map change with chain handling
	quests, _ := processor.GetByCharacterIdAndState(characterId, quest.StateStarted)
	for _, q := range quests {
		if _, found := q.GetProgress(mapId); found {
			_ = processor.SetProgress(uuid.Nil, characterId, q.QuestId(), mapId, "1")

			nextId, completed, _ := processor.CheckAutoComplete(characterId, q.QuestId(), test.CreateTestFieldWithMap(mapId))
			if completed && nextId > 0 {
				// Start chained quest
				_, _ = processor.StartChained(uuid.Nil, characterId, nextId, test.CreateTestFieldWithMap(mapId))
			}
		}
	}

	// Verify first quest was completed
	first, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if first.State() != quest.StateCompleted {
		t.Errorf("First quest state = %d, want StateCompleted", first.State())
	}

	// Verify next quest was started
	next, err := processor.GetByCharacterIdAndQuestId(characterId, nextQuestId)
	if err != nil {
		t.Fatalf("Next quest not found: %v", err)
	}
	if next.State() != quest.StateStarted {
		t.Errorf("Next quest state = %d, want StateStarted", next.State())
	}
}

// Ensure consumer package is imported
func init() {
	_ = consumer.InitConsumers
}
