package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
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