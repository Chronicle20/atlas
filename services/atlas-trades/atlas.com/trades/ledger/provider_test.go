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
// returns two entries here.
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

// TestByCharacterExistsSubqueryIsTenantScoped pins the tenant_id filter INSIDE
// the EXISTS subquery, which the outer query's own filter cannot substitute
// for. A foreign-tenant side row pointing at this tenant's entry is planted
// directly — the administrator would never write one, but a bug, a restore or
// a future cross-tenant feature could — and the lookup must not match on it.
func TestByCharacterExistsSubqueryIsTenantScoped(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	mine := recordTrade(t, db, tenantA, time.Now(), 100, 200)

	// Tenant B's side row, attached to tenant A's entry, for a character that
	// appears nowhere in tenant A's own sides.
	if err := db.Create(&Side{
		Id:            uuid.New(),
		TenantId:      tenantB,
		EntryId:       mine.Id(),
		CharacterId:   999,
		CharacterName: "Planted",
	}).Error; err != nil {
		t.Fatalf("plant foreign-tenant side: %v", err)
	}

	from, to := alwaysWindow()
	found, err := byCharacter(db, tenantA)(999, from, to)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("character 999 belongs to tenant B only: got %d entries, want 0", len(found))
	}
}

// TestPreloadsExcludeForeignTenantChildRows pins the tenant_id filter on both
// preloads. Child rows are fetched by foreign key in their own SELECT, which
// the parent query's filter never reaches, so a foreign-tenant side or item
// pointing at this tenant's parent would otherwise be served inside a local
// entry — an unrelated tenant's character name and item on a GM's screen.
func TestPreloadsExcludeForeignTenantChildRows(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	mine := recordTrade(t, db, tenantA, time.Now(), 100, 200)

	before, err := byId(db, tenantA)(mine.Id())
	if err != nil {
		t.Fatalf("read before planting: %v", err)
	}
	if len(before.Sides()) != 2 || len(before.Sides()[0].Items()) != 1 {
		t.Fatalf("baseline: got %d sides and %d items on the first, want 2 and 1",
			len(before.Sides()), len(before.Sides()[0].Items()))
	}

	plantedSideId := uuid.New()
	if err := db.Create(&Side{
		Id:            plantedSideId,
		TenantId:      tenantB,
		EntryId:       mine.Id(),
		CharacterId:   999,
		CharacterName: "Planted",
	}).Error; err != nil {
		t.Fatalf("plant foreign-tenant side: %v", err)
	}
	// An item of tenant B's hanging off tenant A's own side (side ids are
	// unique, so this reaches the Sides.Items preload rather than the dead
	// planted side above).
	if err := db.Create(&ItemRow{
		Id:       uuid.New(),
		TenantId: tenantB,
		SideId:   before.Sides()[0].Id(),
		ItemId:   4000000,
		Quantity: 1,
	}).Error; err != nil {
		t.Fatalf("plant foreign-tenant item: %v", err)
	}

	after, err := byId(db, tenantA)(mine.Id())
	if err != nil {
		t.Fatalf("read after planting: %v", err)
	}
	if len(after.Sides()) != 2 {
		t.Errorf("sides: got %d, want 2 (tenant B's side must not be preloaded)", len(after.Sides()))
	}
	for _, s := range after.Sides() {
		if s.CharacterName() == "Planted" {
			t.Errorf("tenant B's side leaked into tenant A's entry")
		}
		for _, i := range s.Items() {
			if i.ItemId() == 4000000 {
				t.Errorf("tenant B's item leaked onto tenant A's side %d", s.CharacterId())
			}
		}
	}
}

// TestSidesAreOrderedDeterministically pins the preload's ORDER BY. Without it
// the row order is whatever the storage engine returns, and Task 19 or the REST
// layer indexing Sides()[0] would be reading a coin flip.
func TestSidesAreOrderedDeterministically(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)

	// Added high-id-first, so an unordered preload has an insertion order to
	// disagree with.
	entry := NewBuilder(uuid.New(), field.NewBuilder(1, 2, 100000000).Build(), miniroom.Trade).
		AddSide(900, "High", 0, 0, 0, []Item{NewItem(2000000, 1, nil, nil), NewItem(1000000, 1, nil, nil)}).
		AddSide(100, "Low", 0, 0, 0, nil).
		Build()
	stored, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	readBack, err := byId(db, tenantId)(stored.Id())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got := []character.Id{readBack.Sides()[0].CharacterId(), readBack.Sides()[1].CharacterId()}; got[0] != 100 || got[1] != 900 {
		t.Errorf("side order: got %v, want [100 900] (ascending character id)", got)
	}
	high, ok := sideFor(readBack, 900)
	if !ok {
		t.Fatalf("lost the high side")
	}
	if got := []uint32{uint32(high.Items()[0].ItemId()), uint32(high.Items()[1].ItemId())}; got[0] != 1000000 || got[1] != 2000000 {
		t.Errorf("item order: got %v, want [1000000 2000000] (ascending item id)", got)
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
