package test

import (
	"atlas-skills/macro"
	"atlas-skills/skill"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	logtest "github.com/sirupsen/logrus/hooks/test"
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

	// Run migrations
	if err := db.AutoMigrate(&skill.Entity{}, &macro.Entity{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	l, _ := logtest.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)

	return db
}

// CleanupTestDB closes the database connection
func CleanupTestDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}
