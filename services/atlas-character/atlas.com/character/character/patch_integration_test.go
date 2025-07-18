package character_test

import (
	"atlas-character/character"
	character2 "atlas-character/kafka/message/character"
	"atlas-character/kafka/message"
	"context"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestPatchCharacterIntegration(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test the Update method directly (without Kafka emission)
	updatePayload := character.RestModel{
		Id:        createdCharacter.Id(),
		Name:      "UpdatedName",
		Hair:      30100,
		Face:      20100,
		Gender:    1,
		SkinColor: 1,
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	// Verify the updates
	if updatedCharacter.Name() != "UpdatedName" {
		t.Errorf("Expected name 'UpdatedName', got '%s'", updatedCharacter.Name())
	}
	if updatedCharacter.Hair() != 30100 {
		t.Errorf("Expected hair 30100, got %d", updatedCharacter.Hair())
	}
	if updatedCharacter.Face() != 20100 {
		t.Errorf("Expected face 20100, got %d", updatedCharacter.Face())
	}
	if updatedCharacter.Gender() != 1 {
		t.Errorf("Expected gender 1, got %d", updatedCharacter.Gender())
	}
	if updatedCharacter.SkinColor() != 1 {
		t.Errorf("Expected skin color 1, got %d", updatedCharacter.SkinColor())
	}

	// Verify unchanged values remain the same
	if updatedCharacter.Level() != originalCharacter.Level() {
		t.Errorf("Level should remain unchanged: expected %d, got %d", originalCharacter.Level(), updatedCharacter.Level())
	}
	if updatedCharacter.Strength() != originalCharacter.Strength() {
		t.Errorf("Strength should remain unchanged: expected %d, got %d", originalCharacter.Strength(), updatedCharacter.Strength())
	}
	if updatedCharacter.AccountId() != originalCharacter.AccountId() {
		t.Errorf("AccountId should remain unchanged: expected %d, got %d", originalCharacter.AccountId(), updatedCharacter.AccountId())
	}
}

func TestPatchCharacterPartialUpdate(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("PartialUpdateTest").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test partial update with only name change
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Name: "NewPartialName",
		// Other fields are not set (zero values)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	// Verify only the name was updated
	if updatedCharacter.Name() != "NewPartialName" {
		t.Errorf("Expected name 'NewPartialName', got '%s'", updatedCharacter.Name())
	}

	// Verify all other fields remain unchanged
	if updatedCharacter.Hair() != originalCharacter.Hair() {
		t.Errorf("Hair should remain unchanged: expected %d, got %d", originalCharacter.Hair(), updatedCharacter.Hair())
	}
	if updatedCharacter.Face() != originalCharacter.Face() {
		t.Errorf("Face should remain unchanged: expected %d, got %d", originalCharacter.Face(), updatedCharacter.Face())
	}
	if updatedCharacter.Gender() != originalCharacter.Gender() {
		t.Errorf("Gender should remain unchanged: expected %d, got %d", originalCharacter.Gender(), updatedCharacter.Gender())
	}
	if updatedCharacter.SkinColor() != originalCharacter.SkinColor() {
		t.Errorf("SkinColor should remain unchanged: expected %d, got %d", originalCharacter.SkinColor(), updatedCharacter.SkinColor())
	}
}

func TestPatchCharacterWithInvalidName(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid name (too short)
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Name: "AB", // Too short
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid or duplicate name" {
		t.Fatalf("Expected 'invalid or duplicate name' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the name wasn't changed
	if unchangedCharacter.Name() != "TestCharacter" {
		t.Errorf("Character should not have been updated. Expected name 'TestCharacter', got '%s'", unchangedCharacter.Name())
	}
}

func TestPatchCharacterWithInvalidHair(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid hair ID
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Hair: 1000, // Invalid hair ID (should be 30000-35000)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid hair ID" {
		t.Fatalf("Expected 'invalid hair ID' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the hair wasn't changed
	if unchangedCharacter.Hair() != 30000 {
		t.Errorf("Character hair should not have been updated. Expected hair 30000, got %d", unchangedCharacter.Hair())
	}
}

func TestPatchCharacterNotFound(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Test with non-existent character ID
	updatePayload := character.RestModel{
		Id:   99999, // Non-existent character ID
		Name: "ShouldNotUpdate",
	}

	processor := character.NewProcessor(logger, tctx, db)
	
	// Test the update logic with message buffer
	transactionId := uuid.New()
	err := processor.Update(message.NewBuffer())(transactionId, 99999, updatePayload)
	if err == nil {
		t.Fatal("Expected error for non-existent character, got nil")
	}
	
	// Should return a record not found error
	if err.Error() != "record not found" {
		t.Fatalf("Expected 'record not found' error, got: %v", err)
	}
}

func TestPatchCharacterWithInvalidGender(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid gender
	updatePayload := character.RestModel{
		Id:     createdCharacter.Id(),
		Gender: 5, // Invalid gender (should be 0 or 1)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid gender value" {
		t.Fatalf("Expected 'invalid gender value' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the gender wasn't changed
	if unchangedCharacter.Gender() != 0 {
		t.Errorf("Character gender should not have been updated. Expected gender 0, got %d", unchangedCharacter.Gender())
	}
}

func TestPatchCharacterWithNoUpdates(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with no real updates (empty RestModel)
	updatePayload := character.RestModel{
		Id: createdCharacter.Id(),
		// All other fields are zero values, should trigger no updates
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was NOT changed in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify all fields remain unchanged
	if unchangedCharacter.Name() != originalCharacter.Name() {
		t.Errorf("Name should remain unchanged: expected '%s', got '%s'", originalCharacter.Name(), unchangedCharacter.Name())
	}
	if unchangedCharacter.Hair() != originalCharacter.Hair() {
		t.Errorf("Hair should remain unchanged: expected %d, got %d", originalCharacter.Hair(), unchangedCharacter.Hair())
	}
	if unchangedCharacter.Face() != originalCharacter.Face() {
		t.Errorf("Face should remain unchanged: expected %d, got %d", originalCharacter.Face(), unchangedCharacter.Face())
	}
	if unchangedCharacter.Gender() != originalCharacter.Gender() {
		t.Errorf("Gender should remain unchanged: expected %d, got %d", originalCharacter.Gender(), unchangedCharacter.Gender())
	}
	if unchangedCharacter.SkinColor() != originalCharacter.SkinColor() {
		t.Errorf("SkinColor should remain unchanged: expected %d, got %d", originalCharacter.SkinColor(), unchangedCharacter.SkinColor())
	}
}

func TestPatchCharacterWithDuplicateName(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create first character
	firstCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("ExistingName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	// Create second character
	secondCharacter := character.NewModelBuilder().
		SetAccountId(1001).
		SetWorldId(world.Id(0)).
		SetName("SecondName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	
	// Create both characters
	_, err := processor.Create(message.NewBuffer())(uuid.New(), firstCharacter)
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}
	
	createdSecondCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), secondCharacter)
	if err != nil {
		t.Fatalf("Failed to create second character: %v", err)
	}

	// Try to update second character to have same name as first
	updatePayload := character.RestModel{
		Id:   createdSecondCharacter.Id(),
		Name: "ExistingName", // This name already exists
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdSecondCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid or duplicate name" {
		t.Fatalf("Expected 'invalid or duplicate name' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdSecondCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the name wasn't changed
	if unchangedCharacter.Name() != "SecondName" {
		t.Errorf("Character should not have been updated. Expected name 'SecondName', got '%s'", unchangedCharacter.Name())
	}
}

func TestPatchCharacterWithInvalidFace(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid face ID
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Face: 1000, // Invalid face ID (should be 20000-25000)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid face ID" {
		t.Fatalf("Expected 'invalid face ID' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the face wasn't changed
	if unchangedCharacter.Face() != 20000 {
		t.Errorf("Character face should not have been updated. Expected face 20000, got %d", unchangedCharacter.Face())
	}
}

func TestPatchCharacterWithInvalidSkinColor(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid skin color
	updatePayload := character.RestModel{
		Id:        createdCharacter.Id(),
		SkinColor: 50, // Invalid skin color (should be 0-9)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid skin color value" {
		t.Fatalf("Expected 'invalid skin color value' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the skin color wasn't changed
	if unchangedCharacter.SkinColor() != 0 {
		t.Errorf("Character skin color should not have been updated. Expected skin color 0, got %d", unchangedCharacter.SkinColor())
	}
}

func TestPatchCharacterWithInvalidNameTooShort(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid name (too short)
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Name: "AB", // Too short (< 3 characters)
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid or duplicate name" {
		t.Fatalf("Expected 'invalid or duplicate name' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the name wasn't changed
	if unchangedCharacter.Name() != "TestCharacter" {
		t.Errorf("Character should not have been updated. Expected name 'TestCharacter', got '%s'", unchangedCharacter.Name())
	}
}

func TestPatchCharacterWithInvalidNameSpecialCharacters(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("TestCharacter").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Test with invalid name (contains special characters)
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Name: "%%%", // Contains only special characters not in allowed set
	}

	// Test the update logic with message buffer
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, createdCharacter.Id(), updatePayload)
	if err == nil || err.Error() != "invalid or duplicate name" {
		t.Fatalf("Expected 'invalid or duplicate name' error, got: %v", err)
	}

	// Verify the character was NOT updated in the database
	unchangedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	// Verify the name wasn't changed
	if unchangedCharacter.Name() != "TestCharacter" {
		t.Errorf("Character should not have been updated. Expected name 'TestCharacter', got '%s'", unchangedCharacter.Name())
	}
}

func TestNameChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test name change with message buffer
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Name: "UpdatedName",
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.Name() != "UpdatedName" {
		t.Errorf("Expected name 'UpdatedName', got '%s'", updatedCharacter.Name())
	}

	// Verify NAME_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected NAME_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one NAME_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 NAME_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("NAME_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("NAME_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a NAME_CHANGED event
	// The event should be a StatusEvent[StatusEventNameChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventNameChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeNameChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeNameChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldName != "OriginalName" {
		t.Errorf("Expected old name 'OriginalName', got '%s'", eventValue.Body.OldName)
	}

	if eventValue.Body.NewName != "UpdatedName" {
		t.Errorf("Expected new name 'UpdatedName', got '%s'", eventValue.Body.NewName)
	}
}

func TestHairChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test hair change with message buffer
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Hair: 30100,
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.Hair() != 30100 {
		t.Errorf("Expected hair 30100, got %d", updatedCharacter.Hair())
	}

	// Verify HAIR_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected HAIR_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one HAIR_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 HAIR_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("HAIR_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("HAIR_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a HAIR_CHANGED event
	// The event should be a StatusEvent[StatusEventHairChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventHairChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeHairChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeHairChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldHair != 30000 {
		t.Errorf("Expected old hair 30000, got %d", eventValue.Body.OldHair)
	}

	if eventValue.Body.NewHair != 30100 {
		t.Errorf("Expected new hair 30100, got %d", eventValue.Body.NewHair)
	}
}

func TestFaceChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test face change with message buffer
	updatePayload := character.RestModel{
		Id:   createdCharacter.Id(),
		Face: 20100,
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.Face() != 20100 {
		t.Errorf("Expected face 20100, got %d", updatedCharacter.Face())
	}

	// Verify FACE_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected FACE_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one FACE_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 FACE_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("FACE_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("FACE_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a FACE_CHANGED event
	// The event should be a StatusEvent[StatusEventFaceChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventFaceChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeFaceChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeFaceChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldFace != 20000 {
		t.Errorf("Expected old face 20000, got %d", eventValue.Body.OldFace)
	}

	if eventValue.Body.NewFace != 20100 {
		t.Errorf("Expected new face 20100, got %d", eventValue.Body.NewFace)
	}
}

func TestGenderChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test gender change with message buffer
	updatePayload := character.RestModel{
		Id:     createdCharacter.Id(),
		Gender: 1,
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.Gender() != 1 {
		t.Errorf("Expected gender 1, got %d", updatedCharacter.Gender())
	}

	// Verify GENDER_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected GENDER_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one GENDER_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 GENDER_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("GENDER_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("GENDER_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a GENDER_CHANGED event
	// The event should be a StatusEvent[StatusEventGenderChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventGenderChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeGenderChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeGenderChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldGender != 0 {
		t.Errorf("Expected old gender 0, got %d", eventValue.Body.OldGender)
	}

	if eventValue.Body.NewGender != 1 {
		t.Errorf("Expected new gender 1, got %d", eventValue.Body.NewGender)
	}
}

func TestSkinColorChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test skin color change with message buffer
	updatePayload := character.RestModel{
		Id:        createdCharacter.Id(),
		SkinColor: 5,
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.SkinColor() != 5 {
		t.Errorf("Expected skin color 5, got %d", updatedCharacter.SkinColor())
	}

	// Verify SKIN_COLOR_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected SKIN_COLOR_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one SKIN_COLOR_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 SKIN_COLOR_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("SKIN_COLOR_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("SKIN_COLOR_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a SKIN_COLOR_CHANGED event
	// The event should be a StatusEvent[StatusEventSkinColorChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventSkinColorChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeSkinColorChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeSkinColorChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldSkinColor != 0 {
		t.Errorf("Expected old skin color 0, got %d", eventValue.Body.OldSkinColor)
	}

	if eventValue.Body.NewSkinColor != 5 {
		t.Errorf("Expected new skin color 5, got %d", eventValue.Body.NewSkinColor)
	}
}

func TestGmChangedEventEmission(t *testing.T) {
	// Setup test database
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	// Create a character to update
	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("OriginalName").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetMapId(_map.Id(40000)).
		SetGm(0). // Initially not a GM
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	// Create message buffer to capture events
	buf := message.NewBuffer()

	// Test GM status change with message buffer
	updatePayload := character.RestModel{
		Id: createdCharacter.Id(),
		Gm: 1, // Change to GM level 1
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify the character was updated in the database
	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}

	if updatedCharacter.GM() != 1 {
		t.Errorf("Expected GM status 1, got %d", updatedCharacter.GM())
	}

	// Verify GM_CHANGED event was buffered
	bufferedMessages := buf.GetAll()
	statusMessages, exists := bufferedMessages[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) == 0 {
		t.Fatal("Expected GM_CHANGED event to be buffered to character status topic")
	}

	// Should have exactly one GM_CHANGED event
	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 GM_CHANGED event, got %d", len(statusMessages))
	}

	// Verify the event message structure
	eventMessage := statusMessages[0]
	if eventMessage.Key == nil {
		t.Error("GM_CHANGED event should have a key")
	}

	if eventMessage.Value == nil {
		t.Error("GM_CHANGED event should have a value")
	}

	// Parse the event value to verify it's a GM_CHANGED event
	// The event should be a StatusEvent[StatusEventGmChangedBody]
	var eventValue character2.StatusEvent[character2.StatusEventGmChangedBody]
	err = json.Unmarshal(eventMessage.Value, &eventValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}

	// Verify event contents
	if eventValue.Type != character2.StatusEventTypeGmChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeGmChanged, eventValue.Type)
	}

	if eventValue.CharacterId != createdCharacter.Id() {
		t.Errorf("Expected character ID %d, got %d", createdCharacter.Id(), eventValue.CharacterId)
	}

	if eventValue.WorldId != world.Id(0) {
		t.Errorf("Expected world ID 0, got %d", eventValue.WorldId)
	}

	if eventValue.TransactionId != transactionId {
		t.Errorf("Expected transaction ID %s, got %s", transactionId, eventValue.TransactionId)
	}

	if eventValue.Body.OldGm != false {
		t.Errorf("Expected old GM status false, got %t", eventValue.Body.OldGm)
	}

	if eventValue.Body.NewGm != true {
		t.Errorf("Expected new GM status true, got %t", eventValue.Body.NewGm)
	}
}