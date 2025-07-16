package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"

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
