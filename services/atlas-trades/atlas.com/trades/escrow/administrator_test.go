package escrow

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testDb(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration)
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name())), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return te
}

// testItem builds a staged-item model with the given identity. The snapshot is
// deliberately non-zero in EVERY field group — equip stats, cash, expiry and the
// pet block — so a round trip that dropped a column shows up. The cash/expiry/pet
// groups are the ones the original column set omitted entirely.
func testItem(roomId uuid.UUID, ownerId character.Id, tradeSlot byte) ItemModel {
	return NewItemBuilder(uuid.New(), roomId, ownerId).
		SetTradeSlot(tradeSlot).
		SetSource(inventory.TypeValueEquip, asset.Id(55)).
		SetSnapshot(testSnapshot()).
		Build()
}

// testSnapshot is the reference asset every escrow round-trip test asserts on.
func testSnapshot() sharedsaga.AssetSnapshot {
	return sharedsaga.AssetSnapshot{
		Slot:           3,
		TemplateId:     1302000,
		Expiration:     testExpiration,
		CashId:         987654321,
		Quantity:       1,
		Flag:           2,
		Owner:          "Chronicle",
		Rechargeable:   4200,
		Strength:       11,
		Dexterity:      12,
		Intelligence:   13,
		Luck:           14,
		Hp:             15,
		Mp:             16,
		WeaponAttack:   17,
		MagicAttack:    18,
		WeaponDefense:  19,
		MagicDefense:   20,
		Accuracy:       21,
		Avoidability:   22,
		Hands:          23,
		Speed:          24,
		Jump:           25,
		Slots:          7,
		LevelType:      1,
		Level:          5,
		Experience:     1234,
		HammersApplied: 2,
		PetId:          909,
		PetName:        "Fluffy",
		PetLevel:       3,
		Closeness:      450,
		Fullness:       88,
	}
}

// testExpiration is UTC and truncated to the second: Postgres timestamps do not
// carry a monotonic clock reading, so an untruncated time.Now() would fail the
// round-trip comparison on sub-microsecond drift rather than on a real bug.
var testExpiration = time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC)

// TestCreateItemIsTenantScoped pins that an escrow row written under one tenant
// is invisible to another. Escrow rows name an owner and are returned in bulk by
// the startup reconciler, so a leak here would hand one tenant's item to a
// character in another.
func TestCreateItemIsTenantScoped(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)
	other, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	got, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("own tenant: expected 1 row, got %d", len(got))
	}
	// Whole-snapshot equality, not a spot check on two fields. The defect this
	// replaced was a field that was never carried at all, which a spot check on
	// weaponAttack and owner passed straight through.
	want := testSnapshot()
	gs := got[0].Snapshot()
	gs.Expiration = gs.Expiration.UTC()
	if gs != want {
		t.Errorf("snapshot round trip:\n got %+v\nwant %+v", gs, want)
	}

	foreign, err := ItemsByRoom(db, other.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom(other): %v", err)
	}
	if len(foreign) != 0 {
		t.Fatalf("foreign tenant: expected 0 rows, got %d", len(foreign))
	}
}

// TestDeleteItemIsIdempotent pins that a repeated release succeeds. The unwind
// retries (design §5A.8) and a settlement can be redelivered, so a second delete
// affecting zero rows must NOT be an error — treating it as one would fail a
// saga step whose effect had already landed.
func TestDeleteItemIsIdempotent(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("first DeleteItem: %v", err)
	}
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("second DeleteItem must be a no-op, got: %v", err)
	}

	got, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected the row to be soft-deleted, got %d rows", len(got))
	}
}

// TestRestoreItemUndoesADelete pins the compensating inverse: a release that
// must be rolled back has to bring the row back, or settlement compensation
// loses the item entirely.
func TestRestoreItemUndoesADelete(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if err := RestoreItem(db, te.Id())(m.Id(), uuid.New()); err != nil {
		t.Fatalf("RestoreItem: %v", err)
	}

	got, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the restored row back, got %d rows", len(got))
	}
}

// TestRemoveItemHardDeletes pins the other inverse: a spurious accept must leave
// NO row, not a soft-deleted one, because a soft-deleted row can be restored and
// would resurrect an item the owner already holds again.
func TestRemoveItemHardDeletes(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err := RemoveItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if err := RestoreItem(db, te.Id())(m.Id(), uuid.New()); err != nil {
		t.Fatalf("RestoreItem after a hard delete must be a no-op, got: %v", err)
	}

	got, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a hard-deleted row must not be restorable, got %d rows", len(got))
	}
}

// TestUpsertMesoReplacesRatherThanAccumulates is the load-bearing meso test.
//
// Clientbound mode 16 is an ASSIGNMENT (design §1.6), so atlas-trades tracks the
// ABSOLUTE escrowed total and debits deltas. If the row accumulated instead of
// replacing, a player who staged 1,000 and then re-staged 1,500 would be
// recorded as holding 2,500 in escrow and refunded that on cancel — minting
// 1,000 mesos out of nothing.
func TestUpsertMesoReplacesRatherThanAccumulates(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)

	if err := UpsertMeso(db, te)(roomId, ownerId, 1_000); err != nil {
		t.Fatalf("first UpsertMeso: %v", err)
	}
	if err := UpsertMeso(db, te)(roomId, ownerId, 1_500); err != nil {
		t.Fatalf("second UpsertMeso: %v", err)
	}

	got, err := MesosByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("MesosByRoom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 meso row, got %d", len(got))
	}
	if got[0].Amount() != 1_500 {
		t.Errorf("expected the row to hold the ABSOLUTE total 1500, got %d", got[0].Amount())
	}
}

// TestOrphanedProviderReadsAcrossTenants pins the recovery shape (design §5A.9).
//
// Startup reconciliation runs with NO tenant in context, so it must read every
// row regardless of tenant and rebuild each row's tenant from its stored quad.
// This is the test that fails loudly if the tenant region/major/minor columns
// are dropped from the entity.
func TestOrphanedProviderReadsAcrossTenants(t *testing.T) {
	db := testDb(t)

	teA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	teB, err := tenant.Create(uuid.New(), "JMS", 185, 2)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	if err := CreateItem(db, teA)(testItem(uuid.New(), character.Id(100), 1)); err != nil {
		t.Fatalf("CreateItem(A): %v", err)
	}
	if err := CreateItem(db, teB)(testItem(uuid.New(), character.Id(200), 1)); err != nil {
		t.Fatalf("CreateItem(B): %v", err)
	}

	items, err := AllItems(db)
	if err != nil {
		t.Fatalf("AllItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows across tenants, got %d", len(items))
	}

	seen := map[string]uint16{}
	for _, i := range items {
		te, terr := i.Tenant()
		if terr != nil {
			t.Fatalf("row could not rebuild its tenant: %v", terr)
		}
		seen[te.Region()] = te.MajorVersion()
	}
	if seen["GMS"] != 83 {
		t.Errorf("GMS row lost its version: got %d", seen["GMS"])
	}
	if seen["JMS"] != 185 {
		t.Errorf("JMS row lost its version: got %d", seen["JMS"])
	}
}

// TestAllMesosReadsAcrossTenants is the meso half of the recovery read. Escrowed
// meso is a real debit against the character, so a reconciler that could not see
// these rows would leave players permanently short.
func TestAllMesosReadsAcrossTenants(t *testing.T) {
	db := testDb(t)

	teA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	teB, err := tenant.Create(uuid.New(), "JMS", 185, 2)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	if err := UpsertMeso(db, teA)(uuid.New(), character.Id(100), 5_000); err != nil {
		t.Fatalf("UpsertMeso(A): %v", err)
	}
	if err := UpsertMeso(db, teB)(uuid.New(), character.Id(200), 7_000); err != nil {
		t.Fatalf("UpsertMeso(B): %v", err)
	}

	mesos, err := AllMesos(db)
	if err != nil {
		t.Fatalf("AllMesos: %v", err)
	}
	if len(mesos) != 2 {
		t.Fatalf("expected 2 meso rows across tenants, got %d", len(mesos))
	}
	for _, m := range mesos {
		if _, terr := m.Tenant(); terr != nil {
			t.Fatalf("meso row could not rebuild its tenant: %v", terr)
		}
	}
}

// TestArmThenCommitMesoStake pins the happy path: a committed stake adds its
// delta to Amount and retires its own row, so nothing is left for a later
// delivery or orphan sweep to act on twice.
func TestArmThenCommitMesoStake(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 1_000, 1_000); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	ok, err := CommitMesoStake(db, te.Id())(roomId, ownerId, stakeId)
	if err != nil {
		t.Fatalf("CommitMesoStake: %v", err)
	}
	if !ok {
		t.Fatalf("expected CommitMesoStake to match the armed stake")
	}

	got, found, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if !found {
		t.Fatalf("expected a meso row to exist")
	}
	if got != 1_000 {
		t.Errorf("expected the committed amount 1000, got %d", got)
	}

	mesos, err := MesosByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("MesosByRoom: %v", err)
	}
	if len(mesos) != 1 {
		t.Fatalf("expected exactly 1 meso row, got %d", len(mesos))
	}
	// The stake row goes in the same statement that claimed it. One left behind
	// would be refunded by a later orphan sweep against a debit that already
	// committed.
	stakes, err := MesoStakesByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoStakesByOwner: %v", err)
	}
	if len(stakes) != 0 {
		t.Errorf("expected the committed stake's row to be gone, got %d outstanding", len(stakes))
	}
}

// TestCommitMesoStakeMismatchedIdIsNoOp pins the compare-and-set contract: a
// stakeId matching no outstanding stake must not commit anything. This is what
// protects a misdelivered terminal status from applying a debit no stake ever
// submitted.
func TestCommitMesoStakeMismatchedIdIsNoOp(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()
	otherStakeId := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 1_000, 1_000); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	ok, err := CommitMesoStake(db, te.Id())(roomId, ownerId, otherStakeId)
	if err != nil {
		t.Fatalf("CommitMesoStake: %v", err)
	}
	if ok {
		t.Fatalf("expected CommitMesoStake with a mismatched stakeId to report false")
	}

	got, found, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if !found {
		t.Fatalf("expected a meso row to exist")
	}
	if got != 0 {
		t.Errorf("expected Amount untouched at 0, got %d", got)
	}
}

// TestCommitMesoStakeTwiceOnlyAppliesOnce pins that a redelivered terminal
// status cannot double-apply. Kafka delivery is at-least-once, and the claim
// deletes the stake row in the very transaction that adds its delta, so the
// second delivery finds nothing left to claim.
func TestCommitMesoStakeTwiceOnlyAppliesOnce(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 1_000, 1_000); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	first, err := CommitMesoStake(db, te.Id())(roomId, ownerId, stakeId)
	if err != nil {
		t.Fatalf("first CommitMesoStake: %v", err)
	}
	if !first {
		t.Fatalf("expected the first CommitMesoStake to match")
	}

	second, err := CommitMesoStake(db, te.Id())(roomId, ownerId, stakeId)
	if err != nil {
		t.Fatalf("second CommitMesoStake: %v", err)
	}
	if second {
		t.Fatalf("expected the second CommitMesoStake (redelivery) to report false")
	}

	got, _, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if got != 1_000 {
		t.Errorf("expected the committed amount still 1000 (no double-apply), got %d", got)
	}
}

// TestArmMesoStakeKeepsPriorStakeOutstanding pins the "player retyped the box"
// case. A second arm must leave the first stake ALONE.
//
// This replaces a test that asserted the opposite — that the second arm
// overwrote the first and the first's terminal status was then inert. That was
// the defect, pinned as a contract: the first stake's debit had already left
// the player's pocket, so discarding its status discarded real meso. Reading
// the old expectation as evidence is exactly how the bug survived review.
func TestArmMesoStakeKeepsPriorStakeOutstanding(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	firstStakeId := uuid.New()
	secondStakeId := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, firstStakeId, 1_000, 1_000); err != nil {
		t.Fatalf("first ArmMesoStake: %v", err)
	}
	if err := ArmMesoStake(db, te)(roomId, ownerId, secondStakeId, 1_500, 500); err != nil {
		t.Fatalf("second ArmMesoStake: %v", err)
	}

	stakes, err := MesoStakesByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoStakesByOwner: %v", err)
	}
	if len(stakes) != 2 {
		t.Fatalf("expected both stakes outstanding, got %d", len(stakes))
	}

	// Committed plus in flight is what the player actually typed.
	eff, err := EffectiveMesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("EffectiveMesoByOwner: %v", err)
	}
	if eff != 1_500 {
		t.Errorf("expected the effective stake to be the 1500 last typed, got %d", eff)
	}
}

// TestAbandonMesoStakeClearsWithoutCommitting pins the failure-path inverse of
// commit: the stake row goes but Amount is untouched, because an abandoned
// stake's debit is unwound by the saga's own compensator, not by this row.
func TestAbandonMesoStakeClearsWithoutCommitting(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()

	// Arm on top of a row that already carries a committed amount, to prove
	// abandon leaves that committed amount alone.
	if err := UpsertMeso(db, te)(roomId, ownerId, 500); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}
	// 1000 staked against 500 already escrowed: the saga moves 500.
	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 1_000, 500); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	ok, err := AbandonMesoStake(db, te.Id())(roomId, ownerId, stakeId)
	if err != nil {
		t.Fatalf("AbandonMesoStake: %v", err)
	}
	if !ok {
		t.Fatalf("expected AbandonMesoStake to match the armed stake")
	}

	got, _, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if got != 500 {
		t.Errorf("expected the pre-existing committed amount 500 untouched, got %d", got)
	}

	mesos, err := MesosByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("MesosByRoom: %v", err)
	}
	if len(mesos) != 1 {
		t.Fatalf("expected exactly 1 meso row, got %d", len(mesos))
	}
	stakes, err := MesoStakesByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoStakesByOwner: %v", err)
	}
	if len(stakes) != 0 {
		t.Errorf("expected the abandoned stake's row to be gone, got %d outstanding", len(stakes))
	}
}

// TestMesoStakeById pins the un-tenant-scoped lookup that lets a terminal
// status resolve a stake without knowing which room — or tenant — it belongs
// to in advance.
func TestMesoStakeById(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 1_000, 1_000); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	got, found, err := MesoStakeById(db)(stakeId)
	if err != nil {
		t.Fatalf("MesoStakeById: %v", err)
	}
	if !found {
		t.Fatalf("expected the armed stake to be found")
	}
	if got.RoomId() != roomId {
		t.Errorf("expected RoomId %s, got %s", roomId, got.RoomId())
	}
	if got.OwnerId() != ownerId {
		t.Errorf("expected OwnerId %d, got %d", ownerId, got.OwnerId())
	}
	if got.Amount() != 1_000 {
		t.Errorf("expected the stake's Amount 1000, got %d", got.Amount())
	}
	// The armed delta rides the same lookup: it is what an orphaned stake is
	// refunded by, and nothing else can reproduce it afterwards.
	if got.Delta() != 1_000 {
		t.Errorf("expected Delta 1000, got %d", got.Delta())
	}

	if _, found, err := MesoStakeById(db)(uuid.New()); err != nil {
		t.Fatalf("MesoStakeById(unknown): %v", err)
	} else if found {
		t.Fatalf("expected an unknown stakeId to report not found, not an error")
	}
}

// TestDeleteMesoIsIdempotent mirrors the item case: the unwind retries.
func TestDeleteMesoIsIdempotent(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	if err := UpsertMeso(db, te)(roomId, ownerId, 1_000); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}
	if err := DeleteMeso(db, te.Id())(roomId, ownerId); err != nil {
		t.Fatalf("first DeleteMeso: %v", err)
	}
	if err := DeleteMeso(db, te.Id())(roomId, ownerId); err != nil {
		t.Fatalf("second DeleteMeso must be a no-op, got: %v", err)
	}

	got, err := MesosByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("MesosByRoom: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 meso rows, got %d", len(got))
	}
}

// TestDeleteResolvedMesoRemovesAFullyResolvedRow pins the housekeeping half of
// the conditional delete. A row at zero with no stake in flight records no
// custody at all: nothing will read it again, and AllMesos — which every boot
// sweep runs unfiltered over the whole table — would pay for it forever.
func TestDeleteResolvedMesoRemovesAFullyResolvedRow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	// The state a refunded room leaves behind: the unwind returned the meso and
	// zeroed the total.
	if err := UpsertMeso(db, te)(roomId, ownerId, 0); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}

	deleted, err := DeleteResolvedMeso(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("DeleteResolvedMeso: %v", err)
	}
	if !deleted {
		t.Fatal("a fully-resolved row was not deleted; every later boot sweep re-reads it")
	}

	rows, err := AllMesos(db)
	if err != nil {
		t.Fatalf("AllMesos: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("meso rows after the delete: got %d, want 0", len(rows))
	}

	// The retry is a no-op rather than an error: the callers run inside a
	// transaction that can be replayed by a Kafka redelivery.
	again, err := DeleteResolvedMeso(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("second DeleteResolvedMeso: %v", err)
	}
	if again {
		t.Error("the second delete claimed to have removed a row that was already gone")
	}
}

// TestDeleteResolvedMesoRefusesARowWithAPendingStake is the regression guard the
// delete exists under. A zeroed row whose stake is still in flight is NOT
// resolved: the terminal status resolves against it by pending_stake_id alone,
// and removing it strands a debit the player has already been charged.
func TestDeleteResolvedMesoRefusesARowWithAPendingStake(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()
	// A teardown zeroed the committed total and deliberately left the stake armed.
	if err := ArmMesoStake(db, te)(roomId, ownerId, stakeId, 900, -4_100); err != nil {
		t.Fatalf("ArmMesoStake: %v", err)
	}

	deleted, err := DeleteResolvedMeso(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("DeleteResolvedMeso: %v", err)
	}
	if deleted {
		t.Fatal("a row with a stake in flight was deleted; the stake's terminal status has nothing left to resolve against")
	}
	if _, found, err := MesoStakeById(db)(stakeId); err != nil {
		t.Fatalf("MesoStakeById: %v", err)
	} else if !found {
		t.Fatal("the armed stake is no longer findable by its id")
	}
}

// TestDeleteResolvedMesoRefusesARowStillHoldingEscrow pins the other half of the
// predicate. A non-zero total is meso the service is still holding for its owner,
// and deleting the row would destroy the only record it can be refunded from.
func TestDeleteResolvedMesoRefusesARowStillHoldingEscrow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	if err := UpsertMeso(db, te)(roomId, ownerId, 5_000); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}

	deleted, err := DeleteResolvedMeso(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("DeleteResolvedMeso: %v", err)
	}
	if deleted {
		t.Fatal("a row still holding 5000 escrowed meso was deleted; nothing is left to refund the owner from")
	}
	got, found := mesoOf(t, db, te, roomId, ownerId)
	if !found || got.Amount() != 5_000 {
		t.Fatalf("row after the refused delete: found=%v amount=%d, want 5000", found, got.Amount())
	}
}

// TestDeleteResolvedMesoIsTenantScoped pins that one tenant cannot retire
// another's row. Escrow rows are swept per-tenant at boot, so a cross-tenant
// delete would destroy custody in a database nobody is left to refund from.
func TestDeleteResolvedMesoIsTenantScoped(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)
	other, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	roomId := uuid.New()
	ownerId := character.Id(100)
	if err := UpsertMeso(db, te)(roomId, ownerId, 0); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}

	deleted, err := DeleteResolvedMeso(db, other.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("foreign DeleteResolvedMeso: %v", err)
	}
	if deleted {
		t.Fatal("a foreign tenant deleted the row")
	}
	if _, found := mesoOf(t, db, te, roomId, ownerId); !found {
		t.Fatal("the owning tenant's row is gone")
	}
}

// TestArmingAStakeConcurrentlyWithADeleteKeepsTheStakesRow pins why the two
// conditions live in the DELETE's WHERE clause rather than in a read the caller
// made first.
//
// The two callers are real and independent: a teardown retiring a row it has just
// refunded, and a stage arming a fresh stake against the same (room, owner). A
// read-then-delete lets the teardown observe "resolved", the stage arm, and the
// teardown then delete the row the stage is relying on — the player is debited
// with nothing durable left to resolve or refund the debit.
//
// The invariant is stated as the thing that must never happen: whichever order
// the two land in, a stake that was armed is still findable by its id. If the
// delete wins the row goes and ArmMesoStake — an upsert — re-creates it; if the
// arm wins the delete matches nothing.
//
// The test database is single-connection sqlite, so what is exercised is the
// predicate rather than Postgres row locking. That is the same thing production
// relies on: the decision is in the statement, not in an earlier read.
func TestArmingAStakeConcurrentlyWithADeleteKeepsTheStakesRow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	stakeId := uuid.New()
	if err := UpsertMeso(db, te)(roomId, ownerId, 0); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}

	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	var deleteErr, armErr error
	done.Add(2)
	go func() {
		defer done.Done()
		start.Wait()
		_, deleteErr = DeleteResolvedMeso(db, te.Id())(roomId, ownerId)
	}()
	go func() {
		defer done.Done()
		start.Wait()
		armErr = ArmMesoStake(db, te)(roomId, ownerId, stakeId, 900, 900)
	}()
	start.Done()
	done.Wait()

	if deleteErr != nil {
		t.Fatalf("DeleteResolvedMeso: %v", deleteErr)
	}
	if armErr != nil {
		t.Fatalf("ArmMesoStake: %v", armErr)
	}

	got, found, err := MesoStakeById(db)(stakeId)
	if err != nil {
		t.Fatalf("MesoStakeById: %v", err)
	}
	if !found {
		t.Fatal("the armed stake lost its row to a concurrent delete; its terminal status has nothing to resolve against and the player's debit is stranded")
	}
	if got.Amount() != 900 || got.Delta() != 900 {
		t.Errorf("surviving stake: got amount %d delta %d, want 900/900", got.Amount(), got.Delta())
	}
}

// mesoOf reads back one participant's escrow meso row.
func mesoOf(t *testing.T, db *gorm.DB, te tenant.Model, roomId uuid.UUID, ownerId character.Id) (MesoModel, bool) {
	t.Helper()
	rows, err := MesosByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("MesosByRoom: %v", err)
	}
	for _, r := range rows {
		if r.OwnerId() == ownerId {
			return r, true
		}
	}
	return MesoModel{}, false
}

// TestClaimItemForReturnIsWonByExactlyOneCaller pins the compare-and-set that
// stops one escrow row being returned twice.
//
// The two callers are real and independent: a room teardown returns everything
// ItemsByRoom yields for the room, and an orphaned stage's terminal status
// returns the single row ItemById yields for it. Nothing downstream can tell the
// two submissions apart — each unwind mints its own transaction id,
// accept_to_character grants unconditionally, and DeleteItem treats the second
// release as success — so the item is granted twice unless exactly one of them
// is allowed to submit.
//
// The two claims run concurrently. The test database is single-connection
// sqlite, so what actually gets exercised is the CAS predicate rather than
// Postgres row locking: the loser is the caller whose UPDATE finds the column
// already stamped and therefore matches no row. That is the same thing the
// production statement relies on — the decision lives in the WHERE clause, not
// in a read the caller made earlier.
func TestClaimItemForReturnIsWonByExactlyOneCaller(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	m := testItem(uuid.New(), character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	results := make([]bool, 2)
	errs := make([]error, 2)
	for i := range results {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			results[i], errs[i] = ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
		}(i)
	}
	start.Done()
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
	}
	winners := 0
	for _, won := range results {
		if won {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("claims that won: got %d, want exactly 1 — %d submissions of trade_unwind means the owner receives the item %d times", winners, winners, winners)
	}
}

// TestClaimItemForReturnSurvivesTheProcess pins that the claim is a COLUMN and
// not in-memory state. The boot sweep re-reads every surviving row through
// AllItems, and a row whose unwind is already in flight must lose the claim
// there too — a fresh handle onto the same database stands in for the restart.
func TestClaimItemForReturnSurvivesTheProcess(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	m := testItem(uuid.New(), character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	won, err := ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !won {
		t.Fatal("the first claim on an unclaimed row must win")
	}

	rows, err := AllItems(db)
	if err != nil {
		t.Fatalf("AllItems: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("AllItems: got %d rows, want the still-live claimed row", len(rows))
	}
	again, err := ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if again {
		t.Fatal("a claimed row was claimed a second time; a boot sweep would return an item whose unwind is already in flight")
	}
}

// TestClaimItemForReturnIsTenantScoped pins that one tenant cannot latch
// another's row. The claim is the gate on returning an asset, so a cross-tenant
// claim would silently strand a character's item in a database they cannot be
// swept from.
func TestClaimItemForReturnIsTenantScoped(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)
	other, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	m := testItem(uuid.New(), character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	won, err := ClaimItemForReturn(db, other.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("foreign claim: %v", err)
	}
	if won {
		t.Fatal("a foreign tenant claimed the row")
	}
	won, err = ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("owning claim: %v", err)
	}
	if !won {
		t.Fatal("the owning tenant lost a claim no one else could have taken")
	}
}

// TestClaimItemForReturnRefusesAReleasedRow pins that a row whose item has
// already LEFT custody cannot be claimed. There is nothing left to return, and
// letting a caller latch it would only make the eventual compensating restore
// look like a row somebody is already returning.
func TestClaimItemForReturnRefusesAReleasedRow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	m := testItem(uuid.New(), character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	won, err := ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("ClaimItemForReturn: %v", err)
	}
	if won {
		t.Fatal("a released row was claimed for return; its item is no longer in custody")
	}
}

// TestRestoreItemReleasesTheReturnClaim pins the compensation path on a row that
// WAS claimed and whose unwind then failed — the reachable combination, not a
// hypothetical one: the claiming teardown submits release_from_trade followed by
// accept_to_character, and an accept that fails sends the orchestrator's reverse
// walk back through RestoreTradeEscrow (saga/compensator.go
// DispatchTradeTransactionRollbacks).
//
// The restore must do two things. It must bring the row back, or the item is
// lost outright. And it must UNLATCH it, because the return demonstrably did not
// happen: a row left latched is skipped by every later sweep, which is the one
// retry this case has.
func TestRestoreItemReleasesTheReturnClaim(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	// The unwind's saga id. The claim and the restore carry the SAME value in
	// the real flow, because it is that unwind's own reverse walk asking — which
	// is exactly what tells this case apart from a stale restore sent by a
	// different saga (see RestoreItem).
	unwindTxId := uuid.New()
	won, err := ClaimItemForReturn(db, te.Id())(m.Id(), unwindTxId)
	if err != nil {
		t.Fatalf("ClaimItemForReturn: %v", err)
	}
	if !won {
		t.Fatal("the first claim on an unclaimed row must win")
	}
	// release_from_trade completed; accept_to_character then failed.
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if err := RestoreItem(db, te.Id())(m.Id(), unwindTxId); err != nil {
		t.Fatalf("RestoreItem: %v", err)
	}

	got, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the restored row back, got %d rows", len(got))
	}
	reclaimed, err := ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New())
	if err != nil {
		t.Fatalf("re-claim: %v", err)
	}
	if !reclaimed {
		t.Fatal("a restored row is still latched; every later sweep would skip it and the item would be stranded with no error anywhere")
	}
}

// TestConcurrentMesoStakesConserve pins the invariant that makes the meso
// column safe: the committed Amount must always equal the sum of the deltas
// that award_mesos ACTUALLY MOVED, no more and no less.
//
// Two stakes are outstanding at once because the client permits it — PutMoney
// arms CWvsContext's excl latch and the debit's own STAT_CHANGED clears it, so
// a player who retypes the box faster than a saga round trip submits a second
// stake while the first is still in flight. Each stake's delta is netted
// against committed PLUS what is already in flight, so together they move
// exactly the absolute total the player last typed.
func TestConcurrentMesoStakesConserve(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	first := uuid.New()
	second := uuid.New()

	// Typed 1000: nothing committed, nothing in flight, so the saga moves 1000.
	if err := ArmMesoStake(db, te)(roomId, ownerId, first, 1_000, 1_000); err != nil {
		t.Fatalf("first ArmMesoStake: %v", err)
	}
	// Retyped 1500 before the first resolved: 1000 is already in flight, so
	// this saga moves only the additional 500.
	if err := ArmMesoStake(db, te)(roomId, ownerId, second, 1_500, 500); err != nil {
		t.Fatalf("second ArmMesoStake: %v", err)
	}

	// Both debits landed, so both must be reflected. The first is NOT stale:
	// its 1000 left the player's pocket.
	ok, err := CommitMesoStake(db, te.Id())(roomId, ownerId, first)
	if err != nil {
		t.Fatalf("CommitMesoStake(first): %v", err)
	}
	if !ok {
		t.Fatalf("the first stake's debit of 1000 already moved real meso; its commit must be honoured, not dropped as superseded")
	}
	ok, err = CommitMesoStake(db, te.Id())(roomId, ownerId, second)
	if err != nil {
		t.Fatalf("CommitMesoStake(second): %v", err)
	}
	if !ok {
		t.Fatalf("expected the second stake's commit to match")
	}

	got, _, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	// 1000 + 500 debited, so 1500 escrowed.
	if got != 1_500 {
		t.Errorf("player was debited 1500 across two stakes; escrow holds %d — %d meso destroyed", got, 1_500-int64(got))
	}
}

// TestSupersededMesoStakeFailureDoesNotMint is the inverse, and the case that
// makes independent per-stake resolution mandatory rather than cosmetic.
//
// The earlier stake FAILS after a later one was armed. Its debit is unwound by
// its own saga compensator, so its delta never moved and must never reach
// Amount. Committing the later stake's absolute total on top would credit the
// player escrow nobody paid for.
func TestSupersededMesoStakeFailureDoesNotMint(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	first := uuid.New()
	second := uuid.New()

	if err := ArmMesoStake(db, te)(roomId, ownerId, first, 1_000, 1_000); err != nil {
		t.Fatalf("first ArmMesoStake: %v", err)
	}
	if err := ArmMesoStake(db, te)(roomId, ownerId, second, 1_500, 500); err != nil {
		t.Fatalf("second ArmMesoStake: %v", err)
	}

	// The first stake's saga failed and compensated: 1000 went back.
	ok, err := AbandonMesoStake(db, te.Id())(roomId, ownerId, first)
	if err != nil {
		t.Fatalf("AbandonMesoStake(first): %v", err)
	}
	if !ok {
		t.Fatalf("the first stake is genuinely outstanding; its abandon must claim it")
	}
	// The second stake's 500 did move.
	if _, err := CommitMesoStake(db, te.Id())(roomId, ownerId, second); err != nil {
		t.Fatalf("CommitMesoStake(second): %v", err)
	}

	got, _, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	// Only the second stake's 500 was ever moved and kept.
	if got != 500 {
		t.Errorf("only 500 was actually debited; escrow holds %d — %d meso minted", got, int64(got)-500)
	}
}

// TestClaimMesoForReturnIsExclusive is the meso twin of
// TestClaimItemForReturn's contract, and it exists because meso had no
// arbitration at all.
//
// Two independent paths can each decide to refund one participant's escrowed
// meso: a room teardown reading MesosByRoom, and the boot/ticker sweep reading
// AllMesos. Both build an unwind from the total they read and only afterwards
// zero the row, so under READ COMMITTED — the isolation this fleet runs at —
// both can read the same 5000 and both submit a refund for it. Nothing
// downstream dedupes them: a meso unwind leg is a bare award_mesos credit.
//
// The claim collapses read-and-take into ONE statement, so the amount is
// handed to exactly one caller.
func TestClaimMesoForReturnIsExclusive(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	ownerId := character.Id(100)
	if err := UpsertMeso(db, te)(roomId, ownerId, 5_000); err != nil {
		t.Fatalf("UpsertMeso: %v", err)
	}

	got, ok, err := ClaimMesoForReturn(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("first ClaimMesoForReturn: %v", err)
	}
	if !ok || got != 5_000 {
		t.Fatalf("first claim: got %d (claimed %v), want the whole 5000", got, ok)
	}

	// The second path arrives and must come away with nothing to refund.
	got, ok, err = ClaimMesoForReturn(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("second ClaimMesoForReturn: %v", err)
	}
	if ok || got != 0 {
		t.Fatalf("second claim took %d (claimed %v); the player would be refunded twice", got, ok)
	}

	// The row itself survives the claim — a stake still in flight resolves
	// against it — but records nothing.
	held, found, err := MesoByOwner(db, te.Id())(roomId, ownerId)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if !found || held != 0 {
		t.Errorf("after the claim the row holds %d (found %v), want 0 and still present", held, found)
	}
}

// TestClaimMesoForReturnIgnoresANonPositiveRow pins that there is nothing to
// claim when nothing is owed. A zero row holds no custody, and a negative one
// means more reduction has been confirmed than increase so far.
func TestClaimMesoForReturnIgnoresANonPositiveRow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	for _, tc := range []struct {
		name    string
		balance int64
		owner   character.Id
	}{
		{"zero", 0, 100},
		{"negative", -500, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := UpsertMeso(db, te)(roomId, tc.owner, tc.balance); err != nil {
				t.Fatalf("UpsertMeso: %v", err)
			}
			got, ok, err := ClaimMesoForReturn(db, te.Id())(roomId, tc.owner)
			if err != nil {
				t.Fatalf("ClaimMesoForReturn: %v", err)
			}
			if ok || got != 0 {
				t.Errorf("claimed %d (%v) from a %s row, want nothing", got, ok, tc.name)
			}
		})
	}
}

// TestReleaseItemReturnClaimsUnlatchesOnlyItsOwnTransaction pins the item half
// of failed-unwind recovery.
//
// A latched row is invisible to everything that could return it: the latch
// clears only on a completed release, and the boot sweep skips a latched row by
// design. So an unwind that fails must hand its rows back, or the item sits
// intact in custody owned by nobody.
//
// Scoped to the transaction, because a row another unwind is legitimately
// returning must keep its claim — releasing those would hand the same item to
// two unwinds and grant it twice.
func TestReleaseItemReturnClaimsUnlatchesOnlyItsOwnTransaction(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	failing := testItem(roomId, character.Id(100), 1)
	healthy := testItem(roomId, character.Id(200), 2)
	for _, m := range []ItemModel{failing, healthy} {
		if err := CreateItem(db, te)(m); err != nil {
			t.Fatalf("CreateItem: %v", err)
		}
	}

	failedTx, otherTx := uuid.New(), uuid.New()
	if ok, err := ClaimItemForReturn(db, te.Id())(failing.Id(), failedTx); err != nil || !ok {
		t.Fatalf("claim failing row: ok=%v err=%v", ok, err)
	}
	if ok, err := ClaimItemForReturn(db, te.Id())(healthy.Id(), otherTx); err != nil || !ok {
		t.Fatalf("claim healthy row: ok=%v err=%v", ok, err)
	}

	released, err := ReleaseItemReturnClaims(db, te.Id())(failedTx)
	if err != nil {
		t.Fatalf("ReleaseItemReturnClaims: %v", err)
	}
	if released != 1 {
		t.Fatalf("released %d rows, want exactly the 1 the failed unwind claimed", released)
	}

	// The released row is claimable again — which is what "recoverable" means:
	// the next teardown or boot sweep can take it.
	if ok, err := ClaimItemForReturn(db, te.Id())(failing.Id(), uuid.New()); err != nil || !ok {
		t.Errorf("the released row could not be re-claimed (ok=%v err=%v); it is stranded in custody with nothing able to return it", ok, err)
	}
	// The other unwind's row is untouched, so its item cannot be granted twice.
	if ok, err := ClaimItemForReturn(db, te.Id())(healthy.Id(), uuid.New()); err != nil || ok {
		t.Errorf("a row claimed by a DIFFERENT unwind was released (ok=%v err=%v); that unwind's item would be granted twice", ok, err)
	}
}

// TestRestoreItemCannotResurrectAReturnedRow fences the compensating restore
// against at-least-once redelivery.
//
// The sequence is reachable and needs no race beyond Kafka's own guarantee:
//
//  1. a settlement fails, and its reverse walk emits RESTORE_TRADE_ESCROW for
//     row X, putting X back into custody;
//  2. the failed settlement's unwind then CLAIMS X, releases it, and the owner
//     is granted the item;
//  3. the restore from step 1 is redelivered.
//
// Unfenced, step 3 un-soft-deletes X and clears its claim, leaving a live,
// unclaimed row for an item the owner is already holding — and the next boot
// sweep hands it over a second time.
//
// The existing doc comment argued this was impossible. Its reasoning covers
// only the ordering WITHIN one reverse walk (an accept that succeeded is never
// followed by a restore of its own release) and says nothing about a redelivery
// arriving after a DIFFERENT saga's release, which is this case.
func TestRestoreItemCannotResurrectAReturnedRow(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	// The unwind claims the row and releases it; the owner now holds the item.
	if ok, err := ClaimItemForReturn(db, te.Id())(m.Id(), uuid.New()); err != nil || !ok {
		t.Fatalf("claim for return: ok=%v err=%v", ok, err)
	}
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	// The stale restore lands.
	if err := RestoreItem(db, te.Id())(m.Id(), uuid.New()); err != nil {
		t.Fatalf("RestoreItem: %v", err)
	}

	rows, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("a redelivered restore resurrected a row whose item was already returned; the next boot sweep grants it again (got %d live rows)", len(rows))
	}
}

// TestRestoreItemStillUndoesAnUnclaimedRelease is the other half, and the
// reason the fence keys on the return claim rather than on the soft delete.
//
// A settlement's release is NOT a claim for return — only the unwind paths
// claim — so a settlement reverse walk restoring its own release must still
// work. Fencing on "was soft-deleted" would have broken exactly that.
func TestRestoreItemStillUndoesAnUnclaimedRelease(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	roomId := uuid.New()
	m := testItem(roomId, character.Id(100), 1)
	if err := CreateItem(db, te)(m); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err := DeleteItem(db, te.Id())(m.Id()); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if err := RestoreItem(db, te.Id())(m.Id(), uuid.New()); err != nil {
		t.Fatalf("RestoreItem: %v", err)
	}

	rows, err := ItemsByRoom(db, te.Id())(roomId)
	if err != nil {
		t.Fatalf("ItemsByRoom: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("a settlement reverse walk could not restore its own release; the item is lost (got %d live rows)", len(rows))
	}
}
