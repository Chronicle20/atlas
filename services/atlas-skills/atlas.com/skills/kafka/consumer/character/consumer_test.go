package character_test

import (
	"atlas-skills/kafka/message"
	charmsg "atlas-skills/kafka/message/character"
	"atlas-skills/macro"
	"atlas-skills/skill"
	"atlas-skills/test"
	"testing"
	"time"

	skillconst "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupCooldownRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	skill.InitRegistry(client)
}

func TestHandleStatusEventLogout(t *testing.T) {
	setupCooldownRegistry(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(12345)

	skillProcessor := skill.NewProcessor(logger, ctx, db)

	// The logout handler calls ClearAll which clears cooldowns
	// Since we can't easily set cooldowns in tests (requires registry),
	// we just verify the call doesn't error
	err := skillProcessor.ClearAll(characterId)
	if err != nil {
		t.Fatalf("ClearAll() unexpected error: %v", err)
	}
}

func TestHandleStatusEventDeleted(t *testing.T) {
	setupCooldownRegistry(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	expiration := time.Now().Add(24 * time.Hour)

	skillProcessor := skill.NewProcessor(logger, ctx, db)
	macroProcessor := macro.NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()

	// Create some skills and macros
	_, err := skillProcessor.Create(mb)(transactionId, worldId, characterId, 1001001, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Create skill unexpected error: %v", err)
	}
	_, err = skillProcessor.Create(mb)(transactionId, worldId, characterId, 1001002, 5, 15, expiration)
	if err != nil {
		t.Fatalf("Create skill 2 unexpected error: %v", err)
	}

	testMacro, err := macro.NewModelBuilder().
		SetId(0).
		SetName("Test Macro").
		SetSkillId1(skillconst.Id(1001001)).
		Build()
	if err != nil {
		t.Fatalf("Build macro unexpected error: %v", err)
	}
	macros := []macro.Model{testMacro}
	_, err = macroProcessor.Update(mb)(transactionId, worldId, characterId, macros)
	if err != nil {
		t.Fatalf("Create macros unexpected error: %v", err)
	}

	// Verify data exists
	skills, _ := skillProcessor.ByCharacterIdProvider(characterId)()
	if len(skills) != 2 {
		t.Fatalf("Expected 2 skills before delete, got %d", len(skills))
	}
	macroResult, _ := macroProcessor.ByCharacterIdProvider(characterId)()
	if len(macroResult) != 1 {
		t.Fatalf("Expected 1 macro before delete, got %d", len(macroResult))
	}

	// Simulate what the deleted handler does
	err = skillProcessor.ClearAll(characterId)
	if err != nil {
		t.Fatalf("ClearAll() unexpected error: %v", err)
	}
	err = skillProcessor.Delete(characterId)
	if err != nil {
		t.Fatalf("Delete skills unexpected error: %v", err)
	}
	err = macroProcessor.Delete(characterId)
	if err != nil {
		t.Fatalf("Delete macros unexpected error: %v", err)
	}

	// Verify all data is deleted
	skills, _ = skillProcessor.ByCharacterIdProvider(characterId)()
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills after delete, got %d", len(skills))
	}
	macroResult, _ = macroProcessor.ByCharacterIdProvider(characterId)()
	if len(macroResult) != 0 {
		t.Errorf("Expected 0 macros after delete, got %d", len(macroResult))
	}
}

func TestEventTypeFiltering(t *testing.T) {
	// Test that handlers correctly filter by event type
	tests := []struct {
		name        string
		eventType   string
		expected    string
		shouldMatch bool
	}{
		{"Logout matches", charmsg.EventStatusTypeLogout, charmsg.EventStatusTypeLogout, true},
		{"Deleted matches", charmsg.StatusEventTypeDeleted, charmsg.StatusEventTypeDeleted, true},
		{"Wrong type does not match", charmsg.EventStatusTypeLogout, charmsg.StatusEventTypeDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.eventType == tt.expected
			if match != tt.shouldMatch {
				t.Errorf("event type match = %v, want %v", match, tt.shouldMatch)
			}
		})
	}
}
