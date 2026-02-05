package skill_test

import (
	"atlas-skills/kafka/message"
	"atlas-skills/skill"
	"atlas-skills/test"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupProcessor(t *testing.T) (skill.Processor, func()) {
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := skill.NewProcessor(logger, ctx, db)

	cleanup := func() {
		test.CleanupTestDB(db)
	}

	return processor, cleanup
}

func TestNewProcessor(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	if processor == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

func TestByCharacterIdProvider_Empty(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	skills, err := processor.ByCharacterIdProvider(12345)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("len(skills) = %d, want 0", len(skills))
	}
}

func TestByCharacterIdProvider_WithSkills(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// Create some skills
	_, err := processor.Create(mb)(transactionId, worldId, characterId, 1001001, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	_, err = processor.Create(mb)(transactionId, worldId, characterId, 1001002, 5, 15, expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	skills, err := processor.ByCharacterIdProvider(characterId)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("len(skills) = %d, want 2", len(skills))
	}
}

func TestByIdProvider_NotFound(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	_, err := processor.ByIdProvider(12345, 999999)()
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

func TestByIdProvider_Found(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	skillId := uint32(1001001)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	_, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	s, err := processor.ByIdProvider(characterId, skillId)()
	if err != nil {
		t.Fatalf("ByIdProvider() unexpected error: %v", err)
	}
	if s.Id() != skillId {
		t.Errorf("s.Id() = %d, want %d", s.Id(), skillId)
	}
	if s.Level() != 10 {
		t.Errorf("s.Level() = %d, want 10", s.Level())
	}
	if s.MasterLevel() != 20 {
		t.Errorf("s.MasterLevel() = %d, want 20", s.MasterLevel())
	}
}

func TestCreate_Success(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	skillId := uint32(1001001)
	level := byte(10)
	masterLevel := byte(20)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	s, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, level, masterLevel, expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if s.Id() != skillId {
		t.Errorf("s.Id() = %d, want %d", s.Id(), skillId)
	}
	if s.Level() != level {
		t.Errorf("s.Level() = %d, want %d", s.Level(), level)
	}
	if s.MasterLevel() != masterLevel {
		t.Errorf("s.MasterLevel() = %d, want %d", s.MasterLevel(), masterLevel)
	}

	// Verify message buffer has events
	events := mb.GetAll()
	if len(events) == 0 {
		t.Error("Expected events in message buffer")
	}
}

func TestCreate_AlreadyExists(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	skillId := uint32(1001001)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// First creation should succeed
	_, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, 10, 20, expiration)
	if err != nil {
		t.Fatalf("First Create() unexpected error: %v", err)
	}

	// Second creation should fail
	_, err = processor.Create(mb)(transactionId, worldId, characterId, skillId, 15, 25, expiration)
	if err == nil {
		t.Error("Second Create() expected error for duplicate skill")
	}
}

func TestUpdate_Success(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	skillId := uint32(1001001)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// Create initial skill
	_, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	// Update the skill
	newExpiration := time.Now().Add(48 * time.Hour)
	s, err := processor.Update(mb)(transactionId, worldId, characterId, skillId, 15, 25, newExpiration)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if s.Level() != 15 {
		t.Errorf("s.Level() = %d, want 15", s.Level())
	}
	if s.MasterLevel() != 25 {
		t.Errorf("s.MasterLevel() = %d, want 25", s.MasterLevel())
	}
}

func TestUpdate_NotFound(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	skillId := uint32(999999)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	_, err := processor.Update(mb)(transactionId, worldId, characterId, skillId, 10, 20, expiration)
	if err == nil {
		t.Error("Update() expected error for non-existent skill")
	}
}

func TestDelete(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// Create some skills
	_, _ = processor.Create(mb)(transactionId, worldId, characterId, 1001001, 10, 20, expiration)
	_, _ = processor.Create(mb)(transactionId, worldId, characterId, 1001002, 5, 15, expiration)

	// Verify skills exist
	skills, _ := processor.ByCharacterIdProvider(characterId)()
	if len(skills) != 2 {
		t.Fatalf("Expected 2 skills before delete, got %d", len(skills))
	}

	// Delete all skills for character
	err := processor.Delete(characterId)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	// Verify skills are deleted
	skills, _ = processor.ByCharacterIdProvider(characterId)()
	if len(skills) != 0 {
		t.Errorf("len(skills) = %d, want 0 after delete", len(skills))
	}
}

func TestTenantIsolation(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()

	characterId := uint32(12345)
	skillId1 := uint32(1001001)
	skillId2 := uint32(1001002)
	expiration := time.Now().Add(24 * time.Hour)

	// Create skill with tenant 1
	transactionId := uuid.New()
	worldId := world.Id(0)
	tenant1Id := uuid.New()
	ctx1 := test.CreateTestContextWithTenant(tenant1Id)
	processor1 := skill.NewProcessor(logger, ctx1, db)
	mb1 := message.NewBuffer()
	_, err := processor1.Create(mb1)(transactionId, worldId, characterId, skillId1, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Tenant 1 Create() unexpected error: %v", err)
	}

	// Try to access from tenant 2
	tenant2Id := uuid.New()
	ctx2 := test.CreateTestContextWithTenant(tenant2Id)
	processor2 := skill.NewProcessor(logger, ctx2, db)

	// Tenant 2 should not see tenant 1's skill
	_, err = processor2.ByIdProvider(characterId, skillId1)()
	if err == nil {
		t.Error("Expected error when accessing skill from different tenant")
	}

	// Tenant 2 creates a different skill
	mb2 := message.NewBuffer()
	_, err = processor2.Create(mb2)(transactionId, worldId, characterId, skillId2, 5, 10, expiration)
	if err != nil {
		t.Fatalf("Tenant 2 Create() unexpected error: %v", err)
	}

	// Verify tenant 1 has their skill
	s1, err := processor1.ByIdProvider(characterId, skillId1)()
	if err != nil {
		t.Fatalf("Tenant 1 ByIdProvider() unexpected error: %v", err)
	}
	if s1.Level() != 10 {
		t.Errorf("Tenant 1 skill level = %d, want 10", s1.Level())
	}

	// Verify tenant 2 has their skill
	s2, err := processor2.ByIdProvider(characterId, skillId2)()
	if err != nil {
		t.Fatalf("Tenant 2 ByIdProvider() unexpected error: %v", err)
	}
	if s2.Level() != 5 {
		t.Errorf("Tenant 2 skill level = %d, want 5", s2.Level())
	}

	// Verify tenant 1 cannot see tenant 2's skill
	_, err = processor1.ByIdProvider(characterId, skillId2)()
	if err == nil {
		t.Error("Expected tenant 1 not to see tenant 2's skill")
	}
}

func TestClearAll(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)

	// ClearAll should not error even if no cooldowns exist
	err := processor.ClearAll(characterId)
	if err != nil {
		t.Fatalf("ClearAll() unexpected error: %v", err)
	}
}

func TestMultipleSkillsForCharacter(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(12345)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// Create multiple skills
	skillIds := []uint32{1001001, 1001002, 1001003, 2001001, 2001002}
	for i, skillId := range skillIds {
		level := byte(i + 1)
		masterLevel := byte((i + 1) * 5)
		_, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, level, masterLevel, expiration)
		if err != nil {
			t.Fatalf("Create() for skill %d unexpected error: %v", skillId, err)
		}
	}

	// Fetch all skills
	skills, err := processor.ByCharacterIdProvider(characterId)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills) != len(skillIds) {
		t.Errorf("len(skills) = %d, want %d", len(skills), len(skillIds))
	}
}

func TestDifferentCharacters(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	expiration := time.Now().Add(24 * time.Hour)
	mb := message.NewBuffer()

	// Create skills for different characters
	// Note: Each skill ID is globally unique in the DB, so different characters
	// must have different skill IDs (this models unique skill types per character)
	char1 := uint32(12345)
	char2 := uint32(67890)

	_, err := processor.Create(mb)(transactionId, worldId, char1, 1001001, 10, 20, expiration)
	if err != nil {
		t.Fatalf("Create for char1 skill 1: %v", err)
	}
	_, err = processor.Create(mb)(transactionId, worldId, char1, 1001002, 15, 25, expiration)
	if err != nil {
		t.Fatalf("Create for char1 skill 2: %v", err)
	}
	_, err = processor.Create(mb)(transactionId, worldId, char2, 2001001, 5, 10, expiration)
	if err != nil {
		t.Fatalf("Create for char2 skill: %v", err)
	}

	// Verify character 1 has 2 skills
	skills1, _ := processor.ByCharacterIdProvider(char1)()
	if len(skills1) != 2 {
		t.Errorf("Character 1 skills = %d, want 2", len(skills1))
	}

	// Verify character 2 has 1 skill
	skills2, _ := processor.ByCharacterIdProvider(char2)()
	if len(skills2) != 1 {
		t.Errorf("Character 2 skills = %d, want 1", len(skills2))
	}
}
