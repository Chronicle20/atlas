package wish_test

import (
	"atlas-mts/test"
	"atlas-mts/wish"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func adminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, wish.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func buildWish(t *testing.T, tenantId uuid.UUID, characterId uint32, itemId uint32) wish.Model {
	t.Helper()
	m, err := wish.NewBuilder(tenantId, characterId, itemId).Build()
	if err != nil {
		t.Fatalf("Failed to build wish: %v", err)
	}
	return m
}

// TestAdministratorCreateGetById asserts a created wish entry round-trips
// through getById and preserves its fields.
func TestAdministratorCreateGetById(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := wish.CreateWish(db, buildWish(t, tenantId, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("CreateWish did not assign an id")
	}

	got, err := wish.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.TenantId() != tenantId {
		t.Errorf("tenantId = %s, want %s", got.TenantId(), tenantId)
	}
	if got.CharacterId() != 100 {
		t.Errorf("characterId = %d, want 100", got.CharacterId())
	}
	if got.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", got.ItemId())
	}
}

// TestAdministratorCrossTenantIsolation asserts tenant B cannot read tenant A's
// wish entry.
func TestAdministratorCrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := adminTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	created, err := wish.CreateWish(dbA, buildWish(t, tenantA, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish tenant A: %v", err)
	}

	if _, err := wish.GetById(created.Id().String())(dbA)(); err != nil {
		t.Fatalf("tenant A GetById own wish: %v", err)
	}

	if _, err := wish.GetById(created.Id().String())(dbB)(); err == nil {
		t.Error("tenant B was able to read tenant A's wish")
	}

	allB, err := wish.GetAll()(dbB)()
	if err != nil {
		t.Fatalf("tenant B GetAll: %v", err)
	}
	if len(allB) != 0 {
		t.Errorf("tenant B GetAll returned %d rows, want 0", len(allB))
	}
}

// TestAdministratorDeleteWish asserts hard-delete-by-id: the first DeleteWish
// affects 1 row, a second affects 0 (the row is gone).
func TestAdministratorDeleteWish(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := wish.CreateWish(db, buildWish(t, tenantId, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish: %v", err)
	}

	affected, err := wish.DeleteWish(db, created.Id().String())
	if err != nil {
		t.Fatalf("DeleteWish first: %v", err)
	}
	if affected != 1 {
		t.Errorf("first DeleteWish affected %d rows, want 1", affected)
	}

	if _, err := wish.GetById(created.Id().String())(db)(); err == nil {
		t.Error("deleted wish was still readable")
	}

	affected, err = wish.DeleteWish(db, created.Id().String())
	if err != nil {
		t.Fatalf("DeleteWish second: %v", err)
	}
	if affected != 0 {
		t.Errorf("second DeleteWish affected %d rows, want 0", affected)
	}
}

// TestAdministratorDeleteWishMalformedIdIsScoped is the regression guard for the
// GORM zero-value struct-condition elision bug: a malformed (non-UUID) id must
// NOT delete the whole tenant's wishlist. Before the fix, parseId returned
// uuid.Nil for a bad id, GORM elided the zero-valued Id struct condition, and the
// Delete degraded to a tenant-wide wipe (the only surviving predicate being the
// tenant callback's WHERE tenant_id = ?).
func TestAdministratorDeleteWishMalformedIdIsScoped(t *testing.T) {
	for _, badId := range []string{"not-a-uuid", ""} {
		t.Run("id="+badId, func(t *testing.T) {
			tenantId := uuid.New()
			ctx := tenantCtx(t, tenantId)
			db := adminTestDB(t).WithContext(ctx)

			for i := 0; i < 3; i++ {
				m := buildWish(t, tenantId, uint32(100+i), uint32(1302000+i))
				if _, err := wish.CreateWish(db, m); err != nil {
					t.Fatalf("CreateWish #%d: %v", i, err)
				}
			}

			affected, err := wish.DeleteWish(db, badId)
			if err == nil {
				t.Errorf("DeleteWish(%q) returned nil error, want a malformed-id error", badId)
			}
			if affected != 0 {
				t.Errorf("DeleteWish(%q) affected %d rows, want 0", badId, affected)
			}

			all, err := wish.GetAll()(db)()
			if err != nil {
				t.Fatalf("GetAll: %v", err)
			}
			if len(all) != 3 {
				t.Errorf("after DeleteWish(%q) tenant holds %d wishes, want 3 (all survive)", badId, len(all))
			}
		})
	}
}

// TestAdministratorMultipleWishesPerTenant asserts a single tenant can hold many
// wish entries concurrently. Guards against a unique constraint on tenant_id
// alone (which would cap a tenant at one wish). The (tenant_id, id) unique index
// must permit this.
func TestAdministratorMultipleWishesPerTenant(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	for i := 0; i < 3; i++ {
		m := buildWish(t, tenantId, uint32(100+i), uint32(1302000+i))
		if _, err := wish.CreateWish(db, m); err != nil {
			t.Fatalf("CreateWish #%d for tenant: %v", i, err)
		}
	}

	all, err := wish.GetAll()(db)()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("tenant holds %d wishes, want 3", len(all))
	}
}

// TestAdministratorIndexExists asserts the design index is created by the
// migration.
func TestAdministratorIndexExists(t *testing.T) {
	db := adminTestDB(t)
	mig := db.Migrator()

	if !mig.HasIndex(&wishIndexProbe{}, "idx_wish_entries_character") {
		t.Errorf("expected index %q to exist on wish_entries", "idx_wish_entries_character")
	}
}

type wishIndexProbe struct{}

func (wishIndexProbe) TableName() string { return "wish_entries" }
