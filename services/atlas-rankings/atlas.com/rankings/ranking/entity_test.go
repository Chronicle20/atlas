package ranking

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	database.RegisterTenantCallbacks(logrus.New(), db)
	return db
}

func TestMigrationCreatesTables(t *testing.T) {
	db := testDatabase(t)
	if !db.Migrator().HasTable("character_rankings") {
		t.Fatal("character_rankings table not created")
	}
	if !db.Migrator().HasTable("ranking_cycles") {
		t.Fatal("ranking_cycles table not created")
	}
}
