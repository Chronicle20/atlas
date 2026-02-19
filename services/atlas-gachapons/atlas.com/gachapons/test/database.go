package test

import (
	"context"
	"os"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates a new SQLite in-memory database for testing
func SetupTestDB(t *testing.T, migrations ...func(db *gorm.DB) error) *gorm.DB {
	// Create a new logger that writes to the test log
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Open an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Register tenant callbacks for automatic tenant filtering
	database.RegisterTenantCallbacks(logger, db)

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
