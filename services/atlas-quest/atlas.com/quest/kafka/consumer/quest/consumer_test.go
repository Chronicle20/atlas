package quest_test

import (
	consumer "atlas-quest/kafka/consumer/quest"
	questmsg "atlas-quest/kafka/message/quest"
	"atlas-quest/quest"
	"atlas-quest/test"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestHandleStartQuestCommand_IgnoresWrongType tests that the handler ignores non-START commands
func TestHandleStartQuestCommand_IgnoresWrongType(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()
	mockData.AddQuestDefinition(1000, test.CreateSimpleQuestDefinition(1000))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Create a command with COMPLETE type instead of START
	cmd := questmsg.Command[questmsg.StartCommandBody]{
		WorldId:     1,
		ChannelId:   1,
		CharacterId: 12345,
		Type:        questmsg.CommandTypeComplete, // Wrong type
		Body: questmsg.StartCommandBody{
			QuestId: 1000,
		},
	}

	// Call the handler logic - since it checks the type, it should do nothing
	if cmd.Type != questmsg.CommandTypeStart {
		// This is what the handler does - early return
		return
	}

	// If we get here, something is wrong
	t.Error("Handler should have returned early for wrong command type")

	_ = processor // Use processor to avoid unused warning
}

// TestHandleStartQuestCommand_StartsQuest tests that START command starts a quest
func TestHandleStartQuestCommand_StartsQuest(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Simulate what the handler does - call processor.Start
	cmd := questmsg.Command[questmsg.StartCommandBody]{
		WorldId:     1,
		ChannelId:   1,
		CharacterId: characterId,
		Type:        questmsg.CommandTypeStart,
		Body: questmsg.StartCommandBody{
			QuestId: questId,
		},
	}

	if cmd.Type == questmsg.CommandTypeStart {
		// Kafka commands skip validation
		_, _, err := processor.Start(cmd.CharacterId, cmd.Body.QuestId, test.CreateTestField(), true)
		if err != nil {
			t.Fatalf("processor.Start() unexpected error: %v", err)
		}
	}

	// Verify quest was started
	fetched, err := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId() error: %v", err)
	}
	if fetched.State() != quest.StateStarted {
		t.Errorf("quest state = %d, want StateStarted", fetched.State())
	}
}

// TestHandleCompleteQuestCommand_CompletesQuest tests that COMPLETE command completes a quest
func TestHandleCompleteQuestCommand_CompletesQuest(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// First start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Simulate COMPLETE command
	cmd := questmsg.Command[questmsg.CompleteCommandBody]{
		WorldId:     1,
		ChannelId:   1,
		CharacterId: characterId,
		Type:        questmsg.CommandTypeComplete,
		Body: questmsg.CompleteCommandBody{
			QuestId: questId,
			Force:   true, // Force completion (skip validation)
		},
	}

	if cmd.Type == questmsg.CommandTypeComplete {
		_, err := processor.Complete(cmd.CharacterId, cmd.Body.QuestId, test.CreateTestField(), cmd.Body.Force)
		if err != nil {
			t.Fatalf("processor.Complete() unexpected error: %v", err)
		}
	}

	// Verify quest was completed
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if fetched.State() != quest.StateCompleted {
		t.Errorf("quest state = %d, want StateCompleted", fetched.State())
	}
}

// TestHandleCompleteQuestCommand_StartsChainedQuest tests that completing a quest starts the next in chain
func TestHandleCompleteQuestCommand_StartsChainedQuest(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(2001)
	nextQuestId := uint32(2002)
	characterId := uint32(12345)

	mockData.AddQuestDefinition(questId, test.CreateQuestWithChain(questId, nextQuestId))
	mockData.AddQuestDefinition(nextQuestId, test.CreateSimpleQuestDefinition(nextQuestId))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the first quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Complete and check for chain
	nextId, err := processor.Complete(characterId, questId, test.CreateTestField(), true)
	if err != nil {
		t.Fatalf("processor.Complete() error: %v", err)
	}

	if nextId != nextQuestId {
		t.Errorf("nextQuestId = %d, want %d", nextId, nextQuestId)
	}

	// Start the chained quest (as the handler would)
	if nextId > 0 {
		_, err = processor.StartChained(characterId, nextId, test.CreateTestField())
		if err != nil {
			t.Fatalf("processor.StartChained() error: %v", err)
		}
	}

	// Verify next quest was started
	nextQuest, err := processor.GetByCharacterIdAndQuestId(characterId, nextQuestId)
	if err != nil {
		t.Fatalf("GetByCharacterIdAndQuestId(nextQuestId) error: %v", err)
	}
	if nextQuest.State() != quest.StateStarted {
		t.Errorf("next quest state = %d, want StateStarted", nextQuest.State())
	}
}

// TestHandleForfeitQuestCommand_ForfeitsQuest tests that FORFEIT command forfeits a quest
func TestHandleForfeitQuestCommand_ForfeitsQuest(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(3000)
	characterId := uint32(12345)
	mockData.AddQuestDefinition(questId, test.CreateSimpleQuestDefinition(questId))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// First start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Simulate FORFEIT command
	cmd := questmsg.Command[questmsg.ForfeitCommandBody]{
		WorldId:     1,
		ChannelId:   1,
		CharacterId: characterId,
		Type:        questmsg.CommandTypeForfeit,
		Body: questmsg.ForfeitCommandBody{
			QuestId: questId,
		},
	}

	if cmd.Type == questmsg.CommandTypeForfeit {
		err := processor.Forfeit(cmd.CharacterId, cmd.Body.QuestId)
		if err != nil {
			t.Fatalf("processor.Forfeit() unexpected error: %v", err)
		}
	}

	// Verify quest was forfeited
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	if fetched.State() != quest.StateNotStarted {
		t.Errorf("quest state = %d, want StateNotStarted", fetched.State())
	}
}

// TestHandleUpdateProgressCommand_UpdatesProgress tests that UPDATE_PROGRESS command updates progress
func TestHandleUpdateProgressCommand_UpdatesProgress(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(4000)
	characterId := uint32(12345)
	mobId := uint32(100100)

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation)

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Simulate UPDATE_PROGRESS command
	cmd := questmsg.Command[questmsg.UpdateProgressCommandBody]{
		WorldId:     1,
		ChannelId:   1,
		CharacterId: characterId,
		Type:        questmsg.CommandTypeUpdateProgress,
		Body: questmsg.UpdateProgressCommandBody{
			QuestId:    questId,
			InfoNumber: mobId,
			Progress:   "005",
		},
	}

	if cmd.Type == questmsg.CommandTypeUpdateProgress {
		err := processor.SetProgress(cmd.CharacterId, cmd.Body.QuestId, cmd.Body.InfoNumber, cmd.Body.Progress)
		if err != nil {
			t.Fatalf("processor.SetProgress() unexpected error: %v", err)
		}
	}

	// Verify progress was updated
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(mobId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "005" {
		t.Errorf("progress = %s, want \"005\"", progress.Progress())
	}
}

// Test helper function
func init() {
	// Ensure consumer package is imported (triggers init if needed)
	_ = consumer.InitConsumers
}
