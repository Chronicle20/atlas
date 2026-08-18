package seeder

import (
	"atlas-configurations/templates"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// seederTestEntity mirrors templates.Entity with SQLite-compatible column
// types, the same way templates/processor_test.go's testEntity does. The
// templates package's own harness is unexported, so it is restated here.
type seederTestEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (seederTestEntity) TableName() string {
	return "templates"
}

func setupSeederTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&seederTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func readTemplateData(db *gorm.DB, id uuid.UUID) (string, error) {
	var e seederTestEntity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		return "", err
	}
	return string(e.Data), nil
}

func TestDefaultConfig(t *testing.T) {
	// Test with no environment variables set
	_ = os.Unsetenv("SEED_DATA_PATH")
	_ = os.Unsetenv("SEED_ENABLED")

	config := DefaultConfig()

	if config.SeedPath != "/seed-data" {
		t.Errorf("Expected default SeedPath to be '/seed-data', got '%s'", config.SeedPath)
	}

	if !config.Enabled {
		t.Error("Expected default Enabled to be true")
	}
}

func TestDefaultConfigWithEnvVars(t *testing.T) {
	// Test with environment variables set
	_ = os.Setenv("SEED_DATA_PATH", "/custom/path")
	_ = os.Setenv("SEED_ENABLED", "false")
	defer func() {
		_ = os.Unsetenv("SEED_DATA_PATH")
		_ = os.Unsetenv("SEED_ENABLED")
	}()

	config := DefaultConfig()

	if config.SeedPath != "/custom/path" {
		t.Errorf("Expected SeedPath to be '/custom/path', got '%s'", config.SeedPath)
	}

	if config.Enabled {
		t.Error("Expected Enabled to be false when SEED_ENABLED=false")
	}
}

func TestRunWithSeedingDisabled(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
		db:  nil, // DB not needed when disabled
		config: Config{
			SeedPath: "testdata",
			Enabled:  false,
		},
	}

	err := s.Run()
	if err != nil {
		t.Errorf("Expected no error when seeding disabled, got: %v", err)
	}
}

// THE regression guard (PRD acceptance criteria, FR-4.1/FR-4.2): boot-time
// seeding is create-if-absent. Given an existing row and a seed file whose
// content differs, the seeder must report "skipped" and leave the row
// byte-identical. This is what protects "UI edits survive a redeploy" from
// being quietly broken by a later change.
func TestSeederSkipsExistingWithDifferentContent(t *testing.T) {
	db := setupSeederTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	catalog := templates.LoadCatalog(l, filepath.Join("testdata", "templates"))
	entry, ok := catalog.Lookup("TEST", 1, 0)
	if !ok {
		t.Fatalf("TEST 1.0 fixture missing from testdata/templates")
	}

	// Pre-create a row on the same key with DIFFERENT content.
	p := templates.NewProcessor(l, ctx, db)
	existing := entry.Model
	existing.UsesPin = !existing.UsesPin
	id, err := p.Create(existing)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	before, err := readTemplateData(db, id)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	s := NewSeeder(l, ctx, db, Config{SeedPath: "testdata", Enabled: true}, catalog)
	result := s.seedTemplates()

	if result.Skipped != 1 || result.Imported != 0 || result.Failed != 0 {
		t.Fatalf("SeedResult = %+v, want {Imported:0 Skipped:1 Failed:0}", result)
	}

	after, err := readTemplateData(db, id)
	if err != nil {
		t.Fatalf("read back (after): %v", err)
	}
	if after != before {
		t.Errorf("the seeder rewrote an existing row:\nbefore: %s\nafter:  %s", before, after)
	}
}

// A key with no existing row is imported.
func TestSeederImportsMissingTemplate(t *testing.T) {
	db := setupSeederTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	catalog := templates.LoadCatalog(l, filepath.Join("testdata", "templates"))
	s := NewSeeder(l, ctx, db, Config{SeedPath: "testdata", Enabled: true}, catalog)

	result := s.seedTemplates()
	if result.Imported != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("SeedResult = %+v, want {Imported:1 Skipped:0 Failed:0}", result)
	}

	got, err := templates.NewProcessor(l, ctx, db).GetByRegionAndVersion("TEST", 1, 0)
	if err != nil {
		t.Fatalf("GetByRegionAndVersion: %v", err)
	}
	if got.Region != "TEST" {
		t.Errorf("Region = %q, want TEST", got.Region)
	}
}
