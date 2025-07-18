package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"fmt"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, character.Migration)

	for _, migrator := range migrators {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestCreateSunny(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())

	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()

	c, err := character.NewProcessor(testLogger(), tctx, testDatabase(t)).Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	if c.AccountId() != 1000 {
		t.Fatalf("Account id should be 1000, was %d", c.AccountId())
	}
	if c.WorldId() != 0 {
		t.Fatalf("World id should be 0, was %d", c.WorldId())
	}
	if c.Name() != "Atlas" {
		t.Fatalf("Name should be Atlas")
	}
	if c.Level() != 1 {
		t.Fatalf("Level should be 1, was %d", c.Level())
	}
	if c.Experience() != 0 {
		t.Fatalf("Experience should be 0, was %d", c.Experience())
	}
}

func TestGetByIdWithZeroCharacter(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	// Create a character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("ZeroTest").SetLevel(1).SetExperience(0).Build()
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))
	created, err := cp.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Retrieve the character using the ID assigned by the database
	retrieved, err := cp.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve character with ID %d: %v", created.Id(), err)
	}
	if retrieved.Id() != created.Id() {
		t.Fatalf("Character ID should be %d, was %d", created.Id(), retrieved.Id())
	}
	if retrieved.Name() != "ZeroTest" {
		t.Fatalf("Character name should be ZeroTest, was %s", retrieved.Name())
	}
}

func TestGetByIdWithNonZeroCharacter(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	// Create a character
	input := character.NewModelBuilder().SetAccountId(2000).SetWorldId(0).SetName("NonZeroTest").SetLevel(1).SetExperience(0).Build()
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))
	created, err := cp.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Retrieve the character using the ID assigned by the database
	retrieved, err := cp.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve character with ID %d: %v", created.Id(), err)
	}
	if retrieved.Id() != created.Id() {
		t.Fatalf("Character ID should be %d, was %d", created.Id(), retrieved.Id())
	}
	if retrieved.Name() != "NonZeroTest" {
		t.Fatalf("Character name should be NonZeroTest, was %s", retrieved.Name())
	}
}

func TestCreateAndEmitWithInvalidName(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Test with invalid name - too short
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Ab").SetLevel(1).SetExperience(0).Build()

	processor := character.NewProcessor(testLogger(), tctx, testDatabase(t))
	_, err := processor.CreateAndEmit(uuid.New(), input)
	
	// Should get an error due to invalid name
	if err == nil {
		t.Fatal("Expected error for invalid name, but got none")
	}
	
	// Test with invalid name - contains invalid characters
	input2 := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Test@Name!").SetLevel(1).SetExperience(0).Build()

	_, err2 := processor.CreateAndEmit(uuid.New(), input2)
	
	// Should get an error due to invalid name
	if err2 == nil {
		t.Fatal("Expected error for invalid name with special characters, but got none")
	}
	
	// Test with invalid name - too long
	input3 := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("ThisNameIsTooLong").SetLevel(1).SetExperience(0).Build()

	_, err3 := processor.CreateAndEmit(uuid.New(), input3)
	
	// Should get an error due to invalid name
	if err3 == nil {
		t.Fatal("Expected error for invalid name that's too long, but got none")
	}
}

func TestCreateAndEmitWithDuplicateName(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first using the same pattern as working tests
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("TestDupe").SetLevel(1).SetExperience(0).Build()

	processor := character.NewProcessor(testLogger(), tctx, db)
	_, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}
	
	// Try to create another character with the same name using CreateAndEmit
	input2 := character.NewModelBuilder().SetAccountId(2000).SetWorldId(0).SetName("TestDupe").SetLevel(1).SetExperience(0).Build()

	_, err2 := processor.CreateAndEmit(uuid.New(), input2)
	
	// Should get an error due to duplicate name
	if err2 == nil {
		t.Fatal("Expected error for duplicate name, but got none")
	}
}

func TestCreateAndEmitWithInvalidLevel(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Test with invalid level - too low (0)
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("TestLevel0").SetLevel(0).SetExperience(0).Build()

	processor := character.NewProcessor(testLogger(), tctx, testDatabase(t))
	_, err := processor.CreateAndEmit(uuid.New(), input)
	
	// Should get an error due to invalid level
	if err == nil {
		t.Fatal("Expected error for invalid level 0, but got none")
	}
	
	// Test with invalid level - too high (201)
	input2 := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("TestLevel201").SetLevel(201).SetExperience(0).Build()

	_, err2 := processor.CreateAndEmit(uuid.New(), input2)
	
	// Should get an error due to invalid level
	if err2 == nil {
		t.Fatal("Expected error for invalid level 201, but got none")
	}
}

func TestUpdateValidNameChange(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("OriginalName").SetLevel(10).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update the name
	updateInput := character.RestModel{
		Name: "UpdatedName",
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character name: %v", err)
	}
	
	// Verify the name was updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Name() != "UpdatedName" {
		t.Fatalf("Expected name to be 'UpdatedName', got '%s'", updated.Name())
	}
}

func TestUpdateValidHairChange(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("HairTest").SetLevel(10).SetHair(30000).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update the hair
	updateInput := character.RestModel{
		Hair: 30100,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character hair: %v", err)
	}
	
	// Verify the hair was updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Hair() != 30100 {
		t.Fatalf("Expected hair to be 30100, got %d", updated.Hair())
	}
}

func TestUpdateValidFaceChange(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("FaceTest").SetLevel(10).SetFace(20000).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update the face
	updateInput := character.RestModel{
		Face: 20100,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character face: %v", err)
	}
	
	// Verify the face was updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Face() != 20100 {
		t.Fatalf("Expected face to be 20100, got %d", updated.Face())
	}
}

func TestUpdateValidGenderChange(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("GenderTest").SetLevel(10).SetGender(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update the gender
	updateInput := character.RestModel{
		Gender: 1,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character gender: %v", err)
	}
	
	// Verify the gender was updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Gender() != 1 {
		t.Fatalf("Expected gender to be 1, got %d", updated.Gender())
	}
}

func TestUpdateValidSkinColorChange(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("SkinTest").SetLevel(10).SetSkinColor(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update the skin color
	updateInput := character.RestModel{
		SkinColor: 5,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character skin color: %v", err)
	}
	
	// Verify the skin color was updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.SkinColor() != 5 {
		t.Fatalf("Expected skin color to be 5, got %d", updated.SkinColor())
	}
}

func TestUpdateMultipleFields(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("MultiTest").
		SetLevel(10).
		SetHair(30000).
		SetFace(20000).
		SetGender(0).
		SetSkinColor(0).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update multiple fields
	updateInput := character.RestModel{
		Name:      "NewMultiTest",
		Hair:      30200,
		Face:      20200,
		Gender:    1,
		SkinColor: 7,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}
	
	// Verify all fields were updated
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Name() != "NewMultiTest" {
		t.Fatalf("Expected name to be 'NewMultiTest', got '%s'", updated.Name())
	}
	if updated.Hair() != 30200 {
		t.Fatalf("Expected hair to be 30200, got %d", updated.Hair())
	}
	if updated.Face() != 20200 {
		t.Fatalf("Expected face to be 20200, got %d", updated.Face())
	}
	if updated.Gender() != 1 {
		t.Fatalf("Expected gender to be 1, got %d", updated.Gender())
	}
	if updated.SkinColor() != 7 {
		t.Fatalf("Expected skin color to be 7, got %d", updated.SkinColor())
	}
}

func TestUpdateInvalidName(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("OriginalName").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test invalid name - too short
	updateInput := character.RestModel{
		Name: "AB",
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for invalid name (too short), but got none")
	}
	
	// Test invalid name - contains special characters only (no valid characters)
	updateInput2 := character.RestModel{
		Name: "@!#$%",
	}
	
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput2)
	if err == nil {
		t.Fatal("Expected error for invalid name (special characters only), but got none")
	}
}

func TestUpdateDuplicateName(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create two characters first
	input1 := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("FirstChar").SetLevel(10).Build()
	input2 := character.NewModelBuilder().SetAccountId(2000).SetWorldId(0).SetName("SecondChar").SetLevel(10).Build()
	
	processor := character.NewProcessor(testLogger(), tctx, db)
	_, err := processor.Create(message.NewBuffer())(uuid.New(), input1)
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}
	
	created2, err := processor.Create(message.NewBuffer())(uuid.New(), input2)
	if err != nil {
		t.Fatalf("Failed to create second character: %v", err)
	}
	
	// Try to update second character to have same name as first
	updateInput := character.RestModel{
		Name: "FirstChar",
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created2.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for duplicate name, but got none")
	}
}

func TestUpdateInvalidHair(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("HairTest").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test invalid hair - outside valid range
	updateInput := character.RestModel{
		Hair: 99999,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for invalid hair ID, but got none")
	}
}

func TestUpdateInvalidFace(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("FaceTest").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test invalid face - outside valid range
	updateInput := character.RestModel{
		Face: 99999,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for invalid face ID, but got none")
	}
}

func TestUpdateInvalidGender(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("GenderTest").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test invalid gender - outside valid range
	updateInput := character.RestModel{
		Gender: 5,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for invalid gender, but got none")
	}
}

func TestUpdateInvalidSkinColor(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("SkinTest").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test invalid skin color - outside valid range
	updateInput := character.RestModel{
		SkinColor: 15,
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err == nil {
		t.Fatal("Expected error for invalid skin color, but got none")
	}
}

func TestUpdateNoChanges(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character first
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("NoChangeTest").SetLevel(10).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update with empty input (no changes)
	updateInput := character.RestModel{}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Update with no changes should not fail: %v", err)
	}
	
	// Verify character is unchanged
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve character: %v", err)
	}
	
	if updated.Name() != "NoChangeTest" {
		t.Fatalf("Expected name to remain 'NoChangeTest', got '%s'", updated.Name())
	}
}

func TestUpdateNonExistentCharacter(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	processor := character.NewProcessor(testLogger(), tctx, db)
	
	// Try to update a non-existent character
	updateInput := character.RestModel{
		Name: "NewName",
	}
	
	transactionId := uuid.New()
	err := processor.Update(message.NewBuffer())(transactionId, 99999, updateInput)
	if err == nil {
		t.Fatal("Expected error for non-existent character, but got none")
	}
}

func TestUpdatePreservesUnchangedValues(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	
	// Create a character with specific values
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("PreserveTest").
		SetLevel(25).
		SetHair(30000).
		SetFace(20000).
		SetGender(0).
		SetSkinColor(3).
		SetStrength(100).
		SetDexterity(50).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Update only the name
	updateInput := character.RestModel{
		Name: "PreserveTestUpdated",
	}
	
	transactionId := uuid.New()
	err = processor.Update(message.NewBuffer())(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}
	
	// Verify only name changed, other values preserved
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Name() != "PreserveTestUpdated" {
		t.Fatalf("Expected name to be 'PreserveTestUpdated', got '%s'", updated.Name())
	}
	if updated.Level() != 25 {
		t.Fatalf("Expected level to be preserved as 25, got %d", updated.Level())
	}
	if updated.Hair() != 30000 {
		t.Fatalf("Expected hair to be preserved as 30000, got %d", updated.Hair())
	}
	if updated.Face() != 20000 {
		t.Fatalf("Expected face to be preserved as 20000, got %d", updated.Face())
	}
	if updated.Gender() != 0 {
		t.Fatalf("Expected gender to be preserved as 0, got %d", updated.Gender())
	}
	if updated.SkinColor() != 3 {
		t.Fatalf("Expected skin color to be preserved as 3, got %d", updated.SkinColor())
	}
	if updated.Strength() != 100 {
		t.Fatalf("Expected strength to be preserved as 100, got %d", updated.Strength())
	}
	if updated.Dexterity() != 50 {
		t.Fatalf("Expected dexterity to be preserved as 50, got %d", updated.Dexterity())
	}
}

// Test map accessibility validation
func TestMapAccessibilityValidation(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create a low-level character (level 1)
	lowLevelChar := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("LowLevel").
		SetLevel(1).
		SetExperience(0).
		SetMapId(100000000). // Training map
		Build()

	createdLowLevel, err := cp.Create(message.NewBuffer())(uuid.New(), lowLevelChar)
	if err != nil {
		t.Fatalf("Failed to create low-level character: %v", err)
	}

	// Create a high-level character (level 80)
	highLevelChar := character.NewModelBuilder().
		SetAccountId(2000).
		SetWorldId(0).
		SetName("HighLevel").
		SetLevel(80).
		SetExperience(0).
		SetMapId(100000000). // Training map
		Build()

	createdHighLevel, err := cp.Create(message.NewBuffer())(uuid.New(), highLevelChar)
	if err != nil {
		t.Fatalf("Failed to create high-level character: %v", err)
	}

	// Create a GM character (level 1 but GM level 1)
	gmChar := character.NewModelBuilder().
		SetAccountId(3000).
		SetWorldId(0).
		SetName("GMChar").
		SetLevel(1).
		SetExperience(0).
		SetMapId(100000000). // Training map
		SetGm(1).
		Build()

	createdGM, err := cp.Create(message.NewBuffer())(uuid.New(), gmChar)
	if err != nil {
		t.Fatalf("Failed to create GM character: %v", err)
	}

	// Test cases
	tests := []struct {
		name        string
		characterId uint32
		mapId       uint32
		shouldPass  bool
		description string
	}{
		{
			name:        "Low level character accessing training map",
			characterId: createdLowLevel.Id(),
			mapId:       100000001,
			shouldPass:  true,
			description: "Level 1 character should access training maps",
		},
		{
			name:        "Low level character accessing Victoria Island",
			characterId: createdLowLevel.Id(),
			mapId:       110000000,
			shouldPass:  true,
			description: "Level 1 character should access Victoria Island",
		},
		{
			name:        "Low level character accessing advanced area",
			characterId: createdLowLevel.Id(),
			mapId:       200000000,
			shouldPass:  false,
			description: "Level 1 character should not access level 30+ areas",
		},
		{
			name:        "High level character accessing advanced area",
			characterId: createdHighLevel.Id(),
			mapId:       200000000,
			shouldPass:  true,
			description: "Level 80 character should access level 30+ areas",
		},
		{
			name:        "High level character accessing end-game area",
			characterId: createdHighLevel.Id(),
			mapId:       500000000,
			shouldPass:  true,
			description: "Level 80 character should access level 70+ areas",
		},
		{
			name:        "GM character accessing restricted area",
			characterId: createdGM.Id(),
			mapId:       500000000,
			shouldPass:  true,
			description: "GM character should access all areas regardless of level",
		},
		{
			name:        "Invalid map ID",
			characterId: createdLowLevel.Id(),
			mapId:       50000000,
			shouldPass:  false,
			description: "Invalid map ID should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation through Update method
			mapInput := character.RestModel{
				MapId: _map.Id(tt.mapId),
			}
			
			err := cp.Update(message.NewBuffer())(uuid.New(), tt.characterId, mapInput)
			
			if tt.shouldPass {
				if err != nil {
					t.Errorf("Test %s failed: %s. Expected success, but got error: %v", 
						tt.name, tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Test %s failed: %s. Expected error, but update succeeded", 
						tt.name, tt.description)
				} else if err.Error() != "invalid map ID or character cannot access this map" {
					t.Errorf("Test %s failed: %s. Expected specific error message, got: %v", 
						tt.name, tt.description, err)
				}
			}
		})
	}
}

// Test map accessibility validation through Update method
func TestUpdateMapAccessibility(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create a low-level character
	lowLevelChar := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("UpdateTest").
		SetLevel(1).
		SetExperience(0).
		SetMapId(100000000).
		Build()

	created, err := cp.Create(message.NewBuffer())(uuid.New(), lowLevelChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Test successful map update (accessible map)
	validMapInput := character.RestModel{
		MapId: _map.Id(110000000), // Victoria Island - accessible to level 1
	}

	err = cp.Update(message.NewBuffer())(uuid.New(), created.Id(), validMapInput)
	if err != nil {
		t.Fatalf("Failed to update character with accessible map: %v", err)
	}

	// Test failed map update (inaccessible map)
	invalidMapInput := character.RestModel{
		MapId: _map.Id(200000000), // Advanced area - not accessible to level 1
	}

	err = cp.Update(message.NewBuffer())(uuid.New(), created.Id(), invalidMapInput)
	if err == nil {
		t.Fatal("Expected error when updating character with inaccessible map, but got nil")
	}
	if err.Error() != "invalid map ID or character cannot access this map" {
		t.Fatalf("Expected specific error message, got: %v", err)
	}
}

// Test level-based map restrictions
func TestLevelBasedMapRestrictions(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create characters at different levels
	levels := []byte{1, 25, 35, 55, 75}
	characters := make([]character.Model, len(levels))
	
	for i, level := range levels {
		char := character.NewModelBuilder().
			SetAccountId(1000 + uint32(i)).
			SetWorldId(0).
			SetName(fmt.Sprintf("Level%d", level)).
			SetLevel(level).
			SetExperience(0).
			SetMapId(100000000).
			Build()

		created, err := cp.Create(message.NewBuffer())(uuid.New(), char)
		if err != nil {
			t.Fatalf("Failed to create character at level %d: %v", level, err)
		}
		characters[i] = created
	}

	// Test map access at different levels
	mapTests := []struct {
		mapId      uint32
		minLevel   byte
		mapName    string
	}{
		{100000000, 0, "Training area"},
		{110000000, 0, "Victoria Island"},
		{200000000, 30, "Advanced area"},
		{300000000, 50, "High-level area"},
		{500000000, 70, "End-game area"},
	}

	for _, mapTest := range mapTests {
		for i, char := range characters {
			shouldAccess := levels[i] >= mapTest.minLevel
			
			mapInput := character.RestModel{
				MapId: _map.Id(mapTest.mapId),
			}
			
			err := cp.Update(message.NewBuffer())(uuid.New(), char.Id(), mapInput)
			canAccess := (err == nil)
			
			if canAccess != shouldAccess {
				t.Errorf("Level %d character accessing %s (map %d): expected %v, got %v (error: %v)",
					levels[i], mapTest.mapName, mapTest.mapId, shouldAccess, canAccess, err)
			}
		}
	}
}

// TestMapIdValidationEdgeCases tests additional edge cases for mapId validation
func TestMapIdValidationEdgeCases(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create a regular character (level 50)
	char := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("EdgeCaseTest").
		SetLevel(50).
		SetExperience(0).
		SetMapId(100000000).
		Build()

	created, err := cp.Create(message.NewBuffer())(uuid.New(), char)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Test cases for mapId validation edge cases
	tests := []struct {
		name        string
		mapId       uint32
		shouldPass  bool
		description string
	}{
		{
			name:        "Minimum valid map ID",
			mapId:       100000000,
			shouldPass:  true,
			description: "Minimum valid map ID should be accepted",
		},
		{
			name:        "Maximum valid map ID",
			mapId:       999999999,
			shouldPass:  true,
			description: "Maximum valid map ID should be accepted",
		},
		{
			name:        "Below minimum map ID",
			mapId:       99999999,
			shouldPass:  false,
			description: "Map ID below minimum should be rejected",
		},
		{
			name:        "Above maximum map ID",
			mapId:       1000000000,
			shouldPass:  false,
			description: "Map ID above maximum should be rejected",
		},
		{
			name:        "Zero map ID",
			mapId:       0,
			shouldPass:  true, // Zero mapId should be ignored (no change)
			description: "Zero mapId should be ignored in updates",
		},
		{
			name:        "Event map boundary - minimum",
			mapId:       600000000,
			shouldPass:  true,
			description: "Event map minimum ID should be accessible to level 50",
		},
		{
			name:        "Event map boundary - maximum",
			mapId:       999999999,
			shouldPass:  true,
			description: "Event map maximum ID should be accessible to level 50",
		},
		{
			name:        "Training map boundary - minimum",
			mapId:       100000000,
			shouldPass:  true,
			description: "Training map minimum should be accessible",
		},
		{
			name:        "Training map boundary - maximum",
			mapId:       109999999,
			shouldPass:  true,
			description: "Training map maximum should be accessible",
		},
		{
			name:        "Victoria Island boundary - minimum",
			mapId:       110000000,
			shouldPass:  true,
			description: "Victoria Island minimum should be accessible",
		},
		{
			name:        "Victoria Island boundary - maximum",
			mapId:       119999999,
			shouldPass:  true,
			description: "Victoria Island maximum should be accessible",
		},
		{
			name:        "Advanced area boundary - minimum",
			mapId:       200000000,
			shouldPass:  true,
			description: "Advanced area minimum should be accessible to level 50",
		},
		{
			name:        "Advanced area boundary - maximum",
			mapId:       299999999,
			shouldPass:  true,
			description: "Advanced area maximum should be accessible to level 50",
		},
		{
			name:        "High-level area boundary - minimum",
			mapId:       300000000,
			shouldPass:  true,
			description: "High-level area minimum should be accessible to level 50",
		},
		{
			name:        "High-level area boundary - maximum",
			mapId:       399999999,
			shouldPass:  true,
			description: "High-level area maximum should be accessible to level 50",
		},
		{
			name:        "Gap between ranges",
			mapId:       400000000,
			shouldPass:  true,
			description: "Gap between ranges should default to accessible",
		},
		{
			name:        "End-game area boundary - minimum",
			mapId:       500000000,
			shouldPass:  false,
			description: "End-game area minimum should not be accessible to level 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapInput := character.RestModel{
				MapId: _map.Id(tt.mapId),
			}
			
			err := cp.Update(message.NewBuffer())(uuid.New(), created.Id(), mapInput)
			
			if tt.shouldPass {
				if err != nil {
					t.Errorf("Test %s failed: %s. Expected success, but got error: %v", 
						tt.name, tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Test %s failed: %s. Expected error, but update succeeded", 
						tt.name, tt.description)
				}
			}
		})
	}
}

// TestMapIdValidationForDifferentCharacterLevels tests mapId validation across character levels
func TestMapIdValidationForDifferentCharacterLevels(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create characters at specific boundary levels
	boundaryLevels := []byte{1, 10, 29, 30, 49, 50, 69, 70, 200}
	characters := make([]character.Model, len(boundaryLevels))
	
	for i, level := range boundaryLevels {
		char := character.NewModelBuilder().
			SetAccountId(1000 + uint32(i)).
			SetWorldId(0).
			SetName(fmt.Sprintf("BoundaryLevel%d", level)).
			SetLevel(level).
			SetExperience(0).
			SetMapId(100000000).
			Build()

		created, err := cp.Create(message.NewBuffer())(uuid.New(), char)
		if err != nil {
			t.Fatalf("Failed to create character at level %d: %v", level, err)
		}
		characters[i] = created
	}

	// Test specific boundary conditions
	tests := []struct {
		mapId       uint32
		minLevel    byte
		description string
	}{
		{200000000, 30, "Advanced area minimum level requirement"},
		{299999999, 30, "Advanced area maximum"},
		{300000000, 50, "High-level area minimum level requirement"},
		{399999999, 50, "High-level area maximum"},
		{500000000, 70, "End-game area minimum level requirement"},
		{599999999, 70, "End-game area maximum"},
		{600000000, 10, "Event area minimum level requirement"},
	}

	for _, test := range tests {
		for i, char := range characters {
			level := boundaryLevels[i]
			shouldAccess := level >= test.minLevel
			
			mapInput := character.RestModel{
				MapId: _map.Id(test.mapId),
			}
			
			err := cp.Update(message.NewBuffer())(uuid.New(), char.Id(), mapInput)
			canAccess := (err == nil)
			
			if canAccess != shouldAccess {
				t.Errorf("Level %d character accessing map %d (%s): expected %v, got %v (error: %v)",
					level, test.mapId, test.description, shouldAccess, canAccess, err)
			}
		}
	}
}

// TestMapIdValidationWithGMCharacters tests that GM characters can access all maps
func TestMapIdValidationWithGMCharacters(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create GM character at low level
	gmChar := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("GMTest").
		SetLevel(1).
		SetExperience(0).
		SetMapId(100000000).
		SetGm(1).
		Build()

	created, err := cp.Create(message.NewBuffer())(uuid.New(), gmChar)
	if err != nil {
		t.Fatalf("Failed to create GM character: %v", err)
	}

	// Verify the character was created with GM status
	retrieved, err := cp.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve created GM character: %v", err)
	}
	
	if retrieved.GM() != 1 {
		t.Fatalf("Expected GM status to be 1, but got %d", retrieved.GM())
	}

	// Test that GM can access all valid level-restricted maps
	restrictedMaps := []uint32{
		200000000, // Advanced area (normally level 30+) 
		300000000, // High-level area (normally level 50+)
		500000000, // End-game area (normally level 70+)
		600000000, // Event area (normally level 10+)
	}

	for i, mapId := range restrictedMaps {
		mapInput := character.RestModel{
			MapId: _map.Id(mapId),
		}
		
		err := cp.Update(message.NewBuffer())(uuid.New(), created.Id(), mapInput)
		if err != nil {
			t.Errorf("GM character (GM=%d) should be able to access map %d, but got error: %v", 
				retrieved.GM(), mapId, err)
		}
		
		// After each update, verify GM status is still preserved
		afterUpdate, err := cp.GetById()(created.Id())
		if err != nil {
			t.Fatalf("Failed to retrieve character after update %d: %v", i, err)
		}
		
		if afterUpdate.GM() != 1 {
			t.Errorf("GM status should be preserved after update %d. Expected 1, got %d", i, afterUpdate.GM())
		}
	}
}

// TestMapIdValidationErrorMessages tests specific error messages
func TestMapIdValidationErrorMessages(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Create a low-level character
	char := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("ErrorMsgTest").
		SetLevel(1).
		SetExperience(0).
		SetMapId(100000000).
		Build()

	created, err := cp.Create(message.NewBuffer())(uuid.New(), char)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Test specific error conditions
	errorTests := []struct {
		name           string
		mapId          uint32
		expectedError  string
		description    string
	}{
		{
			name:          "Invalid map ID - too low",
			mapId:         99999999,
			expectedError: "invalid map ID or character cannot access this map",
			description:   "Map ID below valid range should give clear error",
		},
		{
			name:          "Invalid map ID - too high",
			mapId:         1000000000,
			expectedError: "invalid map ID or character cannot access this map",
			description:   "Map ID above valid range should give clear error",
		},
		{
			name:          "Level restricted map",
			mapId:         500000000,
			expectedError: "invalid map ID or character cannot access this map",
			description:   "Level-restricted map should give clear error",
		},
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			mapInput := character.RestModel{
				MapId: _map.Id(test.mapId),
			}
			
			err := cp.Update(message.NewBuffer())(uuid.New(), created.Id(), mapInput)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", test.description)
			} else if err.Error() != test.expectedError {
				t.Errorf("Expected error message '%s', but got '%s'", test.expectedError, err.Error())
			}
		})
	}
}

// TestMapIdValidationWithNonExistentCharacter tests mapId validation with non-existent character
func TestMapIdValidationWithNonExistentCharacter(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	cp := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	// Try to update mapId for a non-existent character
	mapInput := character.RestModel{
		MapId: _map.Id(100000000),
	}
	
	err := cp.Update(message.NewBuffer())(uuid.New(), 99999, mapInput)
	if err == nil {
		t.Fatal("Expected error when updating mapId for non-existent character, but got none")
	}
	
	// The error should be about the character not being found, not about the map ID
	if err.Error() == "invalid map ID or character cannot access this map" {
		t.Error("Error should be about character not found, not map validation")
	}
}
