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
	// 29 days, not 30 — task-23 / RISK-4 resolution (docs/tasks/
	// task-241-duey-parcel-delivery/context.md §11): the client's receive
	// guard divides unsigned and refuses unless the quotient is < 30, and a
	// 30-day window is not survivable on the expiry sweep's own return leg
	// (zero-delay ReceivableAt). See TestReturnLegSurvivesClientExpiryGuard
	// for the client-predicate pin.
	if ExpiryWindow != 696*time.Hour {
		t.Fatalf("ExpiryWindow = %v, want %v", ExpiryWindow, 696*time.Hour)
	}
}

// TestReturnLegSurvivesClientExpiryGuard pins the exact defect RISK-4
// resolves (docs/tasks/task-241-duey-parcel-delivery/context.md §11): the
// client's CTabReceive::ReceiveParcel (v72 @0x65AF41 / v83 @0x6F0D11)
// refuses a receive unless the UNSIGNED quotient (expiresAt-now)/24h is
// strictly < 30. It asserts that predicate — written against the real
// ExpiryWindow constant, not a hardcoded duration, so it re-breaks if
// ExpiryWindow is ever raised back to 30 days — holds for a freshly
// created RETURN LEG at the exact instant it becomes receivable: a return
// leg has ReceivableAt == CreatedAt (no 24h delay), so its remaining life
// at that instant is the full ExpiryWindow, the tightest case in the
// parcel's whole lifecycle.
func TestReturnLegSurvivesClientExpiryGuard(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	createdAt := now // ReceivableAt == CreatedAt on a return leg
	expiresAt := createdAt.Add(ExpiryWindow)

	quotient := uint64(expiresAt.Sub(now)) / uint64(24*time.Hour)
	if !(quotient < 30) {
		t.Fatalf("client's unsigned (expiresAt-now)/24h guard would refuse a freshly-receivable return leg: quotient = %d, want < 30", quotient)
	}
}
