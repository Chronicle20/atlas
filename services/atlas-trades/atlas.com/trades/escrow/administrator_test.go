package escrow

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
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

// testItem builds a staged-item model with the given identity. The stat block
// is deliberately non-zero so a round trip that dropped columns shows up.
func testItem(roomId uuid.UUID, ownerId character.Id, tradeSlot byte) ItemModel {
	return NewItemBuilder(uuid.New(), roomId, ownerId).
		SetTradeSlot(tradeSlot).
		SetSource(inventory.TypeValueEquip, slot.Position(3), asset.Id(55)).
		SetTemplateId(item.Id(1302000)).
		SetQuantity(asset.Quantity(1)).
		SetWeaponAttack(17).
		SetSlots(7).
		SetOwner("Chronicle").
		Build()
}

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
	if got[0].WeaponAttack() != 17 {
		t.Errorf("weaponAttack round trip: expected 17, got %d", got[0].WeaponAttack())
	}
	if got[0].Owner() != "Chronicle" {
		t.Errorf("owner round trip: expected Chronicle, got %q", got[0].Owner())
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
	if err := RestoreItem(db, te.Id())(m.Id()); err != nil {
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
	if err := RestoreItem(db, te.Id())(m.Id()); err != nil {
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
