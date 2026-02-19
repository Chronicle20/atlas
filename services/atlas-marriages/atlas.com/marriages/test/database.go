package test

import (
	"testing"

	database "github.com/Chronicle20/atlas-database"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates a new SQLite in-memory database for testing
func SetupTestDB(t *testing.T, migrations ...func(db *gorm.DB) error) *gorm.DB {
	l := logrus.New()

	// Open an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	// Run migrations
	for _, migration := range migrations {
		if err := migration(db); err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}
	}

	return db
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
