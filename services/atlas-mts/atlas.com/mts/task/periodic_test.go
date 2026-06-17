package task_test

import (
	"context"
	"testing"
	"time"

	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/task"
	"atlas-mts/test"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// newSweepDB builds a DB migrated with both the listing and holding schemas (the
// expire transition writes a seller holding inside its tx) and resets the shared
// in-memory tables so per-test row counts are isolated.
func newSweepDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset holdings: %v", err)
	}
	return db, func() { test.CleanupTestDB(t, db) }
}

// seedAuction persists an active listing for the given tenant with the supplied
// sale type and end time (nil for a fixed-price never-expiring row). It seeds
// under WithoutTenantFilter so a second tenant's row is not scoped away.
func seedAuction(t *testing.T, db *gorm.DB, tenantId uuid.UUID, listingId uuid.UUID, sellerId uint32, saleType listing.SaleType, endsAt *time.Time) {
	t.Helper()
	m, err := listing.NewBuilder(tenantId, 0, sellerId).
		SetId(listingId).
		SetSellerName("Seller").
		SetSaleType(saleType).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetWeaponAttack(17).
		SetSlots(7).
		SetLevel(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("onehand").
		SetEndsAt(endsAt).
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	ctx := database.WithoutTenantFilter(context.Background())
	if _, err := listing.CreateListing(db.WithContext(ctx), m); err != nil {
		t.Fatalf("seed listing: %v", err)
	}
}

// stateOf reads a listing's current state across tenants.
func stateOf(t *testing.T, db *gorm.DB, listingId uuid.UUID) listing.State {
	t.Helper()
	ctx := database.WithoutTenantFilter(context.Background())
	m, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	return m.State()
}

// holdingsForOwner returns the holdings owned by ownerId across all tenants.
func holdingsForOwner(t *testing.T, db *gorm.DB, ownerId uint32) []holding.Model {
	t.Helper()
	ctx := database.WithoutTenantFilter(context.Background())
	all, err := holding.GetAll()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("holding GetAll: %v", err)
	}
	var out []holding.Model
	for _, m := range all {
		if m.OwnerId() == ownerId {
			out = append(out, m)
		}
	}
	return out
}

// TestSweep_ExpiresOnlyExpiredActiveAuctions asserts the DB-driven sweep moves
// every expired active auction (ends_at < now, not null) to the seller's holding
// with origin=expired, ACROSS tenants, while leaving future auctions and
// never-expiring (ends_at NULL) rows untouched. It asserts the swept count is
// returned (NFR 8.3 — bounded, never silently truncated).
func TestSweep_ExpiresOnlyExpiredActiveAuctions(t *testing.T) {
	db, cleanup := newSweepDB(t)
	defer cleanup()

	tenantA := test.TestTenantId
	tenantB := uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	expiredA := uuid.New()
	const sellerExpiredA = uint32(7770001)
	seedAuction(t, db, tenantA, expiredA, sellerExpiredA, listing.SaleTypeAuction, &past)

	expiredB := uuid.New()
	const sellerExpiredB = uint32(7770002)
	seedAuction(t, db, tenantB, expiredB, sellerExpiredB, listing.SaleTypeAuction, &past)

	futureAuction := uuid.New()
	const sellerFuture = uint32(7770003)
	seedAuction(t, db, tenantA, futureAuction, sellerFuture, listing.SaleTypeAuction, &future)

	fixedPrice := uuid.New()
	const sellerFixed = uint32(7770004)
	seedAuction(t, db, tenantA, fixedPrice, sellerFixed, listing.SaleTypeFixed, nil)

	l := logrus.New()
	swept, err := task.Sweep(l, context.Background(), db)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	if swept != 2 {
		t.Fatalf("expected 2 listings swept (one per tenant), got %d", swept)
	}

	for _, c := range []struct {
		id     uuid.UUID
		seller uint32
	}{{expiredA, sellerExpiredA}, {expiredB, sellerExpiredB}} {
		if got := stateOf(t, db, c.id); got != listing.StateExpired {
			t.Fatalf("listing %s: expected state expired, got %s", c.id, got)
		}
		hs := holdingsForOwner(t, db, c.seller)
		if len(hs) != 1 {
			t.Fatalf("seller %d: expected exactly 1 holding, got %d", c.seller, len(hs))
		}
		if hs[0].Origin() != holding.OriginExpired {
			t.Fatalf("seller %d: expected holding origin=expired, got %s", c.seller, hs[0].Origin())
		}
		if hs[0].TemplateId() != 1302000 || hs[0].WeaponAttack() != 17 || hs[0].Slots() != 7 {
			t.Fatalf("seller %d: holding snapshot not copied: tmpl=%d watk=%d slots=%d", c.seller, hs[0].TemplateId(), hs[0].WeaponAttack(), hs[0].Slots())
		}
	}

	if got := stateOf(t, db, futureAuction); got != listing.StateActive {
		t.Fatalf("future auction: expected state active, got %s", got)
	}
	if got := stateOf(t, db, fixedPrice); got != listing.StateActive {
		t.Fatalf("fixed-price listing: expected state active, got %s", got)
	}
	if got := len(holdingsForOwner(t, db, sellerFuture)); got != 0 {
		t.Fatalf("future-auction seller: expected no holding, got %d", got)
	}
	if got := len(holdingsForOwner(t, db, sellerFixed)); got != 0 {
		t.Fatalf("fixed-price seller: expected no holding, got %d", got)
	}
}

// TestSweep_NoExpiredIsNoOp asserts a sweep with no expired auctions returns 0
// and creates no holdings.
func TestSweep_NoExpiredIsNoOp(t *testing.T) {
	db, cleanup := newSweepDB(t)
	defer cleanup()

	future := time.Now().Add(time.Hour)
	id := uuid.New()
	const seller = uint32(7770005)
	seedAuction(t, db, test.TestTenantId, id, seller, listing.SaleTypeAuction, &future)

	l := logrus.New()
	swept, err := task.Sweep(l, context.Background(), db)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if swept != 0 {
		t.Fatalf("expected 0 swept, got %d", swept)
	}
	if got := len(holdingsForOwner(t, db, seller)); got != 0 {
		t.Fatalf("expected no holding, got %d", got)
	}
}
