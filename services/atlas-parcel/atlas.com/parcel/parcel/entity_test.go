package parcel

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	return db
}

func TestEntityTableName(t *testing.T) {
	if got := (&Entity{}).TableName(); got != "parcels" {
		t.Fatalf("TableName() = %q, want %q", got, "parcels")
	}
}

func TestMigrationCreatesParcels(t *testing.T) {
	db := openTestDB(t)

	if err := Migration(db); err != nil {
		t.Fatalf("Migration(db) returned error: %v", err)
	}

	if !db.Migrator().HasTable("parcels") {
		t.Fatalf("expected table %q to exist after migration", "parcels")
	}
}

func TestStatusConstants(t *testing.T) {
	cases := map[string]string{
		StatusPending:   "pending",
		StatusReceived:  "received",
		StatusDiscarded: "discarded",
		StatusExpired:   "expired",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("status constant = %q, want %q", got, want)
		}
	}
}

func TestTimers(t *testing.T) {
	if ReceivableDelay != 24*time.Hour {
		t.Fatalf("ReceivableDelay = %v, want %v", ReceivableDelay, 24*time.Hour)
	}
	if ExpiryWindow != 720*time.Hour {
		t.Fatalf("ExpiryWindow = %v, want %v", ExpiryWindow, 720*time.Hour)
	}
}
