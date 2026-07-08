package listing_test

import (
	"atlas-mts/listing"
	"errors"
	"atlas-mts/saga"
	"atlas-mts/test"
	"testing"
	"time"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// stubBalanceReader is an injectable buyer-prepaid balance source so the buy flow
// can be exercised without a live cashshop wallet REST read. It records the
// account it was asked about and returns a fixed balance.
type stubBalanceReader struct {
	prepaid     uint32
	err         error
	askedAcctId uint32
	called      bool
}

func (r *stubBalanceReader) PrepaidBalance(accountId uint32) (uint32, error) {
	r.called = true
	r.askedAcctId = accountId
	return r.prepaid, r.err
}

// newBuyProcessor builds a listing processor wired to a capturing saga emitter and
// the supplied balance reader, plus seeds one active fixed-price listing for the
// seller and returns its id.
func newBuyProcessor(t *testing.T, br listing.BalanceReader) (listing.Processor, *captureEmitter, *gorm.DB, uuid.UUID, func()) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration)
	ctx := test.CreateTestContext()
	emitter := &captureEmitter{}
	p := listing.NewProcessor(logger, ctx, db,
		listing.WithSagaEmitter(emitter),
		listing.WithBalanceReader(br),
	)
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	cleanup := func() { test.CleanupTestDB(t, db) }
	return p, emitter, db, seedActiveListingForBuy(t, db), cleanup
}

// seedActiveListingForBuy persists an active fixed-price listing with listValue
// 1000 and commissionRate 0.10 (markedUp = 1100) and returns its id.
func seedActiveListingForBuy(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerForBuy).
		SetId(id).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("build seed listing: %v", err)
	}
	if _, err := listing.CreateListing(db, m); err != nil {
		t.Fatalf("create seed listing: %v", err)
	}
	return id
}

const (
	sellerForBuy   = uint32(100)
	buyerForBuy    = uint32(200)
	buyerAcctForBuy = uint32(2000)
	sellerAcctForBuy = uint32(1000)
)

func buyRequest(listingId uuid.UUID) listing.BuyRequest {
	return listing.BuyRequest{
		WorldId:         0,
		ListingId:       listingId,
		BuyerId:         buyerForBuy,
		BuyerAccountId:  buyerAcctForBuy,
		SellerAccountId: sellerAcctForBuy,
	}
}

// TestBuyRejectsInsufficientPrepaid asserts a buyer whose NX Prepaid is below
// markedUpPrice (listValue 1000 * (1 + 0.10) = 1100) is rejected and no saga is
// emitted (nothing granted, nothing moved).
func TestBuyRejectsInsufficientPrepaid(t *testing.T) {
	br := &stubBalanceReader{prepaid: 1099} // one short of the 1100 marked-up price
	p, emitter, _, listingId, cleanup := newBuyProcessor(t, br)
	defer cleanup()

	err := p.Buy(buyRequest(listingId))
	if err == nil {
		t.Fatal("expected buy with insufficient prepaid to be rejected, got nil error")
	}
	if emitter.called {
		t.Error("saga emitted for an under-funded buy; expected no emission")
	}
	if !br.called {
		t.Error("expected the balance reader to be consulted")
	}
	if br.askedAcctId != buyerAcctForBuy {
		t.Errorf("balance read for account %d, want buyer account %d", br.askedAcctId, buyerAcctForBuy)
	}
}

// TestBuyEmitsSettlePurchase asserts a sufficiently-funded buy emits a single
// MtsSettlePurchase step whose payload carries the debit-first money-mover fields:
// buyer prepaid debited by markedUpPrice (1600 = ceil(1000*1.10)+500 base), seller
// points credited by listValue (1000), commission (600) never credited, and the
// item routed to the buyer's holding (never inventory) via the move-to-holding the
// expansion performs.
func TestBuyEmitsSettlePurchase(t *testing.T) {
	br := &stubBalanceReader{prepaid: 1600} // exactly the marked-up price
	p, emitter, _, listingId, cleanup := newBuyProcessor(t, br)
	defer cleanup()

	if err := p.Buy(buyRequest(listingId)); err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if !emitter.called {
		t.Fatal("expected a saga to be emitted for a funded buy")
	}

	sg := emitter.saga
	if sg.SagaType != saga.MtsOperation {
		t.Errorf("saga type = %q, want %q", sg.SagaType, saga.MtsOperation)
	}
	if len(sg.Steps) != 1 {
		t.Fatalf("expected 1 step (MtsSettlePurchase composite), got %d", len(sg.Steps))
	}
	if sg.Steps[0].Action != sharedsaga.MtsSettlePurchase {
		t.Errorf("step[0] action = %q, want %q", sg.Steps[0].Action, sharedsaga.MtsSettlePurchase)
	}

	sp, ok := sg.Steps[0].Payload.(sharedsaga.MtsSettlePurchasePayload)
	if !ok {
		t.Fatalf("step[0] payload type = %T, want MtsSettlePurchasePayload", sg.Steps[0].Payload)
	}

	// Debit-first money-mover fields (the orchestrator expands this into
	// award_currency(buyer prepaid -markedUp) FIRST, award_currency(seller points
	// +listValue), then mts_move_listing_to_holding(buyer)).
	if sp.ListingId != listingId {
		t.Errorf("settle listingId = %s, want %s", sp.ListingId, listingId)
	}
	if sp.BuyerId != buyerForBuy {
		t.Errorf("settle buyerId = %d, want %d", sp.BuyerId, buyerForBuy)
	}
	if sp.BuyerAccountId != buyerAcctForBuy {
		t.Errorf("settle buyerAccountId = %d, want %d", sp.BuyerAccountId, buyerAcctForBuy)
	}
	if sp.SellerId != sellerForBuy {
		t.Errorf("settle sellerId = %d, want %d (from the listing row)", sp.SellerId, sellerForBuy)
	}
	if sp.SellerAccountId != sellerAcctForBuy {
		t.Errorf("settle sellerAccountId = %d, want %d (caller-supplied)", sp.SellerAccountId, sellerAcctForBuy)
	}
	if sp.ListValue != 1000 {
		t.Errorf("settle listValue = %d, want 1000 (the seller credit)", sp.ListValue)
	}
	if sp.MarkedUpPrice != 1600 {
		t.Errorf("settle markedUpPrice = %d, want 1600 (ceil(listValue*1.10)+500 base)", sp.MarkedUpPrice)
	}
	// Commission is the sink: it is markedUp - listValue and is never credited to
	// anyone. The payload only ever carries listValue as the seller credit; assert
	// the commission is exactly the un-credited difference.
	if commission := sp.MarkedUpPrice - sp.ListValue; commission != 600 {
		t.Errorf("commission (markedUp - listValue) = %d, want 600 (never credited)", commission)
	}
	if sp.WorldId != 0 {
		t.Errorf("settle worldId = %d, want 0", sp.WorldId)
	}

	// Timeout MUST be set (never default) and scaled for N=3 (the MtsSettlePurchase
	// composite expands to award_currency x2 + mts_move_listing_to_holding).
	if sg.Timeout <= 0 {
		t.Errorf("saga timeout = %d, want a positive explicit timeout", sg.Timeout)
	}
	got := time.Duration(sg.Timeout) * time.Millisecond
	// base 10s + perStep 1s * 3 = 13s
	if got < 13*time.Second {
		t.Errorf("saga timeout = %s, want at least the N=3 scaled budget (>= 13s)", got)
	}
}

// seedActiveAuctionForBuyNow persists an active AUCTION listing with listValue
// 1000, a buyNowPrice of 5000, and commissionRate 0.10 and returns its id. It is
// the fixture for the buy-now (BUY_AUCTION_IMM) path: the immediate-buyout price
// is the buy-now price, not the auction's starting/list value.
func seedActiveAuctionForBuyNow(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	buyNow := uint32(5000)
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerForBuy).
		SetId(id).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetBuyNowPrice(&buyNow).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("build seed auction: %v", err)
	}
	if _, err := listing.CreateListing(db, m); err != nil {
		t.Fatalf("create seed auction: %v", err)
	}
	return id
}

// TestBuyNowChargesBuyNowPrice asserts a buy-now (BuyNow=true) against an active
// auction charges the buy-now price (5000) marked up (ceil(5000*1.10)+500 = 6000)
// and credits the seller the buy-now price, NOT the auction's listValue (1000).
func TestBuyNowChargesBuyNowPrice(t *testing.T) {
	br := &stubBalanceReader{prepaid: 6000} // exactly the marked-up buy-now price
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration)
	ctx := test.CreateTestContext()
	emitter := &captureEmitter{}
	p := listing.NewProcessor(logger, ctx, db,
		listing.WithSagaEmitter(emitter),
		listing.WithBalanceReader(br),
	)
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	defer test.CleanupTestDB(t, db)
	listingId := seedActiveAuctionForBuyNow(t, db)

	req := buyRequest(listingId)
	req.BuyNow = true
	if err := p.Buy(req); err != nil {
		t.Fatalf("BuyNow: %v", err)
	}
	if !emitter.called {
		t.Fatal("expected a saga to be emitted for a funded buy-now")
	}
	sp, ok := emitter.saga.Steps[0].Payload.(sharedsaga.MtsSettlePurchasePayload)
	if !ok {
		t.Fatalf("step[0] payload type = %T, want MtsSettlePurchasePayload", emitter.saga.Steps[0].Payload)
	}
	if sp.MarkedUpPrice != 6000 {
		t.Errorf("buy-now markedUpPrice = %d, want 6000 (ceil(buyNow 5000 * 1.10)+500 base)", sp.MarkedUpPrice)
	}
	if sp.ListValue != 5000 {
		t.Errorf("buy-now seller credit = %d, want 5000 (the buy-now price)", sp.ListValue)
	}
}

// TestBuyNowRejectsNonAuction asserts BuyNow against a fixed-price listing (no
// buy-now price) is rejected with no saga emission.
func TestBuyNowRejectsNonAuction(t *testing.T) {
	br := &stubBalanceReader{prepaid: 1_000_000}
	p, emitter, _, listingId, cleanup := newBuyProcessor(t, br)
	defer cleanup()

	req := buyRequest(listingId) // a fixed-price listing with no buy-now price
	req.BuyNow = true
	if err := p.Buy(req); err == nil {
		t.Fatal("expected buy-now against a non-buy-now listing to be rejected")
	}
	if emitter.called {
		t.Error("saga emitted for an invalid buy-now; expected no emission")
	}
}

// TestBuyRejectsNonActiveListing asserts a buy against a listing that is not
// active (already sold/cancelled) is rejected with no saga emission.
func TestBuyRejectsNonActiveListing(t *testing.T) {
	br := &stubBalanceReader{prepaid: 1_000_000}
	p, emitter, db, listingId, cleanup := newBuyProcessor(t, br)
	defer cleanup()

	if _, err := listing.UpdateState(db, listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("pre-mark sold: %v", err)
	}

	if err := p.Buy(buyRequest(listingId)); err == nil {
		t.Fatal("expected buy against a non-active listing to be rejected")
	}
	if emitter.called {
		t.Error("saga emitted for a non-active listing; expected no emission")
	}
}

// TestBuyFailureSentinels pins the typed failure sentinels the Kafka consumer
// maps to client NoticeFailReason codes: insufficient prepaid and
// non-active-listing rejections must be errors.Is-matchable.
func TestBuyFailureSentinels(t *testing.T) {
	br := &stubBalanceReader{prepaid: 0}
	p, _, db, listingId, cleanup := newBuyProcessor(t, br)
	defer cleanup()

	if err := p.Buy(buyRequest(listingId)); !errors.Is(err, listing.ErrInsufficientPrepaid) {
		t.Fatalf("low-prepaid buy error = %v, want errors.Is ErrInsufficientPrepaid", err)
	}

	// Non-active listing: transition it out of active first.
	if _, err := listing.UpdateState(db, listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("transition: %v", err)
	}
	br.prepaid = 100000
	if err := p.Buy(buyRequest(listingId)); !errors.Is(err, listing.ErrListingUnavailable) {
		t.Fatalf("sold-listing buy error = %v, want errors.Is ErrListingUnavailable", err)
	}

	// Bid on a non-active listing maps the same way.
	err := p.PlaceBid(listing.BidRequest{WorldId: 0, ListingId: listingId, BidderId: 3, BidderAccountId: 3, Amount: 5000})
	if !errors.Is(err, listing.ErrListingUnavailable) {
		t.Fatalf("sold-listing bid error = %v, want errors.Is ErrListingUnavailable", err)
	}
}
