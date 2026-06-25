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

// buildProcessorListingTemplate is buildProcessorListing with a caller-chosen
// template id, for the ItemIds IN-filter test (which needs listings with distinct
// template ids to assert the set membership).
func buildProcessorListingTemplate(t *testing.T, worldId world.Id, sellerId uint32, category string, templateId uint32) listing.Model {
	t.Helper()
	buyNow := uint32(5000)
	m, err := listing.NewBuilder(test.TestTenantId, worldId, sellerId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(templateId).
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

// TestProcessorBrowseByItemIds asserts the ItemIds browse filter narrows the
// result to listings whose template_id is in the provided set (template_id IN (?)).
// This backs the marketplace name search: a search term resolves to matching item
// template ids and the browse is filtered on them.
func TestProcessorBrowseByItemIds(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	resetListings(t, db)

	if _, err := p.Create(buildProcessorListingTemplate(t, 0, 300, "equip", 1302000)); err != nil {
		t.Fatalf("Create 1302000: %v", err)
	}
	if _, err := p.Create(buildProcessorListingTemplate(t, 0, 301, "equip", 1302001)); err != nil {
		t.Fatalf("Create 1302001: %v", err)
	}
	if _, err := p.Create(buildProcessorListingTemplate(t, 0, 302, "equip", 1402000)); err != nil {
		t.Fatalf("Create 1402000: %v", err)
	}

	// Filter to the two-id set {1302000, 1402000} => exactly those two rows.
	got, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{ItemIds: []uint32{1302000, 1402000}})
	if err != nil {
		t.Fatalf("Browse by itemIds: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Browse(itemIds={1302000,1402000}) returned %d rows, want 2", len(got))
	}
	for _, m := range got {
		if m.TemplateId() != 1302000 && m.TemplateId() != 1402000 {
			t.Errorf("Browse returned unexpected template id %d", m.TemplateId())
		}
	}

	// A single-id set narrows to one row.
	one, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{ItemIds: []uint32{1302001}})
	if err != nil {
		t.Fatalf("Browse by single itemId: %v", err)
	}
	if len(one) != 1 || one[0].TemplateId() != 1302001 {
		t.Fatalf("Browse(itemIds={1302001}) returned %d rows, want 1 with template 1302001", len(one))
	}

	// An id set matching nothing yields an empty result.
	none, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{ItemIds: []uint32{9999999}})
	if err != nil {
		t.Fatalf("Browse by missing itemId: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("Browse(itemIds={9999999}) returned %d rows, want 0", len(none))
	}
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

// TestProcessorBrowseBySerial asserts the Serial browse filter narrows the result
// to the single listing carrying that ITC serial. This backs the channel-side
// serial->listing resolution the zzim/wish ITC arms rely on (SET_ZZIM/DELETE_ZZIM/
// CANCEL_WISH resolve nITCSN -> templateId via this filtered browse).
func TestProcessorBrowseBySerial(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	resetListings(t, db)

	a, err := p.Create(buildProcessorListing(t, 0, 200, "equip"))
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	if _, err := p.Create(buildProcessorListing(t, 0, 201, "use")); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	if a.Serial() == 0 {
		t.Fatalf("expected a non-zero serial assigned on create")
	}

	got, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{Serial: a.Serial()})
	if err != nil {
		t.Fatalf("Browse by serial: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Browse(serial=%d) returned %d rows, want 1", a.Serial(), len(got))
	}
	if got[0].Serial() != a.Serial() || got[0].SellerId() != 200 {
		t.Errorf("Browse by serial returned wrong row: serial=%d seller=%d", got[0].Serial(), got[0].SellerId())
	}

	// A serial that matches no row yields an empty result.
	none, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{Serial: 99999999})
	if err != nil {
		t.Fatalf("Browse by missing serial: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("Browse(serial=99999999) returned %d rows, want 0", len(none))
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
