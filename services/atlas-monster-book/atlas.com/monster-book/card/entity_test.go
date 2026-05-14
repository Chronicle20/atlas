package card

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityTableName(t *testing.T) {
	if (entity{}).TableName() != "monster_book_cards" {
		t.Fatal("table name mismatch")
	}
}

func TestMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !db.Migrator().HasTable(&entity{}) {
		t.Fatal("expected monster_book_cards")
	}
}
