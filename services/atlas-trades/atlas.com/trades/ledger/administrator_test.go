package ledger

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// testDb returns a fresh in-memory database with the three ledger tables
// migrated. Every test gets its own, so no test can see another's rows.
func testDb(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration)
}

// testTenantId derives a tenant id from the test's name: stable across repeated
// calls within one test, and distinct between tests.
func testTenantId(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name()))
}

// testField is the world/channel/map the trade settled on.
func testField(t *testing.T) field.Model {
	t.Helper()
	return field.NewBuilder(1, 2, 100000000).Build()
}

// countEntries reports how many trade_ledger_entries rows the tenant owns.
func countEntries(t *testing.T, db *gorm.DB, tenantId uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&Entry{}).Where("tenant_id = ?", tenantId).Count(&n).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	return n
}

// TestCreateIsIdempotentPerTransaction pins FR-5.7 / design §9: a duplicate
// settle for the same transaction id returns the already-recorded entry
// without writing a second one, so a retried saga never double-records.
func TestCreateIsIdempotentPerTransaction(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	txId := uuid.New()

	entry := NewBuilder(txId, testField(t), miniroom.Trade).
		AddSide(100, "Alice", 10_000_000, 400_000, 0, nil).
		AddSide(200, "Bob", 0, 0, 9_600_000, nil).
		Build()

	first, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	second, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("duplicate create must succeed, got: %v", err)
	}
	if second.Id() != first.Id() {
		t.Errorf("duplicate create wrote a new row: got %s, want %s", second.Id(), first.Id())
	}
	if got := countEntries(t, db, tenantId); got != 1 {
		t.Errorf("entry count: got %d, want 1", got)
	}
}

// TestCreateIsIdempotentAcrossDistinctModels pins the guard to the transaction
// id rather than to the Model's own id: a retried settlement rebuilds its
// entry from scratch, so the second attempt arrives with a fresh entry id and
// fresh side ids and must still resolve to the first attempt's row.
func TestCreateIsIdempotentAcrossDistinctModels(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	txId := uuid.New()

	build := func() Model {
		return NewBuilder(txId, testField(t), miniroom.Trade).
			AddSide(100, "Alice", 0, 0, 0, nil).
			AddSide(200, "Bob", 0, 0, 0, nil).
			Build()
	}

	first, err := create(db, tenantId)(build())
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	retry := build()
	if retry.Id() == first.Id() {
		t.Fatalf("test is not exercising distinct models: both built entries share id %s", retry.Id())
	}

	second, err := create(db, tenantId)(retry)
	if err != nil {
		t.Fatalf("retried create must succeed, got: %v", err)
	}
	if second.Id() != first.Id() {
		t.Errorf("retry returned the unwritten model: got %s, want the stored %s", second.Id(), first.Id())
	}
	if got := countEntries(t, db, tenantId); got != 1 {
		t.Errorf("entry count: got %d, want 1", got)
	}
}

// TestCreateAllowsSameTransactionIdInAnotherTenant proves the idempotency
// guard is per tenant: two tenants that happen to mint the same transaction id
// each get their own entry rather than the second silently reading the first
// tenant's row.
func TestCreateAllowsSameTransactionIdInAnotherTenant(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	txId := uuid.New()

	build := func(name string) Model {
		return NewBuilder(txId, testField(t), miniroom.Trade).
			AddSide(100, name, 0, 0, 0, nil).
			AddSide(200, "Bob", 0, 0, 0, nil).
			Build()
	}

	a, err := create(db, tenantA)(build("Alice"))
	if err != nil {
		t.Fatalf("tenant A create: %v", err)
	}
	b, err := create(db, tenantB)(build("Carol"))
	if err != nil {
		t.Fatalf("tenant B create: %v", err)
	}
	if a.Id() == b.Id() {
		t.Fatalf("tenant B reused tenant A's entry %s", a.Id())
	}
	if got := countEntries(t, db, tenantA); got != 1 {
		t.Errorf("tenant A entry count: got %d, want 1", got)
	}
	if got := countEntries(t, db, tenantB); got != 1 {
		t.Errorf("tenant B entry count: got %d, want 1", got)
	}
}

// TestCreateWritesExactlyTwoSidesWithItems pins PRD §6: an entry always has
// exactly two sides, with their items attached and their optional asset
// identity preserved.
func TestCreateWritesExactlyTwoSidesWithItems(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)

	assetId := asset.Id(9001)
	referenceId := uint32(55)
	entry := NewBuilder(uuid.New(), testField(t), miniroom.Trade).
		AddSide(100, "Alice", 0, 0, 0, []Item{
			NewItem(2000000, 5, nil, nil),
			NewItem(1302000, 1, &assetId, &referenceId),
		}).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()

	stored, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(stored.Sides()) != 2 {
		t.Fatalf("sides: got %d, want 2", len(stored.Sides()))
	}

	readBack, err := byId(db, tenantId)(stored.Id())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	side, ok := sideFor(readBack, 100)
	if !ok {
		t.Fatalf("read back lost Alice's side")
	}
	if len(side.Items()) != 2 {
		t.Fatalf("Alice's items: got %d, want 2", len(side.Items()))
	}

	byItemId := map[uint32]Item{}
	for _, i := range side.Items() {
		byItemId[uint32(i.ItemId())] = i
	}
	plain := byItemId[2000000]
	if _, has := plain.AssetId(); has {
		t.Errorf("stackable item must not gain an asset identity")
	}
	if plain.Quantity() != 5 {
		t.Errorf("stackable quantity: got %d, want 5", plain.Quantity())
	}
	equip := byItemId[1302000]
	if gotAsset, has := equip.AssetId(); !has || gotAsset != assetId {
		t.Errorf("equip asset id: got %d (present=%t), want %d", gotAsset, has, assetId)
	}
	if gotRef, has := equip.ReferenceId(); !has || gotRef != referenceId {
		t.Errorf("equip reference id: got %d (present=%t), want %d", gotRef, has, referenceId)
	}
}

// TestCreateRejectsEntryWithoutTwoSides pins PRD §6's exactly-two-sides
// invariant on the write path: a half-assembled entry is a bug in the caller
// and must not reach the ledger.
func TestCreateRejectsEntryWithoutTwoSides(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)

	for name, entry := range map[string]Model{
		"no sides": NewBuilder(uuid.New(), testField(t), miniroom.Trade).Build(),
		"one side": NewBuilder(uuid.New(), testField(t), miniroom.Trade).
			AddSide(100, "Alice", 0, 0, 0, nil).Build(),
		"three sides": NewBuilder(uuid.New(), testField(t), miniroom.Trade).
			AddSide(100, "Alice", 0, 0, 0, nil).
			AddSide(200, "Bob", 0, 0, 0, nil).
			AddSide(300, "Carol", 0, 0, 0, nil).Build(),
	} {
		if _, err := create(db, tenantId)(entry); !errors.Is(err, ErrSideCount) {
			t.Errorf("%s: got error %v, want ErrSideCount", name, err)
		}
	}
	if got := countEntries(t, db, tenantId); got != 0 {
		t.Errorf("entry count after rejected writes: got %d, want 0", got)
	}
}

// TestCreatePersistsFieldRoomTypeAndSettledAt pins FR-7.1's record contents:
// where the trade settled, which room type it used, and when.
func TestCreatePersistsFieldRoomTypeAndSettledAt(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	settledAt := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	entry := NewBuilder(uuid.New(), testField(t), miniroom.CashTrade).
		SetSettledAt(settledAt).
		AddSide(100, "Alice", 0, 0, 0, nil).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()

	stored, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	readBack, err := byId(db, tenantId)(stored.Id())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if readBack.RoomType() != miniroom.CashTrade {
		t.Errorf("room type: got %d, want %d", readBack.RoomType(), miniroom.CashTrade)
	}
	if readBack.Field().WorldId() != 1 || readBack.Field().ChannelId() != 2 || readBack.Field().MapId() != 100000000 {
		t.Errorf("field: got %d/%d/%d, want 1/2/100000000", readBack.Field().WorldId(), readBack.Field().ChannelId(), readBack.Field().MapId())
	}
	if !readBack.SettledAt().UTC().Equal(settledAt) {
		t.Errorf("settled at: got %s, want %s", readBack.SettledAt().UTC(), settledAt)
	}
	if readBack.TenantId() != tenantId {
		t.Errorf("tenant id: got %s, want %s", readBack.TenantId(), tenantId)
	}
}

// TestCreateIsAtomicAcrossTheThreeTables pins the one-transaction requirement:
// when the item insert fails, the entry and side rows it was written with must
// not survive, so the ledger never holds a trade with a missing item.
func TestCreateIsAtomicAcrossTheThreeTables(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)

	if err := db.Exec(`
		CREATE TRIGGER fail_item_insert BEFORE INSERT ON trade_ledger_items
		BEGIN SELECT RAISE(ABORT, 'forced failure for atomicity test'); END;
	`).Error; err != nil {
		t.Fatalf("install trigger: %v", err)
	}

	entry := NewBuilder(uuid.New(), testField(t), miniroom.Trade).
		AddSide(100, "Alice", 0, 0, 0, []Item{NewItem(2000000, 1, nil, nil)}).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()

	if _, err := create(db, tenantId)(entry); err == nil {
		t.Fatalf("create must fail when the item insert is rejected")
	}
	if got := countEntries(t, db, tenantId); got != 0 {
		t.Errorf("entry rows after rollback: got %d, want 0", got)
	}
	var sides int64
	if err := db.Model(&Side{}).Where("tenant_id = ?", tenantId).Count(&sides).Error; err != nil {
		t.Fatalf("count sides: %v", err)
	}
	if sides != 0 {
		t.Errorf("side rows after rollback: got %d, want 0", sides)
	}
}

// TestIsDuplicateTransactionClassifiesEachDriver pins the classifier the racing
// branch of create depends on. It is the one piece of create's idempotency that
// the sqlite tests cannot reach through create itself — a real race is needed
// to get past the in-transaction pre-read — and misclassifying PostgreSQL's
// 23505 in production would turn a retried settlement into a hard failure.
func TestIsDuplicateTransactionClassifiesEachDriver(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"nil":                       {nil, false},
		"postgres unique_violation": {&pgconn.PgError{Code: "23505"}, true},
		"postgres wrapped":          {fmt.Errorf("create: %w", &pgconn.PgError{Code: "23505"}), true},
		"postgres other sqlstate":   {&pgconn.PgError{Code: "23503"}, false},
		"gorm translated":           {gorm.ErrDuplicatedKey, true},
		"sqlite":                    {errors.New("UNIQUE constraint failed: trade_ledger_entries.tenant_id, trade_ledger_entries.transaction_id"), true},
		"unrelated":                 {gorm.ErrRecordNotFound, false},
	} {
		if got := isDuplicateTransaction(tc.err); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}

// sideFor returns the entry's side for the given character id.
func sideFor(m Model, characterId uint32) (SideModel, bool) {
	for _, s := range m.Sides() {
		if uint32(s.CharacterId()) == characterId {
			return s, true
		}
	}
	return SideModel{}, false
}
