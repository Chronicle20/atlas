package test

import (
	"atlas-quest/quest"
	"atlas-quest/quest/progress"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// SetupTestDB creates an in-memory SQLite database for testing.
//
// The DSN uses a uniquely-named shared-cache in-memory database rather than a
// bare ":memory:". A bare ":memory:" database is private to a single
// connection, so once database.ExecuteTransaction opens a real transaction
// (task-119 fixed it from a no-op) a query issued on the root handle inside
// that transaction lands on a second pooled connection whose schema is empty
// ("no such table"). A shared-cache database is visible to every connection in
// the pool; the unique name keeps each test isolated from the others.
func SetupTestDB(t *testing.T) *gorm.DB {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Keep at least one connection alive for the lifetime of the test so the
	// shared-cache database is not dropped when the pool would otherwise close
	// its last connection.
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetConnMaxIdleTime(0)
	}

	database.RegisterTenantCallbacks(l, db)

	// Run migrations
	if err := db.AutoMigrate(&quest.Entity{}, &progress.Entity{}); err != nil {
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
