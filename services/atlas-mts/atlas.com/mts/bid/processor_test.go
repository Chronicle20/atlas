package bid_test

import (
	"atlas-mts/bid"
	"atlas-mts/test"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// resetBids clears the bids table. The shared in-memory SQLite DB is reused
// across tests in the process, and these processor tests all run under the
// fixed test tenant, so rows from prior tests would otherwise leak into
// GetByListingId/GetAll counts.
func resetBids(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM bids").Error; err != nil {
		t.Fatalf("reset bids: %v", err)
	}
}

// buildProcessorBid builds a held bid for the test tenant. The tenant id MUST
// match the processor's context tenant (test.TestTenantId) so the row is visible
// through the processor's tenant-scoped queries.
func buildProcessorBid(t *testing.T, listingId uuid.UUID, bidderId uint32) bid.Model {
	t.Helper()
	m, err := bid.NewBuilder(test.TestTenantId, listingId, bidderId).
		SetAmount(1000).
		SetEscrowTxnId(uuid.New()).
		SetState(bid.StateHeld).
		Build()
	if err != nil {
		t.Fatalf("Failed to build bid: %v", err)
	}
	return m
}

// TestProcessorCreateGetById asserts a created bid round-trips through the
// processor's GetById.
func TestProcessorCreateGetById(t *testing.T) {
	p, db, cleanup := test.CreateBidProcessor(t)
	defer cleanup()
	resetBids(t, db)

	listingId := uuid.New()
	created, err := p.Create(buildProcessorBid(t, listingId, 100))
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
	if got.ListingId() != listingId {
		t.Errorf("listingId = %s, want %s", got.ListingId(), listingId)
	}
	if got.State() != bid.StateHeld {
		t.Errorf("state = %q, want held", got.State())
	}
}

// TestProcessorGetByListingId asserts GetByListingId filters by listing.
func TestProcessorGetByListingId(t *testing.T) {
	p, db, cleanup := test.CreateBidProcessor(t)
	defer cleanup()
	resetBids(t, db)

	listingA := uuid.New()
	listingB := uuid.New()

	if _, err := p.Create(buildProcessorBid(t, listingA, 100)); err != nil {
		t.Fatalf("Create A #1: %v", err)
	}
	if _, err := p.Create(buildProcessorBid(t, listingA, 101)); err != nil {
		t.Fatalf("Create A #2: %v", err)
	}
	if _, err := p.Create(buildProcessorBid(t, listingB, 102)); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	got, err := p.GetByListingId(listingA)
	if err != nil {
		t.Fatalf("GetByListingId: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetByListingId(A) returned %d rows, want 2", len(got))
	}
	for _, b := range got {
		if b.ListingId() != listingA {
			t.Errorf("GetByListingId returned wrong row: listing=%s", b.ListingId())
		}
	}
}

// TestProcessorTransitionState asserts the conditional transition: the first
// held->won succeeds (true); a second returns false.
func TestProcessorTransitionState(t *testing.T) {
	p, db, cleanup := test.CreateBidProcessor(t)
	defer cleanup()
	resetBids(t, db)

	created, err := p.Create(buildProcessorBid(t, uuid.New(), 100))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := p.TransitionState(created.Id().String(), bid.StateHeld, bid.StateWon)
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
	if got.State() != bid.StateWon {
		t.Errorf("state = %q, want won", got.State())
	}

	ok, err = p.TransitionState(created.Id().String(), bid.StateHeld, bid.StateWon)
	if err != nil {
		t.Fatalf("TransitionState second: %v", err)
	}
	if ok {
		t.Error("second TransitionState returned true, want false")
	}
}
