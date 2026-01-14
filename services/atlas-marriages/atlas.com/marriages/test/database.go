package test

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates a new SQLite in-memory database for testing
func SetupTestDB(t *testing.T, migrations ...func(db *gorm.DB) error) *gorm.DB {
	// Open an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	for _, migration := range migrations {
		if err := migration(db); err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}
	}

	return db
}

// CreateTestContext creates a context with a test tenant for testing
func CreateTestContext() context.Context {
	return tenant.WithContext(context.Background(), CreateDefaultMockTenant())
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database connection: %v", err)
	}

	err = sqlDB.Close()
	if err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}
}
