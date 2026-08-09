package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
)

// recordTrade writes one settled trade between two characters and returns it.
func recordTrade(t *testing.T, db *gorm.DB, tenantId uuid.UUID, settledAt time.Time, giver character.Id, receiver character.Id) Model {
	t.Helper()
	entry := NewBuilder(uuid.New(), field.NewBuilder(1, 2, 100000000).Build(), miniroom.Trade).
		SetSettledAt(settledAt).
		AddSide(giver, "Giver", 1_000, 50, 0, []Item{NewItem(2000000, 3, nil, nil)}).
		AddSide(receiver, "Receiver", 0, 0, 950, nil).
		Build()
	stored, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("record trade: %v", err)
	}
	return stored
}

// alwaysWindow is a time range wide enough to hold every trade a test records.
func alwaysWindow() (time.Time, time.Time) {
	return time.Time{}, time.Now().Add(time.Hour)
}

// TestByCharacterMatchesEitherSide pins FR-7.2: the GM lookup finds a trade
// whether the character gave or received.
func TestByCharacterMatchesEitherSide(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	stored := recordTrade(t, db, tenantId, time.Now(), 100, 200)
	from, to := alwaysWindow()

	for _, id := range []character.Id{100, 200} {
		found, err := byCharacter(db, tenantId)(id, from, to)
		if err != nil {
			t.Fatalf("lookup %d: %v", id, err)
		}
		if len(found) != 1 {
			t.Fatalf("lookup %d: got %d entries, want 1", id, len(found))
		}
		if found[0].Id() != stored.Id() {
			t.Errorf("lookup %d: got entry %s, want %s", id, found[0].Id(), stored.Id())
		}
		if len(found[0].Sides()) != 2 {
			t.Errorf("lookup %d: sides not preloaded, got %d", id, len(found[0].Sides()))
		}
	}
}

// TestByCharacterIgnoresUninvolvedCharacters proves the lookup filters at all:
// a character who traded with nobody in this entry gets nothing back.
func TestByCharacterIgnoresUninvolvedCharacters(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	recordTrade(t, db, tenantId, time.Now(), 100, 200)
	from, to := alwaysWindow()

	found, err := byCharacter(db, tenantId)(300, from, to)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("uninvolved character: got %d entries, want 0", len(found))
	}
}

// TestByCharacterDoesNotLeakOtherTenants is the cross-tenant guard: two tenants
// each record a trade for the SAME character id, and each tenant's lookup must
// return only its own entry. Dropping the tenant_id filter from the entry load
// returns two entries here. (The side scan's own tenant_id filter is defence in
// depth — the entry load's filter alone already excludes the other tenant's
// rows — so this test does not pin it.)
func TestByCharacterDoesNotLeakOtherTenants(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	settledAt := time.Now()

	mine := recordTrade(t, db, tenantA, settledAt, 100, 200)
	theirs := recordTrade(t, db, tenantB, settledAt, 100, 200)
	from, to := alwaysWindow()

	found, err := byCharacter(db, tenantA)(100, from, to)
	if err != nil {
		t.Fatalf("tenant A lookup: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("tenant A lookup: got %d entries, want 1 (tenant B's entry must not appear)", len(found))
	}
	if found[0].Id() != mine.Id() {
		t.Errorf("tenant A lookup: got entry %s, want %s", found[0].Id(), mine.Id())
	}
	found, err = byCharacter(db, tenantB)(100, from, to)
	if err != nil {
		t.Fatalf("tenant B lookup: %v", err)
	}
	if len(found) != 1 || found[0].Id() != theirs.Id() {
		t.Fatalf("tenant B lookup: got %v, want exactly [%s]", found, theirs.Id())
	}
}

// TestByCharacterFiltersOnTimeRange pins the FR-7.2 time-range filter and its
// inclusive bounds.
func TestByCharacterFiltersOnTimeRange(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	older := recordTrade(t, db, tenantId, base.Add(-2*time.Hour), 100, 200)
	newer := recordTrade(t, db, tenantId, base, 100, 200)

	found, err := byCharacter(db, tenantId)(100, base.Add(-time.Hour), base.Add(time.Hour))
	if err != nil {
		t.Fatalf("windowed lookup: %v", err)
	}
	if len(found) != 1 || found[0].Id() != newer.Id() {
		t.Fatalf("windowed lookup: got %d entries, want only %s", len(found), newer.Id())
	}

	found, err = byCharacter(db, tenantId)(100, base.Add(-2*time.Hour), base)
	if err != nil {
		t.Fatalf("inclusive lookup: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("inclusive lookup: got %d entries, want 2 (both bounds are inclusive)", len(found))
	}
	// Newest first.
	if found[0].Id() != newer.Id() || found[1].Id() != older.Id() {
		t.Errorf("ordering: got [%s %s], want [%s %s]", found[0].Id(), found[1].Id(), newer.Id(), older.Id())
	}
}

// TestByIdDoesNotLeakOtherTenants is the single-entry cross-tenant guard: one
// tenant must not be able to read another tenant's entry by guessing its id.
func TestByIdDoesNotLeakOtherTenants(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	theirs := recordTrade(t, db, tenantB, time.Now(), 100, 200)

	if _, err := byId(db, tenantA)(theirs.Id()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("reading another tenant's entry: got error %v, want gorm.ErrRecordNotFound", err)
	}

	mine, err := byId(db, tenantB)(theirs.Id())
	if err != nil {
		t.Fatalf("owning tenant must still read it: %v", err)
	}
	if mine.Id() != theirs.Id() {
		t.Errorf("owning tenant read: got %s, want %s", mine.Id(), theirs.Id())
	}
}

// TestByTransactionIdDoesNotLeakOtherTenants guards the read the idempotency
// path depends on: if it were unscoped, one tenant's settlement would be
// silently answered with another tenant's ledger entry.
func TestByTransactionIdDoesNotLeakOtherTenants(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()

	entry := NewBuilder(uuid.New(), field.NewBuilder(1, 2, 100000000).Build(), miniroom.Trade).
		AddSide(100, "Alice", 0, 0, 0, nil).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()
	if _, err := create(db, tenantB)(entry); err != nil {
		t.Fatalf("tenant B create: %v", err)
	}

	if _, err := byTransactionId(db, tenantA)(entry.TransactionId()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("cross-tenant transaction lookup: got error %v, want gorm.ErrRecordNotFound", err)
	}
	if _, err := byTransactionId(db, tenantB)(entry.TransactionId()); err != nil {
		t.Errorf("owning tenant transaction lookup: %v", err)
	}
}
