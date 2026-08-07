package skill_test

import (
	"atlas-skills/kafka/message"
	skillmsg "atlas-skills/kafka/message/skill"
	"atlas-skills/skill"
	"atlas-skills/test"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func setupCooldownRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	skill.InitRegistry(client)
}

func setupProcessor(t *testing.T) (skill.Processor, func()) {
	setupCooldownRegistry(t)
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

	skills, err := processor.ByCharacterIdProvider(12345, model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills.Items) != 0 {
		t.Errorf("len(skills.Items) = %d, want 0", len(skills.Items))
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

	skills, err := processor.ByCharacterIdProvider(characterId, model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills.Items) != 2 {
		t.Errorf("len(skills.Items) = %d, want 2", len(skills.Items))
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
	skills, _ := processor.ByCharacterIdProvider(characterId, model.Page{Number: 1, Size: 250})()
	if len(skills.Items) != 2 {
		t.Fatalf("Expected 2 skills before delete, got %d", len(skills.Items))
	}

	// Delete all skills for character
	err := processor.Delete(characterId)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	// Verify skills are deleted
	skills, _ = processor.ByCharacterIdProvider(characterId, model.Page{Number: 1, Size: 250})()
	if len(skills.Items) != 0 {
		t.Errorf("len(skills.Items) = %d, want 0 after delete", len(skills.Items))
	}
}

func TestTenantIsolation(t *testing.T) {
	setupCooldownRegistry(t)
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
	skills, err := processor.ByCharacterIdProvider(characterId, model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(skills.Items) != len(skillIds) {
		t.Errorf("len(skills.Items) = %d, want %d", len(skills.Items), len(skillIds))
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
	skills1, _ := processor.ByCharacterIdProvider(char1, model.Page{Number: 1, Size: 250})()
	if len(skills1.Items) != 2 {
		t.Errorf("Character 1 skills = %d, want 2", len(skills1.Items))
	}

	// Verify character 2 has 1 skill
	skills2, _ := processor.ByCharacterIdProvider(char2, model.Page{Number: 1, Size: 250})()
	if len(skills2.Items) != 1 {
		t.Errorf("Character 2 skills = %d, want 1", len(skills2.Items))
	}
}

// TestDeleteForSagaCompensation_Existing covers the happy path: an existing
// skill row is deleted and a saga-correlated DELETED event is buffered
// (plan Phase 5). Uses the buffer-based variant to avoid Kafka dependency.
func TestDeleteForSagaCompensation_Existing(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	transactionId := uuid.New()
	worldId := world.Id(0)
	characterId := uint32(4242)
	skillId := uint32(1001001)
	mb := message.NewBuffer()

	if _, err := processor.Create(mb)(transactionId, worldId, characterId, skillId, 1, 1, time.Now()); err != nil {
		t.Fatalf("setup: create skill: %v", err)
	}

	if err := processor.DeleteForSagaCompensation(mb)(transactionId, worldId, characterId, skillId); err != nil {
		t.Fatalf("DeleteForSagaCompensation returned error on existing row: %v", err)
	}

	// Row should now be gone — a subsequent ByIdProvider should return empty.
	s, _ := processor.ByIdProvider(characterId, skillId)()
	if s.Id() == skillId {
		t.Fatalf("skill %d still exists after delete", skillId)
	}
}

// TestDeleteForSagaCompensation_Missing covers the idempotency contract from
// plan Phase 5.9: an absent skill row must not error — the processor still
// buffers a synthetic DELETED event.
func TestDeleteForSagaCompensation_Missing(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	mb := message.NewBuffer()
	err := processor.DeleteForSagaCompensation(mb)(uuid.New(), world.Id(0), 9999 /* char */, 1001001 /* skill */)
	if err != nil {
		t.Fatalf("DeleteForSagaCompensation should be idempotent on missing row, got error: %v", err)
	}
}

func setupResetProcessor(t *testing.T) (skill.Processor, context.Context, func()) {
	t.Helper()
	setupCooldownRegistry(t)
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	processor := skill.NewProcessor(logger, ctx, db)
	return processor, ctx, func() { test.CleanupTestDB(db) }
}

func decodeCooldownExpiredEvents(t *testing.T, mb *message.Buffer) []skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody] {
	t.Helper()
	events := make([]skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody], 0)
	for _, m := range mb.GetAll()[skillmsg.EnvStatusEventTopic] {
		var e skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			t.Fatalf("failed to decode buffered event: %v", err)
		}
		events = append(events, e)
	}
	return events
}

func TestResetCooldowns_ClearsAllButExcepted(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 5121010, 2940); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if err := skill.GetRegistry().Apply(ctx, characterId, 1311006, 300); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if err := skill.GetRegistry().Apply(ctx, characterId, 5221006, 60); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	transactionId := uuid.New()
	worldId := world.Id(1)
	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(transactionId, worldId, characterId, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 2 {
		t.Fatalf("cleared = %v, want 2 entries", cleared)
	}
	for _, id := range cleared {
		if id == 5121010 {
			t.Fatalf("excepted skill 5121010 was cleared")
		}
	}

	// Registry: excepted survives, others are gone.
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown 5121010 missing from registry: %v", err)
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err == nil {
		t.Fatalf("cooldown 1311006 still in registry after reset")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5221006); err == nil {
		t.Fatalf("cooldown 5221006 still in registry after reset")
	}

	// Events: one COOLDOWN_EXPIRED per cleared skill with real ids.
	events := decodeCooldownExpiredEvents(t, mb)
	if len(events) != 2 {
		t.Fatalf("buffered %d events, want 2", len(events))
	}
	seen := map[uint32]bool{}
	for _, e := range events {
		if e.Type != skillmsg.StatusEventTypeCooldownExpired {
			t.Errorf("event type = %s, want COOLDOWN_EXPIRED", e.Type)
		}
		if e.TransactionId != transactionId {
			t.Errorf("event transactionId = %s, want %s", e.TransactionId, transactionId)
		}
		if e.WorldId != worldId {
			t.Errorf("event worldId = %d, want %d", e.WorldId, worldId)
		}
		if e.CharacterId != characterId {
			t.Errorf("event characterId = %d, want %d", e.CharacterId, characterId)
		}
		seen[e.SkillId] = true
	}
	if !seen[1311006] || !seen[5221006] {
		t.Errorf("events cover skills %v, want 1311006 and 5221006", seen)
	}
	if seen[5121010] {
		t.Errorf("event emitted for excepted skill 5121010")
	}
}

func TestResetCooldowns_NoActiveCooldowns_NoOp(t *testing.T) {
	processor, _, cleanup := setupResetProcessor(t)
	defer cleanup()

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), 100, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("cleared = %v, want empty", cleared)
	}
	if len(decodeCooldownExpiredEvents(t, mb)) != 0 {
		t.Fatalf("events buffered for no-op reset")
	}
}

func TestResetCooldowns_AllExcepted_NoOp(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 5121010, 2940); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), characterId, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("cleared = %v, want empty", cleared)
	}
	if len(decodeCooldownExpiredEvents(t, mb)) != 0 {
		t.Fatalf("events buffered when every cooldown was excepted")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown removed: %v", err)
	}
}

func TestResetCooldowns_MultipleExceptions(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	for _, id := range []uint32{5121010, 1311006, 5221006} {
		if err := skill.GetRegistry().Apply(ctx, characterId, id, 60); err != nil {
			t.Fatalf("Apply() unexpected error: %v", err)
		}
	}

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), characterId, []uint32{5121010, 1311006})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 1 || cleared[0] != 5221006 {
		t.Fatalf("cleared = %v, want [5221006]", cleared)
	}
}
