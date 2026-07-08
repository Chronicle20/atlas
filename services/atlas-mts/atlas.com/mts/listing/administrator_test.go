package listing_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func adminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, listing.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func buildActiveListing(t *testing.T, tenantId uuid.UUID, sellerId uint32) listing.Model {
	t.Helper()
	buyNow := uint32(5000)
	m, err := listing.NewBuilder(tenantId, 0, sellerId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
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
		SetListValue(1000).
		SetBuyNowPrice(&buyNow).
		SetCommissionRate(0.05).
		SetCategory("equip").
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("Failed to build listing: %v", err)
	}
	return m
}

// TestAdministratorCreateGetById asserts a created listing round-trips through
// getById and preserves the full snapshot (explicit columns, not a JSON blob).
func TestAdministratorCreateGetById(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	m := buildActiveListing(t, tenantId, 100)
	created, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("CreateListing: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("CreateListing did not assign an id")
	}

	got, err := listing.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.TenantId() != tenantId {
		t.Errorf("tenantId = %s, want %s", got.TenantId(), tenantId)
	}
	if got.SellerId() != 100 {
		t.Errorf("sellerId = %d, want 100", got.SellerId())
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
	if got.BuyNowPrice() == nil || *got.BuyNowPrice() != 5000 {
		t.Errorf("buyNowPrice not round-tripped: %v", got.BuyNowPrice())
	}
	if got.CommissionRate() != 0.05 {
		t.Errorf("commissionRate = %v, want 0.05", got.CommissionRate())
	}
	if got.Category() != "equip" || got.SubCategory() != "one-handed-sword" {
		t.Errorf("category not round-tripped: %q / %q", got.Category(), got.SubCategory())
	}
	if got.State() != listing.StateActive {
		t.Errorf("state = %q, want active", got.State())
	}
}

// TestAdministratorCrossTenantIsolation asserts tenant B cannot read tenant A's
// listing even within the same world.
func TestAdministratorCrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := adminTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	m := buildActiveListing(t, tenantA, 100)
	created, err := listing.CreateListing(dbA, m)
	if err != nil {
		t.Fatalf("CreateListing tenant A: %v", err)
	}

	// Tenant A can read its own listing.
	if _, err := listing.GetById(created.Id().String())(dbA)(); err != nil {
		t.Fatalf("tenant A GetById own listing: %v", err)
	}

	// Tenant B (same world) must NOT see tenant A's listing.
	if _, err := listing.GetById(created.Id().String())(dbB)(); err == nil {
		t.Error("tenant B was able to read tenant A's listing")
	}

	// Tenant B's getAll must be empty.
	allB, err := listing.GetAll()(dbB)()
	if err != nil {
		t.Fatalf("tenant B GetAll: %v", err)
	}
	if len(allB) != 0 {
		t.Errorf("tenant B GetAll returned %d rows, want 0", len(allB))
	}
}

// TestAdministratorUpdateStateConditional asserts the race-safe conditional
// transition: active->cancelled succeeds (1 row), a second active->cancelled
// affects 0 rows.
func TestAdministratorUpdateStateConditional(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	m := buildActiveListing(t, tenantId, 100)
	created, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("CreateListing: %v", err)
	}

	affected, err := listing.UpdateState(db, created.Id().String(), listing.StateActive, listing.StateCancelled)
	if err != nil {
		t.Fatalf("UpdateState first: %v", err)
	}
	if affected != 1 {
		t.Errorf("first UpdateState affected %d rows, want 1", affected)
	}

	got, err := listing.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById after transition: %v", err)
	}
	if got.State() != listing.StateCancelled {
		t.Errorf("state = %q, want cancelled", got.State())
	}

	// A second active->cancelled must be a no-op (the row is no longer active).
	affected, err = listing.UpdateState(db, created.Id().String(), listing.StateActive, listing.StateCancelled)
	if err != nil {
		t.Fatalf("UpdateState second: %v", err)
	}
	if affected != 0 {
		t.Errorf("second UpdateState affected %d rows, want 0", affected)
	}
}

// TestAdministratorUpdateStateMalformedIdIsScoped is the regression guard for the
// GORM zero-value struct-condition elision bug: a malformed (non-UUID) id must
// NOT transition every active listing in the tenant.
func TestAdministratorUpdateStateMalformedIdIsScoped(t *testing.T) {
	for _, badId := range []string{"bad", ""} {
		t.Run("id="+badId, func(t *testing.T) {
			tenantId := uuid.New()
			ctx := tenantCtx(t, tenantId)
			db := adminTestDB(t).WithContext(ctx)

			var ids []string
			for i := 0; i < 3; i++ {
				created, err := listing.CreateListing(db, buildActiveListing(t, tenantId, uint32(100+i)))
				if err != nil {
					t.Fatalf("CreateListing #%d: %v", i, err)
				}
				ids = append(ids, created.Id().String())
			}

			affected, err := listing.UpdateState(db, badId, listing.StateActive, listing.StateCancelled)
			if err == nil {
				t.Errorf("UpdateState(%q) returned nil error, want a malformed-id error", badId)
			}
			if affected != 0 {
				t.Errorf("UpdateState(%q) affected %d rows, want 0", badId, affected)
			}

			for _, id := range ids {
				got, err := listing.GetById(id)(db)()
				if err != nil {
					t.Fatalf("GetById %s: %v", id, err)
				}
				if got.State() != listing.StateActive {
					t.Errorf("after UpdateState(%q) listing %s state = %q, want active (unchanged)", badId, id, got.State())
				}
			}
		})
	}
}

// TestAdministratorUpdateAuctionMalformedIdIsScoped is the regression guard for
// the most dangerous variant: UpdateAuction has NO state predicate, so an elided
// zero id would rewrite EVERY listing's auction fields. A malformed id must mutate
// nothing.
func TestAdministratorUpdateAuctionMalformedIdIsScoped(t *testing.T) {
	for _, badId := range []string{"bad", ""} {
		t.Run("id="+badId, func(t *testing.T) {
			tenantId := uuid.New()
			ctx := tenantCtx(t, tenantId)
			db := adminTestDB(t).WithContext(ctx)

			var ids []string
			for i := 0; i < 2; i++ {
				created, err := listing.CreateListing(db, buildActiveListing(t, tenantId, uint32(100+i)))
				if err != nil {
					t.Fatalf("CreateListing #%d: %v", i, err)
				}
				ids = append(ids, created.Id().String())
			}

			err := listing.UpdateAuction(db, badId, 99999, 777, nil)
			if err == nil {
				t.Errorf("UpdateAuction(%q) returned nil error, want a malformed-id error", badId)
			}

			for _, id := range ids {
				got, err := listing.GetById(id)(db)()
				if err != nil {
					t.Fatalf("GetById %s: %v", id, err)
				}
				if got.CurrentBid() == 99999 || got.HighBidderId() == 777 {
					t.Errorf("after UpdateAuction(%q) listing %s auction fields were mutated (currentBid=%d, highBidderId=%d)", badId, id, got.CurrentBid(), got.HighBidderId())
				}
			}
		})
	}
}

// TestAdministratorMultipleListingsPerTenant asserts a single tenant can hold
// many active listings concurrently. Guards against a unique constraint on
// tenant_id alone (which would cap a tenant at one listing and break the
// maxActiveListings rule). The (tenant_id, id) unique index must permit this.
func TestAdministratorMultipleListingsPerTenant(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	for i := 0; i < 3; i++ {
		m := buildActiveListing(t, tenantId, uint32(100+i))
		if _, err := listing.CreateListing(db, m); err != nil {
			t.Fatalf("CreateListing #%d for tenant: %v", i, err)
		}
	}

	all, err := listing.GetAll()(db)()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("tenant holds %d listings, want 3", len(all))
	}
}

// TestAdministratorIndexesExist asserts the three design indexes are created by
// the migration.
func TestAdministratorIndexesExist(t *testing.T) {
	db := adminTestDB(t)
	mig := db.Migrator()

	want := []string{
		"idx_listings_world_state_category",
		"idx_listings_seller_state",
		"idx_listings_world_ends_at",
	}
	for _, name := range want {
		if !mig.HasIndex(&listingIndexProbe{}, name) {
			t.Errorf("expected index %q to exist on listings", name)
		}
	}
}

// listingIndexProbe lets the migrator resolve the listings table for HasIndex
// without exporting the unexported entity type.
type listingIndexProbe struct{}

func (listingIndexProbe) TableName() string { return "listings" }

func TestBackdateEndsAt(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	future := time.Now().Add(24 * time.Hour)
	auction, err := listing.NewBuilder(test.TestTenantId, 0, 1001).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("3").
		SetSubCategory("1").
		SetEndsAt(&future).
		Build()
	if err != nil {
		t.Fatalf("build auction: %v", err)
	}
	created, err := listing.CreateListing(db, auction)
	if err != nil {
		t.Fatalf("create auction: %v", err)
	}

	fixed, err := listing.NewBuilder(test.TestTenantId, 0, 1002).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("1").
		SetSubCategory("1").
		Build()
	if err != nil {
		t.Fatalf("build fixed: %v", err)
	}
	createdFixed, err := listing.CreateListing(db, fixed)
	if err != nil {
		t.Fatalf("create fixed: %v", err)
	}

	past := time.Now().Add(-time.Second)

	// Active auction: backdates, 1 row.
	rows, err := listing.BackdateEndsAt(db, created.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate auction: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}
	got, err := listing.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("reload auction: %v", err)
	}
	if got.EndsAt() == nil || !got.EndsAt().Before(time.Now()) {
		t.Fatalf("expected backdated endsAt, got %v", got.EndsAt())
	}

	// Fixed-sale listing WITHOUT a term (legacy NULL ends_at): refused, 0 rows.
	rows, err = listing.BackdateEndsAt(db, createdFixed.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate fixed: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows for termless fixed sale, got %d", rows)
	}

	// Fixed-sale listing WITH a sale term (era-faithful expiry): backdates, 1 row.
	term := time.Now().Add(168 * time.Hour)
	fixedWithTerm, err := listing.NewBuilder(test.TestTenantId, 0, 1003).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("1").
		SetSubCategory("1").
		SetEndsAt(&term).
		Build()
	if err != nil {
		t.Fatalf("build fixed-with-term: %v", err)
	}
	createdFixedTerm, err := listing.CreateListing(db, fixedWithTerm)
	if err != nil {
		t.Fatalf("create fixed-with-term: %v", err)
	}
	rows, err = listing.BackdateEndsAt(db, createdFixedTerm.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate fixed-with-term: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row for fixed sale with a term, got %d", rows)
	}

	// Non-active auction: refused, 0 rows.
	if _, err := listing.UpdateState(db, created.Id().String(), listing.StateActive, listing.StateExpired); err != nil {
		t.Fatalf("transition: %v", err)
	}
	rows, err = listing.BackdateEndsAt(db, created.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate expired: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows for non-active listing, got %d", rows)
	}
}

// TestDeleteActive pins the late-compensation inverse of AcceptToMtsListing:
// DeleteActive removes a listing ONLY while active (1 row), and leaves a
// non-active listing untouched (0 rows) so a bought/cancelled/settled listing is
// never destroyed.
func TestDeleteActive(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	mk := func(seller uint32) listing.Model {
		m, err := listing.NewBuilder(test.TestTenantId, 0, seller).
			SetSellerName("Seller").
			SetSaleType(listing.SaleTypeFixed).
			SetState(listing.StateActive).
			SetTemplateId(1302000).
			SetQuantity(1).
			SetListValue(1000).
			SetCommissionRate(0.10).
			SetCategory("1").
			SetSubCategory("1").
			Build()
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		created, err := listing.CreateListing(db, m)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		return created
	}

	// Active listing: deleted, 1 row, gone afterward.
	active := mk(1001)
	rows, err := listing.DeleteActive(db, active.Id().String())
	if err != nil {
		t.Fatalf("delete active: %v", err)
	}
	if rows != 1 {
		t.Fatalf("delete active affected %d rows, want 1", rows)
	}
	if _, err := listing.GetById(active.Id().String())(db)(); err == nil {
		t.Fatal("expected active listing to be gone after DeleteActive")
	}

	// Sold listing: guard refuses it (0 rows), row still present.
	sold := mk(1002)
	if _, err := listing.UpdateState(db, sold.Id().String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("transition to sold: %v", err)
	}
	rows, err = listing.DeleteActive(db, sold.Id().String())
	if err != nil {
		t.Fatalf("delete sold: %v", err)
	}
	if rows != 0 {
		t.Fatalf("delete sold affected %d rows, want 0 (guard must refuse a non-active listing)", rows)
	}
	if _, err := listing.GetById(sold.Id().String())(db)(); err != nil {
		t.Fatalf("sold listing must survive DeleteActive: %v", err)
	}

	// Garbage id: error, no tenant-wide wipe.
	if _, err := listing.DeleteActive(db, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid listing id")
	}
}
