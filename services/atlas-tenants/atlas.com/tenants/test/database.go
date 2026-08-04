package test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/tenant"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// gorm's sqlite driver hands out a fresh, empty in-memory database per
	// pooled connection, so a goroutine that checks out a second connection
	// (e.g. libs/atlas-seeder's background seed run) sees "no such table"
	// against an otherwise-migrated schema. Capping the pool to a single
	// connection keeps every query on the one connection that actually has
	// the migrated schema.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// Run migrations
	if err := db.AutoMigrate(&tenant.Entity{}, &configuration.Entity{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// CleanupTestDB closes the database connection
func CleanupTestDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}
