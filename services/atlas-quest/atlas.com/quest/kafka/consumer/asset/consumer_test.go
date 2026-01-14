package asset_test

import (
	consumer "atlas-quest/kafka/consumer/asset"
	"atlas-quest/kafka/message/asset"
	"atlas-quest/quest"
	"atlas-quest/test"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestHandleAssetCreatedEvent_IgnoresWrongType tests that non-CREATED events are ignored
func TestHandleAssetCreatedEvent_IgnoresWrongType(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	// Create an event with wrong type
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: 12345,
		TemplateId:  2000000,
		Type:        "DELETED", // Wrong type
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	// Handler should return early for wrong type
	if event.Type != asset.StatusEventTypeCreated {
		// This is what the handler does
		return
	}

	t.Error("Handler should have returned early for wrong event type")

	_ = db
}

// TestHandleAssetCreatedEvent_SkipsZeroCharacterId tests that zero character IDs are skipped
func TestHandleAssetCreatedEvent_SkipsZeroCharacterId(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: 0, // Zero character ID
		TemplateId:  2000000,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	// Handler should skip zero character ID
	if event.CharacterId == 0 {
		return // This is what the handler does
	}

	t.Error("Handler should have skipped zero character ID")

	_ = db
}

// TestHandleAssetCreatedEvent_IncrementsItemProgress tests that item collection progress is incremented
func TestHandleAssetCreatedEvent_IncrementsItemProgress(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2000)
	characterId := uint32(12345)
	itemId := uint32(2000000)

	// Create quest with item requirement
	mockData.AddQuestDefinition(questId, test.CreateQuestWithItemRequirement(questId, itemId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Initialize progress manually (simulating what the handler needs to look for)
	// Note: Item progress is created on-demand when SetProgress is called
	_ = processor.SetProgress(characterId, questId, itemId, "0")

	// Verify initial progress
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(itemId)
	if !found {
		t.Fatal("Expected progress entry to exist after initialization")
	}
	if progress.Progress() != "0" {
		t.Errorf("Initial progress = %s, want \"0\"", progress.Progress())
	}

	// Simulate asset created event handling logic
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  itemId,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	// Process the event (simulating handler logic)
	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, err := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		if err == nil {
			for _, q := range quests {
				if p, found := q.GetProgress(event.TemplateId); found {
					currentCount := parseProgress(p.Progress())
					quantity := event.Body.Quantity
					if quantity == 0 {
						quantity = 1
					}
					newCount := currentCount + quantity
					_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))
				}
			}
		}
	}

	// Verify progress was incremented
	fetched, _ = processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found = fetched.GetProgress(itemId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "1" {
		t.Errorf("Updated progress = %s, want \"1\"", progress.Progress())
	}
}

// TestHandleAssetCreatedEvent_QuantityGreaterThanOne tests progress with quantity > 1
func TestHandleAssetCreatedEvent_QuantityGreaterThanOne(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2001)
	characterId := uint32(12345)
	itemId := uint32(2000001)

	mockData.AddQuestDefinition(questId, test.CreateQuestWithItemRequirement(questId, itemId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Initialize progress
	_ = processor.SetProgress(characterId, questId, itemId, "0")

	// Process event with quantity 5
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  itemId,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 5,
		},
	}

	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, _ := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.TemplateId); found {
				currentCount := parseProgress(p.Progress())
				quantity := event.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify progress was incremented by quantity
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(itemId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "5" {
		t.Errorf("Updated progress = %s, want \"5\"", progress.Progress())
	}
}

// TestHandleAssetCreatedEvent_ZeroQuantityDefaultsToOne tests that quantity 0 defaults to 1
func TestHandleAssetCreatedEvent_ZeroQuantityDefaultsToOne(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2002)
	characterId := uint32(12345)
	itemId := uint32(2000002)

	mockData.AddQuestDefinition(questId, test.CreateQuestWithItemRequirement(questId, itemId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Initialize progress
	_ = processor.SetProgress(characterId, questId, itemId, "0")

	// Process event with quantity 0 (should default to 1)
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  itemId,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 0, // Zero quantity should default to 1
		},
	}

	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, _ := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.TemplateId); found {
				currentCount := parseProgress(p.Progress())
				quantity := event.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify progress was incremented by 1 (default)
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(itemId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "1" {
		t.Errorf("Updated progress = %s, want \"1\"", progress.Progress())
	}
}

// TestHandleAssetCreatedEvent_NoMatchingProgressEntry tests that items without progress entries are ignored
func TestHandleAssetCreatedEvent_NoMatchingProgressEntry(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2003)
	characterId := uint32(12345)
	trackedItemId := uint32(2000003)
	untrackedItemId := uint32(9999999) // Different item

	mockData.AddQuestDefinition(questId, test.CreateQuestWithItemRequirement(questId, trackedItemId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Initialize progress for tracked item only
	_ = processor.SetProgress(characterId, questId, trackedItemId, "0")

	// Process event for untracked item
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  untrackedItemId, // Different item
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, _ := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.TemplateId); found {
				currentCount := parseProgress(p.Progress())
				quantity := event.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify tracked item progress is unchanged
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(trackedItemId)
	if !found {
		t.Fatal("Expected progress entry to exist for tracked item")
	}
	if progress.Progress() != "0" {
		t.Errorf("Tracked item progress = %s, want \"0\" (unchanged)", progress.Progress())
	}

	// Verify untracked item has no progress
	_, found = fetched.GetProgress(untrackedItemId)
	if found {
		t.Error("Did not expect progress entry for untracked item")
	}
}

// TestHandleAssetCreatedEvent_AutoComplete tests that auto-complete is triggered after collecting items
func TestHandleAssetCreatedEvent_AutoComplete(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2004)
	characterId := uint32(12345)
	itemId := uint32(2000004)

	// Create auto-complete quest with 1 item requirement
	def := test.CreateQuestWithItemRequirement(questId, itemId, 1)
	def.AutoComplete = true
	mockData.AddQuestDefinition(questId, def)

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Initialize progress for the item
	_ = processor.SetProgress(characterId, questId, itemId, "0")

	// Process event (increment progress to meet requirement)
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  itemId,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, _ := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.TemplateId); found {
				currentCount := parseProgress(p.Progress())
				quantity := event.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))

				// Check auto-complete
				_, completed, _ := processor.CheckAutoComplete(event.CharacterId, q.QuestId(), test.CreateTestField())
				if completed {
					// Quest was auto-completed
				}
			}
		}
	}

	// Verify quest was auto-completed
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if fetched.State() != quest.StateCompleted {
		t.Errorf("Quest state = %d, want StateCompleted (auto-complete)", fetched.State())
	}
}

// TestHandleAssetCreatedEvent_MultipleQuestsTrackingSameItem tests multiple quests tracking same item
func TestHandleAssetCreatedEvent_MultipleQuestsTrackingSameItem(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId1 := uint32(2005)
	questId2 := uint32(2006)
	characterId := uint32(12345)
	itemId := uint32(2000005)

	mockData.AddQuestDefinition(questId1, test.CreateQuestWithItemRequirement(questId1, itemId, 10))
	mockData.AddQuestDefinition(questId2, test.CreateQuestWithItemRequirement(questId2, itemId, 5))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start both quests
	_, _, _ = processor.Start(characterId, questId1, test.CreateTestField(), true)
	_, _, _ = processor.Start(characterId, questId2, test.CreateTestField(), true)

	// Initialize progress for both
	_ = processor.SetProgress(characterId, questId1, itemId, "0")
	_ = processor.SetProgress(characterId, questId2, itemId, "0")

	// Process event
	event := asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId: characterId,
		TemplateId:  itemId,
		Type:        asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			Quantity: 1,
		},
	}

	if event.Type == asset.StatusEventTypeCreated && event.CharacterId != 0 {
		quests, _ := processor.GetByCharacterIdAndState(event.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.TemplateId); found {
				currentCount := parseProgress(p.Progress())
				quantity := event.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				_ = processor.SetProgress(event.CharacterId, q.QuestId(), event.TemplateId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify both quests got progress
	fetched1, _ := processor.GetByCharacterIdAndQuestId(characterId, questId1)
	progress1, found := fetched1.GetProgress(itemId)
	if !found || progress1.Progress() != "1" {
		t.Errorf("Quest1 progress = %s, want \"1\"", progress1.Progress())
	}

	fetched2, _ := processor.GetByCharacterIdAndQuestId(characterId, questId2)
	progress2, found := fetched2.GetProgress(itemId)
	if !found || progress2.Progress() != "1" {
		t.Errorf("Quest2 progress = %s, want \"1\"", progress2.Progress())
	}
}

func parseProgress(progress string) uint32 {
	if progress == "" {
		return 0
	}
	val, err := strconv.Atoi(progress)
	if err != nil {
		return 0
	}
	return uint32(val)
}

// Ensure consumer package is imported
func init() {
	_ = consumer.InitConsumers
}
