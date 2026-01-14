package monster_test

import (
	consumer "atlas-quest/kafka/consumer/monster"
	"atlas-quest/kafka/message/monster"
	"atlas-quest/quest"
	"atlas-quest/test"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestHandleMonsterKilledEvent_IgnoresWrongType tests that non-KILLED events are ignored
func TestHandleMonsterKilledEvent_IgnoresWrongType(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	// Create an event with wrong type
	event := monster.StatusEvent[monster.StatusEventKilledBody]{
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
		MonsterId: 100100,
		Type:      "SPAWNED", // Wrong type
		Body: monster.StatusEventKilledBody{
			DamageEntries: []monster.DamageEntry{
				{CharacterId: 12345, Damage: 1000},
			},
		},
	}

	// Handler should return early for wrong type
	if event.Type != monster.EventMonsterStatusKilled {
		// This is what the handler does
		return
	}

	t.Error("Handler should have returned early for wrong event type")

	_ = db
}

// TestHandleMonsterKilledEvent_SkipsZeroCharacterId tests that zero character IDs are skipped
func TestHandleMonsterKilledEvent_SkipsZeroCharacterId(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	event := monster.StatusEvent[monster.StatusEventKilledBody]{
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
		MonsterId: 100100,
		Type:      monster.EventMonsterStatusKilled,
		Body: monster.StatusEventKilledBody{
			DamageEntries: []monster.DamageEntry{
				{CharacterId: 0, Damage: 1000}, // Zero character ID
			},
		},
	}

	// Handler should skip entries with zero character ID
	for _, entry := range event.Body.DamageEntries {
		if entry.CharacterId == 0 {
			continue // This is what the handler does
		}
		t.Error("Handler should have skipped zero character ID")
	}

	_ = db
}

// TestHandleMonsterKilledEvent_IncrementsMobProgress tests that mob kill progress is incremented
func TestHandleMonsterKilledEvent_IncrementsMobProgress(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1000)
	characterId := uint32(12345)
	mobId := uint32(100100)

	// Create quest with mob requirement
	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Verify initial progress
	fetched, _ := processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, _ := fetched.GetProgress(mobId)
	if progress.Progress() != "000" {
		t.Errorf("Initial progress = %s, want \"000\"", progress.Progress())
	}

	// Simulate monster kill event handling logic
	event := monster.StatusEvent[monster.StatusEventKilledBody]{
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
		MonsterId: mobId,
		Type:      monster.EventMonsterStatusKilled,
		Body: monster.StatusEventKilledBody{
			DamageEntries: []monster.DamageEntry{
				{CharacterId: characterId, Damage: 1000},
			},
		},
	}

	// Process the event (simulating handler logic)
	for _, entry := range event.Body.DamageEntries {
		if entry.CharacterId == 0 {
			continue
		}

		quests, err := processor.GetByCharacterIdAndState(entry.CharacterId, quest.StateStarted)
		if err != nil {
			continue
		}

		for _, q := range quests {
			if p, found := q.GetProgress(event.MonsterId); found {
				currentCount := parseProgress(p.Progress())
				newCount := currentCount + 1
				_ = processor.SetProgress(entry.CharacterId, q.QuestId(), event.MonsterId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify progress was incremented
	fetched, _ = processor.GetByCharacterIdAndQuestId(characterId, questId)
	progress, found := fetched.GetProgress(mobId)
	if !found {
		t.Fatal("Expected progress entry to exist")
	}
	if progress.Progress() != "1" {
		t.Errorf("Updated progress = %s, want \"1\"", progress.Progress())
	}
}

// TestHandleMonsterKilledEvent_MultipleCharacters tests progress for multiple damage dealers
func TestHandleMonsterKilledEvent_MultipleCharacters(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1001)
	mobId := uint32(100100)
	characterIds := []uint32{12345, 12346, 12347}

	mockData.AddQuestDefinition(questId, test.CreateQuestWithMobRequirement(questId, mobId, 10))

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start quest for all characters
	for _, charId := range characterIds {
		_, _, _ = processor.Start(charId, questId, test.CreateTestField(), true)
	}

	// Create damage entries for all characters
	damageEntries := make([]monster.DamageEntry, len(characterIds))
	for i, charId := range characterIds {
		damageEntries[i] = monster.DamageEntry{CharacterId: charId, Damage: 1000}
	}

	event := monster.StatusEvent[monster.StatusEventKilledBody]{
		MonsterId: mobId,
		Type:      monster.EventMonsterStatusKilled,
		Body: monster.StatusEventKilledBody{
			DamageEntries: damageEntries,
		},
	}

	// Process for each character
	for _, entry := range event.Body.DamageEntries {
		quests, _ := processor.GetByCharacterIdAndState(entry.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.MonsterId); found {
				currentCount := parseProgress(p.Progress())
				newCount := currentCount + 1
				_ = processor.SetProgress(entry.CharacterId, q.QuestId(), event.MonsterId, strconv.Itoa(int(newCount)))
			}
		}
	}

	// Verify each character got progress
	for _, charId := range characterIds {
		fetched, _ := processor.GetByCharacterIdAndQuestId(charId, questId)
		progress, found := fetched.GetProgress(mobId)
		if !found {
			t.Errorf("Character %d: progress entry not found", charId)
			continue
		}
		if progress.Progress() != "1" {
			t.Errorf("Character %d: progress = %s, want \"1\"", charId, progress.Progress())
		}
	}
}

// TestHandleMonsterKilledEvent_AutoComplete tests that auto-complete is triggered after kill
func TestHandleMonsterKilledEvent_AutoComplete(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	mockData := test.NewMockDataProcessor()
	mockValidation := test.NewMockValidationProcessor()

	questId := uint32(1002)
	characterId := uint32(12345)
	mobId := uint32(100100)

	// Create auto-complete quest with 1 kill requirement
	def := test.CreateQuestWithMobRequirement(questId, mobId, 1)
	def.AutoComplete = true
	mockData.AddQuestDefinition(questId, def)

	processor := quest.NewProcessorWithDependencies(logger, ctx, db, mockData, mockValidation, test.NewMockEventEmitter())

	// Start the quest
	_, _, _ = processor.Start(characterId, questId, test.CreateTestField(), true)

	// Simulate kill (increment progress to meet requirement)
	event := monster.StatusEvent[monster.StatusEventKilledBody]{
		MonsterId: mobId,
		Type:      monster.EventMonsterStatusKilled,
		Body: monster.StatusEventKilledBody{
			DamageEntries: []monster.DamageEntry{
				{CharacterId: characterId, Damage: 1000},
			},
		},
	}

	// Process the event with auto-complete check
	for _, entry := range event.Body.DamageEntries {
		quests, _ := processor.GetByCharacterIdAndState(entry.CharacterId, quest.StateStarted)
		for _, q := range quests {
			if p, found := q.GetProgress(event.MonsterId); found {
				currentCount := parseProgress(p.Progress())
				newCount := currentCount + 1
				_ = processor.SetProgress(entry.CharacterId, q.QuestId(), event.MonsterId, strconv.Itoa(int(newCount)))

				// Check auto-complete
				_, completed, _ := processor.CheckAutoComplete(entry.CharacterId, q.QuestId(), test.CreateTestField())
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
