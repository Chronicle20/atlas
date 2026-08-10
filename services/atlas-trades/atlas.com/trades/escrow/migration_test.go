package escrow

import (
	"testing"
)

// legacyItemEntity is the pre-snapshot row shape, kept only in this test so the
// migration can be driven against a table that actually has the stale columns.
type legacyItemEntity struct {
	Id           string `gorm:"column:id;primaryKey"`
	RingId       uint32 `gorm:"column:ring_id;not null"`
	ItemLevel    byte   `gorm:"column:item_level;not null"`
	ItemExp      uint32 `gorm:"column:item_exp;not null"`
	ViciousCount uint32 `gorm:"column:vicious_count;not null"`
}

func (legacyItemEntity) TableName() string { return itemTable }

// TestMigrationDropsStaleColumns pins that a database already migrated to the
// pre-snapshot shape loses the columns nothing writes any more. AutoMigrate adds
// but never drops, and each stale column was created NOT NULL with no default —
// so leaving one behind makes every subsequent INSERT fail.
func TestMigrationDropsStaleColumns(t *testing.T) {
	db := testDb(t)
	if err := db.AutoMigrate(&legacyItemEntity{}); err != nil {
		t.Fatalf("legacy AutoMigrate: %v", err)
	}
	for _, c := range staleItemColumns {
		if !db.Migrator().HasColumn(&ItemEntity{}, c) {
			t.Fatalf("precondition: legacy table is missing column %q", c)
		}
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	for _, c := range staleItemColumns {
		if db.Migrator().HasColumn(&ItemEntity{}, c) {
			t.Errorf("stale column %q survived the migration", c)
		}
	}
}
