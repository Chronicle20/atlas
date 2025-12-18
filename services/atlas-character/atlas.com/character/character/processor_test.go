package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
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
