package listing_test

import (
	"atlas-mts/listing"
	"atlas-mts/saga"
	"atlas-mts/test"
	"testing"
	"time"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// captureEmitter records the saga handed to it instead of producing to Kafka, so
// the list flow can be asserted without a live broker.
type captureEmitter struct {
	saga   saga.Saga
	all    []saga.Saga
	called bool
}

func (e *captureEmitter) Create(s saga.Saga) error {
	e.saga = s
	e.all = append(e.all, s)
	e.called = true
	return nil
}

// sagas returns every saga the emitter has captured (the bid path may emit a
// hold and a release as two single-step sagas).
func (e *captureEmitter) sagas() []saga.Saga { return e.all }

// reset clears the capture state so a follow-up call's emission can be asserted
// in isolation (used by the multi-step bid/settle flows).
func (e *captureEmitter) reset() {
	e.saga = saga.Saga{}
	e.all = nil
	e.called = false
}

// newListProcessor builds a listing processor wired to a capturing saga emitter.
func newListProcessor(t *testing.T) (listing.Processor, *captureEmitter, *gorm.DB, func()) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration)
	ctx := test.CreateTestContext()
	emitter := &captureEmitter{}
	p := listing.NewProcessor(logger, ctx, db, listing.WithSagaEmitter(emitter))
	// The shared in-memory SQLite DB (file::memory:?cache=shared) persists across
	// tests in this process; clear the listings table so seed rows from a prior
	// test (e.g. the cap test) do not leak into this test's cap count.
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	cleanup := func() { test.CleanupTestDB(t, db) }
	return p, emitter, db, cleanup
}

func validFixedListRequest() listing.ListRequest {
	buyNow := uint32(0)
	_ = buyNow
	return listing.ListRequest{
		WorldId:             0,
		SellerId:            100,
		SellerName:          "Seller",
		SaleType:            listing.SaleTypeFixed,
		SourceInventoryType: 1,
		AssetId:             5001,
		Quantity:            1,
		ListValue:           1000,
		Category:            "equip",
		SubCategory:         "one-handed-sword",
	}
}

// seedActiveListings creates n active listings for the given seller in the test
// tenant so the per-character cap can be exercised.
func seedActiveListings(t *testing.T, db *gorm.DB, sellerId uint32, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
			SetSellerName("Seller").
			SetSaleType(listing.SaleTypeFixed).
			SetState(listing.StateActive).
			SetTemplateId(1302000).
			SetQuantity(1).
			SetListValue(1000).
			SetCategory("equip").
			SetSubCategory("one-handed-sword").
			Build()
		if err != nil {
			t.Fatalf("build seed listing: %v", err)
		}
		if _, err := listing.CreateListing(db, m); err != nil {
			t.Fatalf("create seed listing: %v", err)
		}
	}
}

// TestListRejectsBelowPriceFloor asserts a list with listValue below the config
// priceFloor (110) is rejected and no saga is emitted.
func TestListRejectsBelowPriceFloor(t *testing.T) {
	p, emitter, _, cleanup := newListProcessor(t)
	defer cleanup()

	req := validFixedListRequest()
	req.ListValue = 109 // below the 110 floor

	_, err := p.List(req)
	if err == nil {
		t.Fatal("expected list below price floor to be rejected, got nil error")
	}
	if emitter.called {
		t.Error("saga emitted for a sub-floor list; expected no emission")
	}
}

// TestListRejectsWhenSellerAtCap asserts a seller already at maxActiveListings
// (10) cannot create another listing.
func TestListRejectsWhenSellerAtCap(t *testing.T) {
	p, emitter, db, cleanup := newListProcessor(t)
	defer cleanup()

	seedActiveListings(t, db, 100, 10) // default maxActiveListings = 10

	_, err := p.List(validFixedListRequest())
	if err == nil {
		t.Fatal("expected list to be rejected when seller is at the active-listing cap")
	}
	if emitter.called {
		t.Error("saga emitted while seller at cap; expected no emission")
	}
}

// TestListRejectsAuctionDurationOutOfRange asserts an auction duration below the
// min (24h), above the max (168h), or non-1h-step is rejected.
func TestListRejectsAuctionDurationOutOfRange(t *testing.T) {
	cases := []struct {
		name     string
		duration int
	}{
		{"below min", 23},
		{"above max", 169},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, emitter, _, cleanup := newListProcessor(t)
			defer cleanup()

			req := validFixedListRequest()
			req.SaleType = listing.SaleTypeAuction
			req.DurationHours = c.duration

			_, err := p.List(req)
			if err == nil {
				t.Fatalf("expected auction duration %dh to be rejected", c.duration)
			}
			if emitter.called {
				t.Error("saga emitted for an out-of-range auction duration; expected no emission")
			}
		})
	}
}

// TestValidFixedListBuildsSaga asserts a valid fixed list pre-allocates a listing
// id, charges the registration fee via AwardMesos(-fee), then transfers to MTS,
// and sets a step-count-scaled timeout (N=2 for the TransferToMts composite).
func TestValidFixedListBuildsSaga(t *testing.T) {
	p, emitter, _, cleanup := newListProcessor(t)
	defer cleanup()

	req := validFixedListRequest()

	listingId, err := p.List(req)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if listingId == uuid.Nil {
		t.Fatal("List did not pre-allocate a listing id")
	}
	if !emitter.called {
		t.Fatal("expected a saga to be emitted for a valid list")
	}

	sg := emitter.saga
	if sg.SagaType != saga.MtsOperation {
		t.Errorf("saga type = %q, want %q", sg.SagaType, saga.MtsOperation)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("expected 2 steps (AwardMesos + TransferToMts), got %d", len(sg.Steps))
	}

	// Step 1: AwardMesos debit — the flat meso listing fee (default 5000).
	if sg.Steps[0].Action != saga.AwardMesos {
		t.Errorf("step[0] action = %q, want %q", sg.Steps[0].Action, saga.AwardMesos)
	}
	mp, ok := sg.Steps[0].Payload.(sharedsaga.AwardMesosPayload)
	if !ok {
		t.Fatalf("step[0] payload type = %T, want AwardMesosPayload", sg.Steps[0].Payload)
	}
	if mp.Amount != -5000 { // default flat listing fee
		t.Errorf("AwardMesos amount = %d, want -5000 (the flat listing fee)", mp.Amount)
	}
	if mp.CharacterId != req.SellerId {
		t.Errorf("AwardMesos characterId = %d, want %d", mp.CharacterId, req.SellerId)
	}

	// Step 2: TransferToMts carrying the pre-allocated listing id + sale params.
	if sg.Steps[1].Action != saga.TransferToMts {
		t.Errorf("step[1] action = %q, want %q", sg.Steps[1].Action, saga.TransferToMts)
	}
	tp, ok := sg.Steps[1].Payload.(sharedsaga.TransferToMtsPayload)
	if !ok {
		t.Fatalf("step[1] payload type = %T, want TransferToMtsPayload", sg.Steps[1].Payload)
	}
	if tp.ListingId != listingId {
		t.Errorf("TransferToMts listingId = %s, want %s (the pre-allocated id)", tp.ListingId, listingId)
	}
	if tp.CharacterId != req.SellerId {
		t.Errorf("TransferToMts characterId = %d, want %d", tp.CharacterId, req.SellerId)
	}
	if tp.AssetId != req.AssetId {
		t.Errorf("TransferToMts assetId = %d, want %d", tp.AssetId, req.AssetId)
	}
	if tp.ListValue != req.ListValue {
		t.Errorf("TransferToMts listValue = %d, want %d", tp.ListValue, req.ListValue)
	}
	if tp.SaleType != string(req.SaleType) {
		t.Errorf("TransferToMts saleType = %q, want %q", tp.SaleType, req.SaleType)
	}
	if tp.CommissionRate != 0.07 { // default commissionRate (client m_nCommissionRate = 7%)
		t.Errorf("TransferToMts commissionRate = %v, want 0.07", tp.CommissionRate)
	}

	// Timeout must be set (never default) and scaled for N=2 expansion: the fee
	// step + the TransferToMts composite which expands to 2 steps.
	if sg.Timeout <= 0 {
		t.Errorf("saga timeout = %d, want a positive explicit timeout", sg.Timeout)
	}
	got := time.Duration(sg.Timeout) * time.Millisecond
	// base 10s + perStep 1s * 2 = 12s
	if got < 12*time.Second {
		t.Errorf("saga timeout = %s, want at least the N=2 scaled budget (>= 12s)", got)
	}

	// No listing row is created here — the row is created only on the custody
	// consumer's AcceptToMtsListing.
	got2, err := p.GetById(listingId.String())
	if err == nil && got2.Id() == listingId {
		t.Error("a listing row was created by List; the row must only be created on AcceptToMtsListing")
	}
}
