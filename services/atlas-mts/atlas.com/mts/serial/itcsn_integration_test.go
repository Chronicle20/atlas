package serial_test

import (
	"testing"

	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// combinedDB migrates the listing + holding (and, transitively, the shared serial
// counter) schemas into one in-memory DB so the cross-table serial behaviour can
// be exercised.
func combinedDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, listing.Migration, holding.Migration)
}

func buildListing(t *testing.T, tenantId uuid.UUID, worldId world.Id, sellerId uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(tenantId, worldId, sellerId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.05).
		SetCategory("equip").
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	return m
}

func buildHolding(t *testing.T, tenantId uuid.UUID, worldId world.Id, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(tenantId, worldId, ownerId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	return m
}

// TestSerialSharedAcrossListingsAndHoldings asserts the serial counter is shared
// across BOTH tables in a (tenant, world): interleaved listing/holding creates
// draw strictly increasing, distinct serials (no listing serial equals any
// holding serial in the same world).
func TestSerialSharedAcrossListingsAndHoldings(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := combinedDB(t).WithContext(ctx)

	var serials []uint32

	l1, err := listing.CreateListing(db, buildListing(t, tenantId, 0, 100))
	if err != nil {
		t.Fatalf("create listing 1: %v", err)
	}
	serials = append(serials, l1.Serial())

	h1, err := holding.CreateHolding(db, buildHolding(t, tenantId, 0, 200))
	if err != nil {
		t.Fatalf("create holding 1: %v", err)
	}
	serials = append(serials, h1.Serial())

	l2, err := listing.CreateListing(db, buildListing(t, tenantId, 0, 101))
	if err != nil {
		t.Fatalf("create listing 2: %v", err)
	}
	serials = append(serials, l2.Serial())

	h2, err := holding.CreateHolding(db, buildHolding(t, tenantId, 0, 201))
	if err != nil {
		t.Fatalf("create holding 2: %v", err)
	}
	serials = append(serials, h2.Serial())

	// Monotonic, distinct, from a single shared sequence: 1,2,3,4.
	want := []uint32{1, 2, 3, 4}
	for i := range want {
		if serials[i] != want[i] {
			t.Errorf("serial #%d = %d, want %d (shared monotonic counter): got sequence %v", i, serials[i], want[i], serials)
		}
	}
	// Distinctness across the two tables.
	seen := map[uint32]bool{}
	for _, s := range serials {
		if seen[s] {
			t.Errorf("serial %d assigned twice across listings+holdings in the same world", s)
		}
		seen[s] = true
	}
}

// TestSerialIndependentPerWorldAcrossTables asserts each world has its own shared
// counter: a listing in world 0 and a holding in world 1 both start at 1.
func TestSerialIndependentPerWorldAcrossTables(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := combinedDB(t).WithContext(ctx)

	l0, err := listing.CreateListing(db, buildListing(t, tenantId, world.Id(0), 100))
	if err != nil {
		t.Fatalf("create listing world 0: %v", err)
	}
	h1, err := holding.CreateHolding(db, buildHolding(t, tenantId, world.Id(1), 200))
	if err != nil {
		t.Fatalf("create holding world 1: %v", err)
	}

	if l0.Serial() != 1 {
		t.Errorf("world 0 listing serial = %d, want 1", l0.Serial())
	}
	if h1.Serial() != 1 {
		t.Errorf("world 1 holding serial = %d, want 1 (independent per-world counter)", h1.Serial())
	}
}

// TestListingGetBySerialRoundTrip asserts a created listing resolves by its serial
// (world-scoped) and is tenant-isolated.
func TestListingGetBySerialRoundTrip(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := combinedDB(t)
	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	created, err := listing.CreateListing(dbA, buildListing(t, tenantA, world.Id(0), 100))
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	got, err := listing.GetBySerial(world.Id(0), created.Serial())(dbA)()
	if err != nil {
		t.Fatalf("GetBySerial: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("GetBySerial resolved id %s, want %s", got.Id(), created.Id())
	}
	if got.Serial() != created.Serial() {
		t.Errorf("round-tripped serial %d, want %d", got.Serial(), created.Serial())
	}

	// Wrong world must NOT resolve.
	if _, err := listing.GetBySerial(world.Id(1), created.Serial())(dbA)(); err == nil {
		t.Error("GetBySerial resolved a listing under the wrong world")
	}
	// Tenant B must NOT resolve tenant A's listing.
	if _, err := listing.GetBySerial(world.Id(0), created.Serial())(dbB)(); err == nil {
		t.Error("GetBySerial leaked tenant A's listing to tenant B")
	}
}

// TestHoldingGetBySerialRoundTrip asserts a created holding resolves by its serial
// (world-scoped) and is tenant-isolated.
func TestHoldingGetBySerialRoundTrip(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := combinedDB(t)
	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	created, err := holding.CreateHolding(dbA, buildHolding(t, tenantA, world.Id(0), 200))
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}

	got, err := holding.GetBySerial(world.Id(0), created.Serial())(dbA)()
	if err != nil {
		t.Fatalf("GetBySerial: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("GetBySerial resolved id %s, want %s", got.Id(), created.Id())
	}

	if _, err := holding.GetBySerial(world.Id(1), created.Serial())(dbA)(); err == nil {
		t.Error("GetBySerial resolved a holding under the wrong world")
	}
	if _, err := holding.GetBySerial(world.Id(0), created.Serial())(dbB)(); err == nil {
		t.Error("GetBySerial leaked tenant A's holding to tenant B")
	}
}
