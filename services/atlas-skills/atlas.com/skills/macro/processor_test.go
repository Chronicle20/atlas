package macro_test

import (
	"atlas-skills/kafka/message"
	"atlas-skills/macro"
	"atlas-skills/test"
	"testing"

	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// buildMacro is a test helper that builds a macro and fails the test on error
func buildMacro(t *testing.T, id uint32, name string, shout bool, skillId1, skillId2, skillId3 skill.Id) macro.Model {
	t.Helper()
	m, err := macro.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetShout(shout).
		SetSkillId1(skillId1).
		SetSkillId2(skillId2).
		SetSkillId3(skillId3).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	return m
}

func setupProcessor(t *testing.T) (macro.Processor, func()) {
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := macro.NewProcessor(logger, ctx, db)

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

	macros, err := processor.ByCharacterIdProvider(12345)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(macros) != 0 {
		t.Errorf("len(macros) = %d, want 0", len(macros))
	}
}

func TestByCharacterIdProvider_WithMacros(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	// Create some macros
	macros := []macro.Model{
		buildMacro(t, 0, "Attack", false, skill.Id(1001001), skill.Id(1001002), skill.Id(0)),
		buildMacro(t, 1, "Buff", true, skill.Id(2001001), skill.Id(2001002), skill.Id(2001003)),
	}

	_, err := processor.Update(mb)(characterId)(macros)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	result, err := processor.ByCharacterIdProvider(characterId)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(macros) = %d, want 2", len(result))
	}
}

func TestUpdate_Success(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	macros := []macro.Model{
		buildMacro(t, 0, "Attack Combo", true, skill.Id(1001001), skill.Id(1001002), skill.Id(1001003)),
	}

	result, err := processor.Update(mb)(characterId)(macros)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
	if result[0].Name() != "Attack Combo" {
		t.Errorf("result[0].Name() = %s, want \"Attack Combo\"", result[0].Name())
	}
	if result[0].Shout() != true {
		t.Errorf("result[0].Shout() = %v, want true", result[0].Shout())
	}

	// Verify message buffer has events
	events := mb.GetAll()
	if len(events) == 0 {
		t.Error("Expected events in message buffer")
	}
}

func TestUpdate_ReplacesExisting(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	// Create initial macros
	initialMacros := []macro.Model{
		buildMacro(t, 0, "Old Attack", false, skill.Id(1001001), 0, 0),
		buildMacro(t, 1, "Old Buff", true, skill.Id(2001001), 0, 0),
	}

	_, err := processor.Update(mb)(characterId)(initialMacros)
	if err != nil {
		t.Fatalf("Initial Update() unexpected error: %v", err)
	}

	// Verify initial macros
	result, _ := processor.ByCharacterIdProvider(characterId)()
	if len(result) != 2 {
		t.Fatalf("Expected 2 macros after initial update, got %d", len(result))
	}

	// Replace with new macros
	newMacros := []macro.Model{
		buildMacro(t, 0, "New Attack", true, skill.Id(3001001), 0, 0),
	}

	_, err = processor.Update(mb)(characterId)(newMacros)
	if err != nil {
		t.Fatalf("Second Update() unexpected error: %v", err)
	}

	// Verify old macros are replaced
	result, _ = processor.ByCharacterIdProvider(characterId)()
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1 after replacement", len(result))
	}
	if result[0].Name() != "New Attack" {
		t.Errorf("result[0].Name() = %s, want \"New Attack\"", result[0].Name())
	}
}

func TestUpdate_EmptyList(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	// Create initial macros
	initialMacros := []macro.Model{
		buildMacro(t, 0, "Attack", false, 0, 0, 0),
	}

	_, _ = processor.Update(mb)(characterId)(initialMacros)

	// Replace with empty list
	_, err := processor.Update(mb)(characterId)([]macro.Model{})
	if err != nil {
		t.Fatalf("Update() with empty list unexpected error: %v", err)
	}

	// Verify all macros are deleted
	result, _ := processor.ByCharacterIdProvider(characterId)()
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 after empty update", len(result))
	}
}

func TestDelete(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	// Create some macros
	macros := []macro.Model{
		buildMacro(t, 0, "Attack", false, 0, 0, 0),
		buildMacro(t, 1, "Buff", false, 0, 0, 0),
	}

	_, _ = processor.Update(mb)(characterId)(macros)

	// Verify macros exist
	result, _ := processor.ByCharacterIdProvider(characterId)()
	if len(result) != 2 {
		t.Fatalf("Expected 2 macros before delete, got %d", len(result))
	}

	// Delete all macros for character
	err := processor.Delete(characterId)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	// Verify macros are deleted
	result, _ = processor.ByCharacterIdProvider(characterId)()
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 after delete", len(result))
	}
}

func TestTenantIsolation(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	logger, _ := logtest.NewNullLogger()

	// Use different character IDs for different tenants to avoid PK collision
	// (The macro table has composite PK of character_id + id)
	char1 := uint32(12345)
	char2 := uint32(67890)

	// Create macro with tenant 1
	tenant1Id := uuid.New()
	ctx1 := test.CreateTestContextWithTenant(tenant1Id)
	processor1 := macro.NewProcessor(logger, ctx1, db)
	mb1 := message.NewBuffer()

	macros1 := []macro.Model{
		buildMacro(t, 0, "Tenant1 Macro", false, 0, 0, 0),
	}
	_, err := processor1.Update(mb1)(char1)(macros1)
	if err != nil {
		t.Fatalf("Tenant 1 Update() unexpected error: %v", err)
	}

	// Create macro with tenant 2
	tenant2Id := uuid.New()
	ctx2 := test.CreateTestContextWithTenant(tenant2Id)
	processor2 := macro.NewProcessor(logger, ctx2, db)
	mb2 := message.NewBuffer()

	macros2 := []macro.Model{
		buildMacro(t, 0, "Tenant2 Macro", false, 0, 0, 0),
	}
	_, err = processor2.Update(mb2)(char2)(macros2)
	if err != nil {
		t.Fatalf("Tenant 2 Update() unexpected error: %v", err)
	}

	// Verify tenant 1 only sees their character's macro
	result1, _ := processor1.ByCharacterIdProvider(char1)()
	if len(result1) != 1 {
		t.Errorf("Tenant 1 macros = %d, want 1", len(result1))
	}
	if result1[0].Name() != "Tenant1 Macro" {
		t.Errorf("Tenant 1 macro name = %s, want \"Tenant1 Macro\"", result1[0].Name())
	}

	// Verify tenant 1 cannot see tenant 2's character's macros
	result1Other, _ := processor1.ByCharacterIdProvider(char2)()
	if len(result1Other) != 0 {
		t.Errorf("Tenant 1 should not see tenant 2's macros, got %d", len(result1Other))
	}

	// Verify tenant 2 only sees their character's macro
	result2, _ := processor2.ByCharacterIdProvider(char2)()
	if len(result2) != 1 {
		t.Errorf("Tenant 2 macros = %d, want 1", len(result2))
	}
	if result2[0].Name() != "Tenant2 Macro" {
		t.Errorf("Tenant 2 macro name = %s, want \"Tenant2 Macro\"", result2[0].Name())
	}

	// Verify tenant 2 cannot see tenant 1's character's macros
	result2Other, _ := processor2.ByCharacterIdProvider(char1)()
	if len(result2Other) != 0 {
		t.Errorf("Tenant 2 should not see tenant 1's macros, got %d", len(result2Other))
	}
}

func TestDifferentCharacters(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	mb := message.NewBuffer()

	char1 := uint32(12345)
	char2 := uint32(67890)

	// Create macros for character 1
	macros1 := []macro.Model{
		buildMacro(t, 0, "Char1 Attack", false, 0, 0, 0),
		buildMacro(t, 1, "Char1 Buff", false, 0, 0, 0),
	}
	_, _ = processor.Update(mb)(char1)(macros1)

	// Create macros for character 2
	macros2 := []macro.Model{
		buildMacro(t, 0, "Char2 Attack", false, 0, 0, 0),
	}
	_, _ = processor.Update(mb)(char2)(macros2)

	// Verify character 1 has 2 macros
	result1, _ := processor.ByCharacterIdProvider(char1)()
	if len(result1) != 2 {
		t.Errorf("Character 1 macros = %d, want 2", len(result1))
	}

	// Verify character 2 has 1 macro
	result2, _ := processor.ByCharacterIdProvider(char2)()
	if len(result2) != 1 {
		t.Errorf("Character 2 macros = %d, want 1", len(result2))
	}
}

func TestMacroSkillIds(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	characterId := uint32(12345)
	mb := message.NewBuffer()

	macros := []macro.Model{
		buildMacro(t, 0, "Multi-Skill Macro", false, skill.Id(1001001), skill.Id(1001002), skill.Id(1001003)),
	}

	_, err := processor.Update(mb)(characterId)(macros)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	result, _ := processor.ByCharacterIdProvider(characterId)()
	if len(result) != 1 {
		t.Fatalf("Expected 1 macro, got %d", len(result))
	}

	if result[0].SkillId1() != skill.Id(1001001) {
		t.Errorf("SkillId1 = %d, want 1001001", result[0].SkillId1())
	}
	if result[0].SkillId2() != skill.Id(1001002) {
		t.Errorf("SkillId2 = %d, want 1001002", result[0].SkillId2())
	}
	if result[0].SkillId3() != skill.Id(1001003) {
		t.Errorf("SkillId3 = %d, want 1001003", result[0].SkillId3())
	}
}
