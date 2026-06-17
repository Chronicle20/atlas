package holding_test

import (
	"atlas-mts/holding"
	"atlas-mts/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func adminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, holding.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func buildHolding(t *testing.T, tenantId uuid.UUID, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(tenantId, 0, ownerId).
		SetOrigin(holding.OriginUnsold).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetStrength(10).
		SetDexterity(20).
		SetWeaponAttack(30).
		SetSlots(7).
		SetLevel(5).
		SetItemLevel(3).
		SetItemExp(123).
		SetRingId(456).
		SetViciousCount(2).
		SetFlags(0x10).
		Build()
	if err != nil {
		t.Fatalf("Failed to build holding: %v", err)
	}
	return m
}

// TestAdministratorCreateGetById asserts a created holding round-trips through
// getById and preserves the full snapshot (explicit columns, not a JSON blob).
func TestAdministratorCreateGetById(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	m := buildHolding(t, tenantId, 100)
	created, err := holding.CreateHolding(db, m)
	if err != nil {
		t.Fatalf("CreateHolding: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("CreateHolding did not assign an id")
	}

	got, err := holding.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.TenantId() != tenantId {
		t.Errorf("tenantId = %s, want %s", got.TenantId(), tenantId)
	}
	if got.OwnerId() != 100 {
		t.Errorf("ownerId = %d, want 100", got.OwnerId())
	}
	if got.Origin() != holding.OriginUnsold {
		t.Errorf("origin = %q, want unsold", got.Origin())
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("templateId = %d, want 1302000", got.TemplateId())
	}
	if got.Strength() != 10 || got.Dexterity() != 20 || got.WeaponAttack() != 30 {
		t.Errorf("equip stat block not round-tripped: str=%d dex=%d wAtt=%d", got.Strength(), got.Dexterity(), got.WeaponAttack())
	}
	if got.Slots() != 7 || got.Level() != 5 || got.ItemLevel() != 3 {
		t.Errorf("slots/level not round-tripped: slots=%d level=%d itemLevel=%d", got.Slots(), got.Level(), got.ItemLevel())
	}
	if got.ItemExp() != 123 || got.RingId() != 456 || got.ViciousCount() != 2 || got.Flags() != 0x10 {
		t.Errorf("itemExp/ringId/vicious/flags not round-tripped")
	}
}

// TestAdministratorCrossTenantIsolation asserts tenant B cannot read tenant A's
// holding even within the same world.
func TestAdministratorCrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := adminTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	m := buildHolding(t, tenantA, 100)
	created, err := holding.CreateHolding(dbA, m)
	if err != nil {
		t.Fatalf("CreateHolding tenant A: %v", err)
	}

	if _, err := holding.GetById(created.Id().String())(dbA)(); err != nil {
		t.Fatalf("tenant A GetById own holding: %v", err)
	}

	if _, err := holding.GetById(created.Id().String())(dbB)(); err == nil {
		t.Error("tenant B was able to read tenant A's holding")
	}

	allB, err := holding.GetAll()(dbB)()
	if err != nil {
		t.Fatalf("tenant B GetAll: %v", err)
	}
	if len(allB) != 0 {
		t.Errorf("tenant B GetAll returned %d rows, want 0", len(allB))
	}
}

// TestAdministratorSoftDeleteIdempotent asserts soft-delete-by-id is idempotent:
// the first SoftDelete affects 1 row, a second affects 0 (the row is gone from
// the default scope). The row no longer appears in getByOwner afterward.
func TestAdministratorSoftDeleteIdempotent(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	m := buildHolding(t, tenantId, 100)
	created, err := holding.CreateHolding(db, m)
	if err != nil {
		t.Fatalf("CreateHolding: %v", err)
	}

	affected, err := holding.SoftDelete(db, created.Id().String())
	if err != nil {
		t.Fatalf("SoftDelete first: %v", err)
	}
	if affected != 1 {
		t.Errorf("first SoftDelete affected %d rows, want 1", affected)
	}

	// The soft-deleted row must not be readable.
	if _, err := holding.GetById(created.Id().String())(db)(); err == nil {
		t.Error("soft-deleted holding was still readable")
	}

	affected, err = holding.SoftDelete(db, created.Id().String())
	if err != nil {
		t.Fatalf("SoftDelete second: %v", err)
	}
	if affected != 0 {
		t.Errorf("second SoftDelete affected %d rows, want 0 (idempotent)", affected)
	}
}

// TestAdministratorRestoreUndoesSoftDelete asserts Restore is the inverse of
// SoftDelete (the WithdrawFromMts compensation path): a soft-deleted holding
// becomes readable again after Restore, and Restore is idempotent — restoring an
// already-live row affects 0 rows.
func TestAdministratorRestoreUndoesSoftDelete(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	m := buildHolding(t, tenantId, 100)
	created, err := holding.CreateHolding(db, m)
	if err != nil {
		t.Fatalf("CreateHolding: %v", err)
	}

	if _, err := holding.SoftDelete(db, created.Id().String()); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, err := holding.GetById(created.Id().String())(db)(); err == nil {
		t.Fatal("soft-deleted holding was still readable before restore")
	}

	affected, err := holding.Restore(db, created.Id().String())
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if affected != 1 {
		t.Errorf("Restore affected %d rows, want 1", affected)
	}

	// The restored row must be readable again.
	got, err := holding.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById after Restore: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("restored id = %s, want %s", got.Id(), created.Id())
	}

	// Restore is idempotent: restoring an already-live row affects 0 rows.
	affected, err = holding.Restore(db, created.Id().String())
	if err != nil {
		t.Fatalf("Restore second: %v", err)
	}
	if affected != 0 {
		t.Errorf("second Restore affected %d rows, want 0 (idempotent)", affected)
	}
}

// TestAdministratorMultipleHoldingsPerTenant asserts a single tenant can hold
// many holdings concurrently. Guards against a unique constraint on tenant_id
// alone (which would cap a tenant at one holding). The (tenant_id, id) unique
// index must permit this.
func TestAdministratorMultipleHoldingsPerTenant(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	for i := 0; i < 3; i++ {
		m := buildHolding(t, tenantId, uint32(100+i))
		if _, err := holding.CreateHolding(db, m); err != nil {
			t.Fatalf("CreateHolding #%d for tenant: %v", i, err)
		}
	}

	all, err := holding.GetAll()(db)()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("tenant holds %d holdings, want 3", len(all))
	}
}

// TestAdministratorIndexExists asserts the design index is created by the
// migration.
func TestAdministratorIndexExists(t *testing.T) {
	db := adminTestDB(t)
	mig := db.Migrator()

	if !mig.HasIndex(&holdingIndexProbe{}, "idx_holdings_world_owner") {
		t.Errorf("expected index %q to exist on holdings", "idx_holdings_world_owner")
	}
}

type holdingIndexProbe struct{}

func (holdingIndexProbe) TableName() string { return "holdings" }
