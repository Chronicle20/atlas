package environmentcol

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The production entities (tenants.Entity, templates.Entity, services.Entity)
// declare `default:uuid_generate_v4()`, a postgres-only function that SQLite's
// CREATE TABLE syntax cannot express. Every other package in this module
// tests against a SQLite-compatible shadow of the entity (see
// templates/processor_test.go, tenants/processor_test.go,
// services/processor_test.go); this test follows the same pattern rather
// than AutoMigrate-ing the production structs directly.

type testTenantEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (testTenantEntity) TableName() string { return "tenants" }

type testTemplateEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (testTemplateEntity) TableName() string { return "templates" }

type testServiceEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	Type        string          `gorm:"type:text;not null"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testServiceEntity) TableName() string { return "services" }

type testServiceHistoryEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	ServiceId   uuid.UUID       `gorm:"type:text"`
	Type        string          `gorm:"type:text;not null"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	CreatedAt   time.Time       `gorm:"not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testServiceHistoryEntity) TableName() string { return "service_history" }

// testDatabase creates an in-memory SQLite database migrated with the
// SQLite-compatible shadow entities for all four control-plane tables
// BackfillEnvironment touches.
func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&testTenantEntity{}, &testTemplateEntity{}, &testServiceEntity{}, &testServiceHistoryEntity{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestBackfillAssignsExistingRowsToTheBaseline(t *testing.T) {
	db := testDatabase(t)

	// A row written before the column existed has the zero value.
	db.Exec(`INSERT INTO templates (id, region, major_version, minor_version, data, environment) VALUES (?, 'GMS', 83, 1, '{}', '')`, uuid.New())

	if err := BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment: %v", err)
	}

	var got string
	db.Raw(`SELECT environment FROM templates LIMIT 1`).Scan(&got)
	if got != "main" {
		t.Fatalf("environment = %q, want \"main\"", got)
	}
}

func TestBackfillIsIdempotent(t *testing.T) {
	db := testDatabase(t)

	// A row that already carries a non-baseline environment must survive a
	// second (or first) backfill run untouched.
	id := uuid.New()
	db.Exec(`INSERT INTO templates (id, region, major_version, minor_version, data, environment) VALUES (?, 'GMS', 83, 1, '{}', 'pr-42')`, id)

	if err := BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment (first run): %v", err)
	}
	if err := BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment (second run): %v", err)
	}

	var got string
	db.Raw(`SELECT environment FROM templates WHERE id = ?`, id).Scan(&got)
	if got != "pr-42" {
		t.Fatalf("environment = %q, want unchanged \"pr-42\"", got)
	}
}
