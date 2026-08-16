// scope_test is an external test package: it exercises Strict/AuthorizeWrite
// against templates.Entity, and templates/overlay.go imports scope, so a
// same-package test would be a compile-time import cycle. An external test
// package is allowed to import both scope and templates (R13-1).
package scope_test

import (
	"atlas-configurations/scope"
	"atlas-configurations/templates"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// sqliteTemplateEntity is a SQLite-compatible schema for the "templates"
// table - templates.Entity's gorm tags (uuid type, uuid_generate_v4()
// default) are Postgres-only and fail AutoMigrate under sqlite. This type is
// used ONLY to create/seed the table; Strict/OverlayX and Find scan directly
// into templates.Entity by column name via reflection, so the two types
// never need to match beyond column names.
type sqliteTemplateEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (sqliteTemplateEntity) TableName() string {
	return "templates"
}

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	// :memory: is per-connection; the default pool can open more than one
	// connection and silently query an empty database.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&sqliteTemplateEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// seedTemplates inserts one arbitrary row per environment given, each with a
// distinct version key so rows never collide.
func seedTemplates(t *testing.T, db *gorm.DB, environments ...string) {
	t.Helper()
	for i, e := range environments {
		row := sqliteTemplateEntity{
			Id:           uuid.New(),
			Region:       "GMS",
			MajorVersion: uint16(80 + i),
			MinorVersion: 1,
			Data:         json.RawMessage("{}"),
			Environment:  e,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("failed to seed template for environment %q: %v", e, err)
		}
	}
}

func TestStrictWithEmptyEnvironmentAppliesNoFilter(t *testing.T) {
	db := testDatabase(t)
	seedTemplates(t, db, "main", "pr-123")

	var rows []templates.Entity
	if err := scope.Strict(db.Model(&templates.Entity{}), env.Id("")).Find(&rows).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("legacy read returned %d rows, want 2 (unfiltered)", len(rows))
	}
}

func TestStrictFiltersToTheCallersEnvironment(t *testing.T) {
	db := testDatabase(t)
	seedTemplates(t, db, "main", "pr-123")

	var rows []templates.Entity
	if err := scope.Strict(db.Model(&templates.Entity{}), env.Id("pr-123")).Find(&rows).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(rows) != 1 || rows[0].Environment != "pr-123" {
		t.Fatalf("got %d rows (%v), want exactly pr-123's", len(rows), rows)
	}
}

func TestAuthorizeWriteRejectsAnotherEnvironment(t *testing.T) {
	if err := scope.AuthorizeWrite(env.Id("pr-123"), env.Id("main")); !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("got %v, want ErrCrossEnvironmentWrite", err)
	}
	if err := scope.AuthorizeWrite(env.Id("pr-123"), env.Id("pr-123")); err != nil {
		t.Fatalf("same-environment write rejected: %v", err)
	}
	if err := scope.AuthorizeWrite(env.Id(""), env.Id("")); err != nil {
		t.Fatalf("legacy write rejected: %v", err)
	}
}
