package ranking

import (
	"testing"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
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

func TestMakeCarriesDisplayFields(t *testing.T) {
	e := Entity{CharacterId: 5, Name: "Gamma", Level: 50, JobId: 412, JobCategory: 4, OverallRank: 3}
	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if m.Name() != "Gamma" || m.Level() != 50 || m.JobId() != 412 {
		t.Fatalf("Make dropped display fields: name=%q level=%d job=%d", m.Name(), m.Level(), m.JobId())
	}
}
