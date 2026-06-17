package listing_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"gorm.io/gorm"
)

// resetListings clears the listings table. The shared in-memory SQLite DB
// (file::memory:?cache=shared) is reused across tests in the process, and these
// processor tests all run under the fixed test tenant, so rows from prior tests
// would otherwise leak into Browse/GetAll counts. Truncating up front makes each
// processor test deterministic regardless of execution order.
func resetListings(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
}

// buildProcessorListing builds an active fixed-sale listing for the test tenant.
// The tenant id MUST match the processor's context tenant (test.TestTenantId)
// so the row is visible through the processor's tenant-scoped queries.
func buildProcessorListing(t *testing.T, worldId world.Id, sellerId uint32, category string) listing.Model {
	t.Helper()
	buyNow := uint32(5000)
	m, err := listing.NewBuilder(test.TestTenantId, worldId, sellerId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetBuyNowPrice(&buyNow).
		SetCommissionRate(0.05).
		SetCategory(category).
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("Failed to build listing: %v", err)
	}
	return m
}

// TestProcessorCreateGetById asserts a created listing round-trips through the
// processor's GetById.
func TestProcessorCreateGetById(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	resetListings(t, db)

	created, err := p.Create(buildProcessorListing(t, 0, 100, "equip"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Id().String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("Create did not assign an id")
	}

	got, err := p.GetById(created.Id().String())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.SellerId() != 100 {
		t.Errorf("sellerId = %d, want 100", got.SellerId())
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("templateId = %d, want 1302000", got.TemplateId())
	}
	if got.State() != listing.StateActive {
		t.Errorf("state = %q, want active", got.State())
	}
}

// TestProcessorBrowse asserts Browse filters by world, state, and category.
func TestProcessorBrowse(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	resetListings(t, db)

	// world 0, equip
	if _, err := p.Create(buildProcessorListing(t, 0, 100, "equip")); err != nil {
		t.Fatalf("Create w0 equip: %v", err)
	}
	// world 0, use
	if _, err := p.Create(buildProcessorListing(t, 0, 101, "use")); err != nil {
		t.Fatalf("Create w0 use: %v", err)
	}
	// world 1, equip
	if _, err := p.Create(buildProcessorListing(t, 1, 102, "equip")); err != nil {
		t.Fatalf("Create w1 equip: %v", err)
	}

	// Browse world 0, active, equip => exactly 1 row.
	got, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{Category: "equip"})
	if err != nil {
		t.Fatalf("Browse: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Browse(w0, active, equip) returned %d rows, want 1", len(got))
	}
	if got[0].WorldId() != 0 || got[0].Category() != "equip" || got[0].State() != listing.StateActive {
		t.Errorf("Browse returned wrong row: world=%d category=%q state=%q",
			got[0].WorldId(), got[0].Category(), got[0].State())
	}

	// Browse world 0, active, use => exactly 1 row.
	gotUse, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{Category: "use"})
	if err != nil {
		t.Fatalf("Browse use: %v", err)
	}
	if len(gotUse) != 1 {
		t.Errorf("Browse(w0, active, use) returned %d rows, want 1", len(gotUse))
	}

	// Browse world 0, cancelled, equip => 0 rows (none cancelled).
	gotCancelled, err := p.Browse(0, listing.StateCancelled, listing.BrowseFilter{Category: "equip"})
	if err != nil {
		t.Fatalf("Browse cancelled: %v", err)
	}
	if len(gotCancelled) != 0 {
		t.Errorf("Browse(w0, cancelled, equip) returned %d rows, want 0", len(gotCancelled))
	}
}

// TestProcessorTransitionState asserts the conditional transition: the first
// active->cancelled succeeds (true); a second returns false (the row is no
// longer active).
func TestProcessorTransitionState(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	resetListings(t, db)

	created, err := p.Create(buildProcessorListing(t, 0, 100, "equip"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := p.TransitionState(created.Id().String(), listing.StateActive, listing.StateCancelled)
	if err != nil {
		t.Fatalf("TransitionState first: %v", err)
	}
	if !ok {
		t.Error("first TransitionState returned false, want true")
	}

	got, err := p.GetById(created.Id().String())
	if err != nil {
		t.Fatalf("GetById after transition: %v", err)
	}
	if got.State() != listing.StateCancelled {
		t.Errorf("state = %q, want cancelled", got.State())
	}

	// A second active->cancelled must be a no-op (false).
	ok, err = p.TransitionState(created.Id().String(), listing.StateActive, listing.StateCancelled)
	if err != nil {
		t.Fatalf("TransitionState second: %v", err)
	}
	if ok {
		t.Error("second TransitionState returned true, want false")
	}
}
