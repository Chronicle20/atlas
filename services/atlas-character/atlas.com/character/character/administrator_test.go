package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestSetName(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("OldName").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetName EntityUpdateFunction
	setNameFunc := character.SetName("NewName")
	columns, _ := setNameFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "Name" {
		t.Fatalf("Expected columns [Name], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		Name: "UpdatedName",
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character name: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Name() != "UpdatedName" {
		t.Fatalf("Expected updated name to be 'UpdatedName', got '%s'", updated.Name())
	}
}

func TestSetHair(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("HairTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetHair EntityUpdateFunction
	setHairFunc := character.SetHair(30100)
	columns, _ := setHairFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "Hair" {
		t.Fatalf("Expected columns [Hair], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		Hair: 30200,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character hair: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Hair() != 30200 {
		t.Fatalf("Expected updated hair to be 30200, got %d", updated.Hair())
	}
}

func TestSetFace(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("FaceTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetFace EntityUpdateFunction
	setFaceFunc := character.SetFace(20100)
	columns, _ := setFaceFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "Face" {
		t.Fatalf("Expected columns [Face], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		Face: 20200,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character face: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Face() != 20200 {
		t.Fatalf("Expected updated face to be 20200, got %d", updated.Face())
	}
}

func TestSetGender(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("GenderTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetGender EntityUpdateFunction
	setGenderFunc := character.SetGender(1)
	columns, _ := setGenderFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "Gender" {
		t.Fatalf("Expected columns [Gender], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		Gender: 0,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character gender: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Gender() != 0 {
		t.Fatalf("Expected updated gender to be 0, got %d", updated.Gender())
	}
}

func TestSetSkinColor(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("SkinTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetSkinColor EntityUpdateFunction
	setSkinColorFunc := character.SetSkinColor(3)
	columns, _ := setSkinColorFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "SkinColor" {
		t.Fatalf("Expected columns [SkinColor], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		SkinColor: 5,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character skin color: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.SkinColor() != 5 {
		t.Fatalf("Expected updated skin color to be 5, got %d", updated.SkinColor())
	}
}

func TestSetGm(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("GmTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetGm EntityUpdateFunction
	setGmFunc := character.SetGm(1)
	columns, _ := setGmFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "GM" {
		t.Fatalf("Expected columns [GM], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		Gm: 2,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character GM status: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.GM() != 2 {
		t.Fatalf("Expected updated GM status to be 2, got %d", updated.GM())
	}
}

func TestMultipleEntityUpdateFunctions(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("MultiTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test multiple update functions together via processor
	updateInput := character.RestModel{
		Name:      "UpdatedMulti",
		Hair:      30300,
		Face:      20300,
		Gender:    1,
		SkinColor: 7,
		Gm:        3,
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character with multiple functions: %v", err)
	}
	
	// Verify all updates persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.Name() != "UpdatedMulti" {
		t.Fatalf("Expected updated name to be 'UpdatedMulti', got '%s'", updated.Name())
	}
	if updated.Hair() != 30300 {
		t.Fatalf("Expected updated hair to be 30300, got %d", updated.Hair())
	}
	if updated.Face() != 20300 {
		t.Fatalf("Expected updated face to be 20300, got %d", updated.Face())
	}
	if updated.Gender() != 1 {
		t.Fatalf("Expected updated gender to be 1, got %d", updated.Gender())
	}
	if updated.SkinColor() != 7 {
		t.Fatalf("Expected updated skin color to be 7, got %d", updated.SkinColor())
	}
	if updated.GM() != 3 {
		t.Fatalf("Expected updated GM status to be 3, got %d", updated.GM())
	}
}

func TestSetMapId(t *testing.T) {
	db := testDatabase(t)
	tctx := tenant.WithContext(context.Background(), testTenant())
	
	// Create a test character
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("MapTest").SetLevel(1).SetExperience(0).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	
	// Test the SetMapId EntityUpdateFunction
	setMapIdFunc := character.SetMapId(_map.Id(110000000))
	columns, _ := setMapIdFunc()
	
	// Check that the correct columns are returned
	if len(columns) != 1 || columns[0] != "MapId" {
		t.Fatalf("Expected columns [MapId], got %v", columns)
	}
	
	// Test the dynamic update functionality via processor
	updateInput := character.RestModel{
		MapId: _map.Id(110000001),
	}
	err = processor.Update(message.NewBuffer())(uuid.New(), created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character map ID: %v", err)
	}
	
	// Verify the update persisted
	updated, err := processor.GetById()(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}
	
	if updated.MapId() != 110000001 {
		t.Fatalf("Expected updated map ID to be 110000001, got %d", updated.MapId())
	}
}