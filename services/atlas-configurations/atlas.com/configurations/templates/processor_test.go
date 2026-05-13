package templates

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testEntity is a SQLite-compatible version of Entity for testing
type testEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
}

func (testEntity) TableName() string {
	return "templates"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Use SQLite-compatible schema
	err = db.AutoMigrate(&testEntity{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func createTestRestModel(region string, majorVersion, minorVersion uint16) RestModel {
	return RestModel{
		Region:       region,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		UsesPin:      true,
	}
}

func TestProcessor_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	results, err := p.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProcessor_GetAll_WithData(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create test data
	input1 := createTestRestModel("GMS", 83, 1)
	input2 := createTestRestModel("SEA", 83, 2)

	_, err := p.Create(input1)
	if err != nil {
		t.Fatalf("failed to create first template: %v", err)
	}

	_, err = p.Create(input2)
	if err != nil {
		t.Fatalf("failed to create second template: %v", err)
	}

	results, err := p.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestProcessor_Create(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)

	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}

	// Verify it was created
	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get created template: %v", err)
	}

	if result.Region != input.Region {
		t.Errorf("expected region '%s', got '%s'", input.Region, result.Region)
	}
	if result.MajorVersion != input.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", input.MajorVersion, result.MajorVersion)
	}
	if result.MinorVersion != input.MinorVersion {
		t.Errorf("expected minorVersion %d, got %d", input.MinorVersion, result.MinorVersion)
	}
	if result.UsesPin != input.UsesPin {
		t.Errorf("expected usesPin %v, got %v", input.UsesPin, result.UsesPin)
	}
}

func TestProcessor_GetById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}

	if result.Id != id.String() {
		t.Errorf("expected id '%s', got '%s'", id.String(), result.Id)
	}
	if result.Region != input.Region {
		t.Errorf("expected region '%s', got '%s'", input.Region, result.Region)
	}
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	_, err := p.GetById(nonExistentId)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestProcessor_GetByRegionAndVersion(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	input := createTestRestModel("GMS", 83, 1)
	_, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	result, err := p.GetByRegionAndVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to get template by region/version: %v", err)
	}

	if result.Region != "GMS" {
		t.Errorf("expected region 'GMS', got '%s'", result.Region)
	}
	if result.MajorVersion != 83 {
		t.Errorf("expected majorVersion 83, got %d", result.MajorVersion)
	}
	if result.MinorVersion != 1 {
		t.Errorf("expected minorVersion 1, got %d", result.MinorVersion)
	}
}

func TestProcessor_GetByRegionAndVersion_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	_, err := p.GetByRegionAndVersion("NONEXISTENT", 99, 99)
	if err == nil {
		t.Error("expected error for non-existent region/version")
	}
}

func TestProcessor_UpdateById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create initial template
	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Update the template
	updated := createTestRestModel("SEA", 84, 2)
	updated.UsesPin = false
	err = p.UpdateById(id, updated)
	if err != nil {
		t.Fatalf("failed to update template: %v", err)
	}

	// Verify the update
	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}

	if result.Region != updated.Region {
		t.Errorf("expected region '%s', got '%s'", updated.Region, result.Region)
	}
	if result.MajorVersion != updated.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", updated.MajorVersion, result.MajorVersion)
	}
	if result.UsesPin != updated.UsesPin {
		t.Errorf("expected usesPin %v, got %v", updated.UsesPin, result.UsesPin)
	}
}

func TestProcessor_UpdateById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	input := createTestRestModel("GMS", 83, 1)
	err := p.UpdateById(nonExistentId, input)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestProcessor_DeleteById(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create a template
	input := createTestRestModel("GMS", 83, 1)
	id, err := p.Create(input)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Verify it exists
	_, err = p.GetById(id)
	if err != nil {
		t.Fatalf("template should exist before delete: %v", err)
	}

	// Delete it
	err = p.DeleteById(id)
	if err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}

	// Verify it's gone
	_, err = p.GetById(id)
	if err == nil {
		t.Error("expected error for deleted template")
	}
}

func TestProcessor_DeleteById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	err := p.DeleteById(nonExistentId)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestMake(t *testing.T) {
	testId := uuid.New()
	testData := RestModel{
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		UsesPin:      true,
	}
	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	entity := Entity{
		Id:           testId,
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Data:         jsonData,
	}

	result, err := Make(entity)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	if result.Id != testId.String() {
		t.Errorf("expected id '%s', got '%s'", testId.String(), result.Id)
	}
	if result.Region != testData.Region {
		t.Errorf("expected region '%s', got '%s'", testData.Region, result.Region)
	}
	if result.MajorVersion != testData.MajorVersion {
		t.Errorf("expected majorVersion %d, got %d", testData.MajorVersion, result.MajorVersion)
	}
	if result.UsesPin != testData.UsesPin {
		t.Errorf("expected usesPin %v, got %v", testData.UsesPin, result.UsesPin)
	}
}

func TestMake_InvalidJSON(t *testing.T) {
	entity := Entity{
		Id:           uuid.New(),
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Data:         json.RawMessage(`{invalid json`),
	}

	_, err := Make(entity)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
