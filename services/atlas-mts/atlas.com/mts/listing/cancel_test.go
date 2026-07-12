package listing_test

import (
	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/test"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// newCancelProcessor builds a listing processor backed by a DB migrated with both
// the listing and holding schemas (Cancel writes a seller holding inside its tx).
func newCancelProcessor(t *testing.T) (listing.Processor, *gorm.DB, func()) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	p := listing.NewProcessor(logger, ctx, db)
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset holdings: %v", err)
	}
	cleanup := func() { test.CleanupTestDB(t, db) }
	return p, db, cleanup
}

// seedActiveListingRow persists an active listing with a known item snapshot.
func seedActiveListingRow(t *testing.T, db *gorm.DB, listingId uuid.UUID, sellerId uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
		SetId(listingId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
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
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	stored, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	return stored
}

// holdingsForSeller returns the (non-deleted) holdings owned by ownerId. The
// cache=shared in-memory DB leaks rows across tests, so per-owner filtering keeps
// the "exactly one holding" assertion isolated.
func holdingsForSeller(t *testing.T, db *gorm.DB, ownerId uint32) []holding.Model {
	t.Helper()
	all, err := holding.GetAll()(db.WithContext(test.CreateTestContext()))()
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

// TestCancel_ActiveMovesToSellerHolding asserts a cancel of an active listing
// transitions the row to cancelled and creates exactly one seller holding with
// origin=cancelled carrying the listing's item snapshot, in one tx (Won=true).
func TestCancel_ActiveMovesToSellerHolding(t *testing.T) {
	p, db, cleanup := newCancelProcessor(t)
	defer cleanup()

	listingId := uuid.New()
	const sellerId = uint32(7770001)
	seedActiveListingRow(t, db, listingId, sellerId)

	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !res.Won {
		t.Fatal("expected Cancel of an active listing to win the race (Won=true)")
	}
	if res.HoldingId == uuid.Nil {
		t.Error("expected a non-nil holding id on a winning cancel")
	}
	if res.SellerId != sellerId {
		t.Errorf("CancelResult.SellerId = %d, want %d", res.SellerId, sellerId)
	}
	if res.ItemId != 1302000 {
		t.Errorf("CancelResult.ItemId = %d, want 1302000", res.ItemId)
	}

	// listing transitioned to cancelled
	stored, err := p.GetById(listingId.String())
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateCancelled {
		t.Fatalf("expected listing state cancelled, got %s", stored.State())
	}

	// exactly one seller holding, origin cancelled, snapshot copied
	hs := holdingsForSeller(t, db, sellerId)
	if len(hs) != 1 {
		t.Fatalf("expected exactly 1 holding for seller %d, got %d", sellerId, len(hs))
	}
	h := hs[0]
	if h.Origin() != holding.OriginCancelled {
		t.Fatalf("expected origin cancelled, got %s", h.Origin())
	}
	if h.TemplateId() != 1302000 || h.Quantity() != 1 || h.WeaponAttack() != 17 || h.Slots() != 7 {
		t.Fatalf("holding snapshot not copied: tmpl=%d qty=%d watk=%d slots=%d", h.TemplateId(), h.Quantity(), h.WeaponAttack(), h.Slots())
	}
}

// TestCancel_NonActiveIsNoOp asserts cancelling a non-active listing (race loser /
// already settled) returns Won=false and creates no holding — the conditional
// WHERE state='active' affects 0 rows.
func TestCancel_NonActiveIsNoOp(t *testing.T) {
	p, db, cleanup := newCancelProcessor(t)
	defer cleanup()

	listingId := uuid.New()
	const sellerId = uint32(7770002)
	seedActiveListingRow(t, db, listingId, sellerId)

	// Simulate a concurrent buy winning the race: the listing is already sold.
	if _, err := listing.UpdateState(db, listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate concurrent buy: %v", err)
	}

	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if res.Won {
		t.Fatal("expected Cancel of a non-active listing to lose the race (Won=false)")
	}

	// listing remains sold (cancel did not clobber it)
	stored, err := p.GetById(listingId.String())
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing to remain sold, got %s", stored.State())
	}

	// no seller holding created
	if got := len(holdingsForSeller(t, db, sellerId)); got != 0 {
		t.Fatalf("expected no holding for race-losing seller %d, got %d", sellerId, got)
	}
}
