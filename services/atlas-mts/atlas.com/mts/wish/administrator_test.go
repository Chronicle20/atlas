package wish_test

import (
	"atlas-mts/test"
	"atlas-mts/wish"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// TestAdministratorCreatePreservesListingSerial asserts a "cart" wish persists
// the favorited listing's serial through CreateWish and getById, so the Cart can
// resolve and render exactly that listing (bug 1: the cart must track the
// favorited listing, not re-resolve the item template).
func TestAdministratorCreatePreservesListingSerial(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	const favoritedListingSerial = uint32(4242)
	m, err := wish.NewBuilder(tenantId, 100, 1302000).
		SetType(wish.TypeCart).
		SetListingSerial(favoritedListingSerial).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	created, err := wish.CreateWish(db, m)
	if err != nil {
		t.Fatalf("CreateWish: %v", err)
	}
	if created.ListingSerial() != favoritedListingSerial {
		t.Errorf("created listingSerial = %d, want %d", created.ListingSerial(), favoritedListingSerial)
	}

	got, err := wish.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ListingSerial() != favoritedListingSerial {
		t.Errorf("persisted listingSerial = %d, want %d", got.ListingSerial(), favoritedListingSerial)
	}
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

// TestAdministratorDeleteExpiredWanted asserts the want-ad sweep deletes ONLY
// expired "wanted" entries: an expired wanted (expires_at < now) is removed,
// while a future-dated wanted and a "cart" entry (no expiry) both survive.
func TestAdministratorDeleteExpiredWanted(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	mkWish := func(characterId uint32, itemId uint32, wishType string, exp *time.Time) wish.Model {
		b := wish.NewBuilder(tenantId, characterId, itemId).
			SetWorldId(world.Id(0)).
			SetType(wishType)
		if exp != nil {
			b = b.SetExpiresAt(exp)
		}
		m, err := b.Build()
		if err != nil {
			t.Fatalf("build wish: %v", err)
		}
		return m
	}

	expired, err := wish.CreateWish(db, mkWish(100, 1302000, wish.TypeWanted, &past))
	if err != nil {
		t.Fatalf("CreateWish expired wanted: %v", err)
	}
	futureWanted, err := wish.CreateWish(db, mkWish(101, 1302001, wish.TypeWanted, &future))
	if err != nil {
		t.Fatalf("CreateWish future wanted: %v", err)
	}
	cart, err := wish.CreateWish(db, mkWish(102, 1302002, wish.TypeCart, nil))
	if err != nil {
		t.Fatalf("CreateWish cart: %v", err)
	}

	deleted, err := wish.DeleteExpiredWanted(db, now)
	if err != nil {
		t.Fatalf("DeleteExpiredWanted: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("DeleteExpiredWanted removed %d rows, want 1 (only the expired wanted)", deleted)
	}

	if _, err := wish.GetById(expired.Id().String())(db)(); err == nil {
		t.Error("expired wanted want-ad was not deleted")
	}
	if _, err := wish.GetById(futureWanted.Id().String())(db)(); err != nil {
		t.Errorf("future-dated wanted was deleted: %v", err)
	}
	if _, err := wish.GetById(cart.Id().String())(db)(); err != nil {
		t.Errorf("cart entry (no expiry) was deleted: %v", err)
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

// buildWorldWish builds a world-scoped wish entry (the gameplay create path
// carries a worldId; the serial counter is per-(tenant, world)).
func buildWorldWish(t *testing.T, tenantId uuid.UUID, worldId byte, characterId uint32, itemId uint32) wish.Model {
	t.Helper()
	m, err := wish.NewBuilder(tenantId, characterId, itemId).
		SetWorldId(world.Id(worldId)).
		Build()
	if err != nil {
		t.Fatalf("Failed to build wish: %v", err)
	}
	return m
}

// TestAdministratorCreateAssignsSerialAndRoundTrips is the core CANCEL_WISH-fix
// guard: CreateWish must draw a nonzero per-(tenant, world) serial, and that
// serial must resolve straight back to the wish entry via GetBySerial — the
// exact round-trip the channel performs when the client echoes the wish
// ITCITEM's nITCSN back on CANCEL_WISH (IDA: CITC::OnCancelWish, v83 0x59fb07,
// Encode4 of the item's offset-0x20 nITCSN field). Before the fix the wish item
// carried itcSn=0, so the client always sent 0 and nothing resolved.
func TestAdministratorCreateAssignsSerialAndRoundTrips(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := wish.CreateWish(db, buildWorldWish(t, tenantId, 0, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish: %v", err)
	}
	if created.Serial() == 0 {
		t.Fatal("CreateWish did not assign a nonzero serial")
	}

	got, err := wish.GetBySerial(world.Id(0), created.Serial())(db)()
	if err != nil {
		t.Fatalf("GetBySerial(%d): %v", created.Serial(), err)
	}
	if got.Id() != created.Id() {
		t.Errorf("GetBySerial resolved id %s, want %s", got.Id(), created.Id())
	}
	if got.ItemId() != 1302000 {
		t.Errorf("GetBySerial resolved itemId %d, want 1302000", got.ItemId())
	}
	if got.CharacterId() != 100 {
		t.Errorf("GetBySerial resolved characterId %d, want 100", got.CharacterId())
	}
}

// TestAdministratorSerialIsWorldScoped asserts a serial resolves only within the
// world it was drawn in — the same serial number in a different world must not
// resolve (serials are per-(tenant, world), shared with listings/holdings).
func TestAdministratorSerialIsWorldScoped(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := wish.CreateWish(db, buildWorldWish(t, tenantId, 0, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish: %v", err)
	}

	if _, err := wish.GetBySerial(world.Id(1), created.Serial())(db)(); err == nil {
		t.Errorf("GetBySerial in world 1 resolved a world-0 serial [%d]", created.Serial())
	}
}

// TestAdministratorCreateIsIdempotent asserts the "one wish per (tenant, world,
// character, item)" invariant: a duplicate CreateWish for the same key returns
// the EXISTING entry (same id, same serial) and consumes NO new serial. This is
// what makes the serial-based CANCEL_WISH resolution unambiguous.
func TestAdministratorCreateIsIdempotent(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	first, err := wish.CreateWish(db, buildWorldWish(t, tenantId, 0, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish first: %v", err)
	}

	second, err := wish.CreateWish(db, buildWorldWish(t, tenantId, 0, 100, 1302000))
	if err != nil {
		t.Fatalf("CreateWish duplicate: %v", err)
	}

	if second.Id() != first.Id() {
		t.Errorf("duplicate create returned id %s, want existing %s", second.Id(), first.Id())
	}
	if second.Serial() != first.Serial() {
		t.Errorf("duplicate create returned serial %d, want existing %d (no new serial consumed)", second.Serial(), first.Serial())
	}

	all, err := wish.GetAll()(db)()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("after duplicate create, tenant holds %d wishes, want 1", len(all))
	}

	// A different item in the same world is a distinct wish with its own serial.
	other, err := wish.CreateWish(db, buildWorldWish(t, tenantId, 0, 100, 1302001))
	if err != nil {
		t.Fatalf("CreateWish other item: %v", err)
	}
	if other.Serial() == first.Serial() {
		t.Errorf("distinct wish got the same serial %d as the first", other.Serial())
	}
}
