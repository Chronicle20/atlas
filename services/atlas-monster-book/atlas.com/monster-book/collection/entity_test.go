package collection

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityTableName(t *testing.T) {
	var e entity
	if got := e.TableName(); got != "monster_book_collections" {
		t.Fatalf("expected monster_book_collections, got %q", got)
	}
}

func TestMigrationCreatesTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if !db.Migrator().HasTable(&entity{}) {
		t.Fatal("expected monster_book_collections to exist after migration")
	}
}
