package listing_test

import (
	"atlas-mts/bid"
	"atlas-mts/holding"
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

const (
	sellerForBid     = uint32(300)
	bidderForBid     = uint32(400)
	bidderAcctForBid = uint32(4000)
	priorBidder      = uint32(500)
	priorBidderAcct  = uint32(5000)
	sellerAcctForBid = uint32(3000)
)

// newBidProcessor builds a listing processor wired to a capturing saga emitter and
// seeds one active auction listing (listValue 1000 — the seller's BASE price
// under the base-price pricing model — commissionRate 0.10, minIncrement 100, no
// bids yet) and returns its id. The tenant config registry falls back to
// DefaultConfig() in tests (no live atlas-tenants), so commissionBase is 500 —
// MarkedUp(x) = ceil(x*1.10)+500 throughout this file.
func newBidProcessor(t *testing.T) (listing.Processor, *captureEmitter, *gorm.DB, uuid.UUID, func()) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration, bid.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	emitter := &captureEmitter{}
	p := listing.NewProcessor(logger, ctx, db, listing.WithSagaEmitter(emitter))
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	if err := db.Exec("DELETE FROM bids").Error; err != nil {
		t.Fatalf("reset bids: %v", err)
	}
	cleanup := func() { test.CleanupTestDB(t, db) }
	return p, emitter, db, seedActiveAuction(t, db), cleanup
}

// seedActiveAuction persists an active auction listing with listValue 1000,
// commissionRate 0.10, minIncrement 100, ending one hour out and with no bids.
func seedActiveAuction(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ends := time.Now().Add(1 * time.Hour)
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerForBid).
		SetId(id).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetMinIncrement(100).
		SetEndsAt(&ends).
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

func bidRequest(listingId uuid.UUID, bidderId uint32, acctId uint32, amount uint32) listing.BidRequest {
	return listing.BidRequest{
		WorldId:         0,
		ListingId:       listingId,
		BidderId:        bidderId,
		BidderAccountId: acctId,
		Amount:          amount,
	}
}

// findHeldBid returns the single held bid for a listing, failing if not exactly one.
func findBids(t *testing.T, db *gorm.DB, listingId uuid.UUID, state bid.State) []bid.Model {
	t.Helper()
	ctx := test.CreateTestContext()
	all, err := bid.NewProcessor(logrus.New(), ctx, db).GetByListingId(listingId)
	if err != nil {
		t.Fatalf("GetByListingId: %v", err)
	}
	var out []bid.Model
	for _, b := range all {
		if b.State() == state {
			out = append(out, b)
		}
	}
	return out
}

// TestPlaceBidRejectsBelowFloor asserts a first bid below the listValue floor is
// rejected with no bid recorded and no escrow saga emitted.
func TestPlaceBidRejectsBelowFloor(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	// First bid: floor is listValue (1000). 999 is below.
	if _, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 999)); err == nil {
		t.Fatal("expected first bid below the listValue floor to be rejected")
	}
	if emitter.called {
		t.Error("escrow saga emitted for a below-floor bid; expected none")
	}
	if held := findBids(t, db, listingId, bid.StateHeld); len(held) != 0 {
		t.Errorf("recorded %d held bids for a rejected bid, want 0", len(held))
	}
}

// TestPlaceBidRejectsBelowIncrement asserts a subsequent bid that does not clear
// currentBid + minIncrement is rejected.
func TestPlaceBidRejectsBelowIncrement(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	// First valid bid: 1000 (== floor). Records held + updates currentBid.
	if _, err := p.PlaceBid(bidRequest(listingId, priorBidder, priorBidderAcct, 1000)); err != nil {
		t.Fatalf("first bid: %v", err)
	}
	emitter.called = false

	// Second bid must clear 1000 + 100 = 1100. 1099 is too low.
	if _, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 1099)); err == nil {
		t.Fatal("expected a bid below currentBid+minIncrement to be rejected")
	}
	if emitter.called {
		t.Error("escrow saga emitted for a below-increment bid; expected none")
	}
	// Only the first bidder's held bid should remain.
	held := findBids(t, db, listingId, bid.StateHeld)
	if len(held) != 1 || held[0].BidderId() != priorBidder {
		t.Errorf("held bids = %+v, want exactly the prior bidder's", held)
	}
}

// TestPlaceBidEscrowsMarkedUp asserts a first valid bid records a held bid (with a
// fresh escrowTxnId, RAW base amount matching the listing's currentBid), updates
// the listing currentBid/highBidder, and emits a single MtsBidEscrow step holding
// the MARKED-UP bid amount (negative) — the bid is a BASE price under the new
// pricing model, so the escrow hold must match what the winner would owe at
// settle/buy-now: MarkedUp(1000, rate=0.10, base=500) = ceil(1000*1.10)+500 =
// 1600.
func TestPlaceBidEscrowsMarkedUp(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	if _, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 1000)); err != nil {
		t.Fatalf("PlaceBid: %v", err)
	}

	// Listing auction fields updated under the row.
	lm, err := p.GetById(listingId.String())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if lm.CurrentBid() != 1000 {
		t.Errorf("currentBid = %d, want 1000", lm.CurrentBid())
	}
	if lm.HighBidderId() != bidderForBid {
		t.Errorf("highBidderId = %d, want %d", lm.HighBidderId(), bidderForBid)
	}

	// A held bid recorded with a non-nil escrow txn id.
	held := findBids(t, db, listingId, bid.StateHeld)
	if len(held) != 1 {
		t.Fatalf("held bids = %d, want 1", len(held))
	}
	if held[0].Amount() != 1000 {
		t.Errorf("held bid amount = %d, want 1000 (the bid, same figure as the escrow)", held[0].Amount())
	}
	if held[0].EscrowTxnId() == uuid.Nil {
		t.Error("held bid escrowTxnId is nil; want a fresh uuid")
	}

	// Single MtsBidEscrow step holding the MARKED-UP bid amount (1600), negative —
	// the bid (1000) is a BASE price under the new pricing model.
	if !emitter.called {
		t.Fatal("expected an escrow saga to be emitted")
	}
	sg := emitter.saga
	if sg.SagaType != saga.MtsOperation {
		t.Errorf("saga type = %q, want %q", sg.SagaType, saga.MtsOperation)
	}
	if len(sg.Steps) != 1 {
		t.Fatalf("expected 1 escrow step, got %d", len(sg.Steps))
	}
	if sg.Steps[0].Action != sharedsaga.MtsBidEscrow {
		t.Errorf("step[0] action = %q, want %q", sg.Steps[0].Action, sharedsaga.MtsBidEscrow)
	}
	ep, ok := sg.Steps[0].Payload.(sharedsaga.MtsBidEscrowPayload)
	if !ok {
		t.Fatalf("payload type = %T, want MtsBidEscrowPayload", sg.Steps[0].Payload)
	}
	if ep.Amount != -1600 {
		t.Errorf("escrow amount = %d, want -1600 (MarkedUp(1000, rate=0.10, base=500): the bid is a BASE price)", ep.Amount)
	}
	if ep.BidderId != bidderForBid || ep.BidderAccountId != bidderAcctForBid {
		t.Errorf("escrow bidder = (%d,%d), want (%d,%d)", ep.BidderId, ep.BidderAccountId, bidderForBid, bidderAcctForBid)
	}
	if ep.TransactionId == uuid.Nil {
		t.Error("escrow transactionId is nil")
	}
	// Timeout scaled for N=1.
	if sg.Timeout <= 0 {
		t.Errorf("escrow saga timeout = %d, want positive", sg.Timeout)
	}
}

// TestPlaceBidOutbidReleasesPrior asserts an outbid emits a RELEASE escrow
// (positive, MARKED-UP amount) for the prior bidder and marks their bid released.
func TestPlaceBidOutbidReleasesPrior(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	// Prior high bid at 1000 (base). Marked-up hold: -MarkedUp(1000)=-1600.
	if _, err := p.PlaceBid(bidRequest(listingId, priorBidder, priorBidderAcct, 1000)); err != nil {
		t.Fatalf("prior bid: %v", err)
	}
	emitter.called = false

	// Outbid at 1200 (>= 1000 + 100, base). Marked-up hold: -MarkedUp(1200)=-1820.
	res, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 1200))
	if err != nil {
		t.Fatalf("outbid: %v", err)
	}
	// The BidResult reports the displaced bidder so the consumer can emit OUTBID and
	// record the outbid bidder's bid-lost history row (task-102 #1/#2).
	if !res.HadPrior {
		t.Error("outbid BidResult.HadPrior = false, want true")
	}
	if res.PreviousBidderId != priorBidder {
		t.Errorf("outbid BidResult.PreviousBidderId = %d, want %d", res.PreviousBidderId, priorBidder)
	}
	if res.PreviousBidAmount != 1000 {
		t.Errorf("outbid BidResult.PreviousBidAmount = %d, want 1000 (the prior raw bid)", res.PreviousBidAmount)
	}
	if res.ItemId == 0 || res.SellerId == 0 {
		t.Errorf("outbid BidResult missing item/seller: itemId=%d sellerId=%d", res.ItemId, res.SellerId)
	}

	// Prior bid marked released; new bid held.
	released := findBids(t, db, listingId, bid.StateReleased)
	if len(released) != 1 || released[0].BidderId() != priorBidder {
		t.Errorf("released bids = %+v, want exactly the prior bidder's", released)
	}
	held := findBids(t, db, listingId, bid.StateHeld)
	if len(held) != 1 || held[0].BidderId() != bidderForBid {
		t.Errorf("held bids = %+v, want exactly the new bidder's", held)
	}

	// The emitted saga(s) must include: a hold for the new bidder (-1820, i.e.
	// -MarkedUp(1200)) AND a release for the prior bidder (+1600, i.e.
	// +MarkedUp(1000)). The release marks the prior escrow freed — the exact
	// marked-up amount that was held for their bid.
	if !emitter.called {
		t.Fatal("expected an escrow saga on outbid")
	}
	var sawHold, sawRelease bool
	for _, sg := range emitter.sagas() {
		for _, st := range sg.Steps {
			if st.Action != sharedsaga.MtsBidEscrow {
				continue
			}
			ep := st.Payload.(sharedsaga.MtsBidEscrowPayload)
			switch {
			case ep.Amount == -1820 && ep.BidderId == bidderForBid:
				sawHold = true
			case ep.Amount == 1600 && ep.BidderId == priorBidder:
				sawRelease = true
			}
		}
	}
	if !sawHold {
		t.Error("expected a -1820 hold for the new bidder (MarkedUp(1200))")
	}
	if !sawRelease {
		t.Error("expected a +1600 release for the prior bidder (MarkedUp(1000))")
	}

	// Listing high bid advanced.
	lm, _ := p.GetById(listingId.String())
	if lm.CurrentBid() != 1200 || lm.HighBidderId() != bidderForBid {
		t.Errorf("listing high bid = (%d,%d), want (1200,%d)", lm.CurrentBid(), lm.HighBidderId(), bidderForBid)
	}
}

// TestCancelReleasesHeldBidEscrow asserts cancelling an auction with an open high
// bid releases the held bidder's escrow and marks their bid released — otherwise
// the bidder's NX stays held forever. The escrow HELD the MARKED-UP amount at bid
// time (MarkedUp(1000, rate=0.10, base=500) = 1600), so the release must reverse
// that exact hold — the release Amount is 1600, not the raw base bid.
func TestCancelReleasesHeldBidEscrow(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	// A live high bid at 1000 (base). Marked-up hold: -MarkedUp(1000)=-1600.
	if _, err := p.PlaceBid(bidRequest(listingId, priorBidder, priorBidderAcct, 1000)); err != nil {
		t.Fatalf("bid: %v", err)
	}
	emitter.called = false

	// Cancelling the auction must release the held bidder's escrow (+1600) and mark
	// their bid released — otherwise the bidder's NX stays held forever.
	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !res.Won {
		t.Fatal("cancel did not win the active->holding transition")
	}
	if res.HeldBidderId != priorBidder || res.HeldBidAmount != 1600 || res.HeldBidderAccountId != priorBidderAcct {
		t.Errorf("cancel held-bid = (bidder=%d acct=%d amt=%d), want (%d,%d,1600)",
			res.HeldBidderId, res.HeldBidderAccountId, res.HeldBidAmount, priorBidder, priorBidderAcct)
	}

	if released := findBids(t, db, listingId, bid.StateReleased); len(released) != 1 || released[0].BidderId() != priorBidder {
		t.Errorf("released bids = %+v, want exactly the held bidder's", released)
	}
	if held := findBids(t, db, listingId, bid.StateHeld); len(held) != 0 {
		t.Errorf("held bids = %+v, want none after cancel", held)
	}

	if !emitter.called {
		t.Fatal("expected an escrow-release saga on cancel")
	}
	var sawRelease bool
	for _, sg := range emitter.sagas() {
		for _, st := range sg.Steps {
			if st.Action != sharedsaga.MtsBidEscrow {
				continue
			}
			ep := st.Payload.(sharedsaga.MtsBidEscrowPayload)
			if ep.Amount == 1600 && ep.BidderId == priorBidder && ep.BidderAccountId == priorBidderAcct {
				sawRelease = true
			}
		}
	}
	if !sawRelease {
		t.Error("expected a +1600 escrow release for the held bidder on cancel (MarkedUp(1000))")
	}
}

// TestReleaseHighBidEscrowRefundsBuyNowLoser asserts that when an auction with a
// live high bid is settled by a BUY-NOW (the buyer takes the item; the high bidder
// never wins), ReleaseHighBidEscrow refunds the high bidder's escrow
// (+MarkedUp(1000)=1600) and marks their bid released — so the bidder's NX is not
// lost to the system. A second call is a no-op (the idempotent replay guard).
func TestReleaseHighBidEscrowRefundsBuyNowLoser(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	if _, err := p.PlaceBid(bidRequest(listingId, priorBidder, priorBidderAcct, 1000)); err != nil {
		t.Fatalf("bid: %v", err)
	}
	emitter.called = false

	if err := p.ReleaseHighBidEscrow(0, listingId); err != nil {
		t.Fatalf("release: %v", err)
	}
	if released := findBids(t, db, listingId, bid.StateReleased); len(released) != 1 || released[0].BidderId() != priorBidder {
		t.Errorf("released bids = %+v, want exactly the held bidder's", released)
	}
	if held := findBids(t, db, listingId, bid.StateHeld); len(held) != 0 {
		t.Errorf("held bids = %+v, want none after buy-now release", held)
	}
	var sawRelease bool
	for _, sg := range emitter.sagas() {
		for _, st := range sg.Steps {
			if st.Action != sharedsaga.MtsBidEscrow {
				continue
			}
			ep := st.Payload.(sharedsaga.MtsBidEscrowPayload)
			if ep.Amount == 1600 && ep.BidderId == priorBidder && ep.BidderAccountId == priorBidderAcct {
				sawRelease = true
			}
		}
	}
	if !sawRelease {
		t.Error("expected a +1600 escrow release for the high bidder on buy-now (MarkedUp(1000))")
	}

	// Idempotent: the bid is already released, so a replayed settle move does not
	// double-refund.
	emitter.called = false
	if err := p.ReleaseHighBidEscrow(0, listingId); err != nil {
		t.Fatalf("second release: %v", err)
	}
	if emitter.called {
		t.Error("a second ReleaseHighBidEscrow must be a no-op (bid already released)")
	}
}

// TestSettleAuctionAtExpiryCreditsSellerNoDoubleDebit asserts the settle-at-expiry
// path for an auction WITH a high bidder credits the seller points
// (+winningBid, the BASE amount — no UnMarkUp), moves custody to the winner, marks
// the winning bid won, and does NOT re-debit the winner (the debit happened,
// marked up, at bid time). The emitted saga MUST NOT contain any negative
// AwardCurrency / MtsBidEscrow for the winner.
//
// The winning bid is the base price 1000; the seller nets it AS-IS (1000) — the
// commission was realised on the buyer's escrow hold (MarkedUp(1000)=1600) at bid
// time, not inverted here.
func TestSettleAuctionAtExpiryCreditsSellerNoDoubleDebit(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	if _, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 1000)); err != nil {
		t.Fatalf("bid: %v", err)
	}
	emitter.reset()

	res, err := p.SettleAuction(listing.SettleRequest{
		ListingId:       listingId,
		WorldId:         0,
		WinnerId:        bidderForBid,
		WinnerAccountId: bidderAcctForBid,
		SellerAccountId: sellerAcctForBid,
	})
	if err != nil {
		t.Fatalf("SettleAuction: %v", err)
	}
	if !res.HadWinner {
		t.Fatal("expected SettleAuction to report a winner")
	}

	// Winning bid marked won.
	won := findBids(t, db, listingId, bid.StateWon)
	if len(won) != 1 || won[0].BidderId() != bidderForBid {
		t.Errorf("won bids = %+v, want the winner's", won)
	}

	// Saga: seller-credit (+winningBid points, base amount) + move-to-winner-holding.
	// NO buyer debit.
	if !emitter.called {
		t.Fatal("expected a settle saga")
	}
	sg := emitter.saga
	var sawSellerCredit, sawMove bool
	for _, st := range sg.Steps {
		switch st.Action {
		case sharedsaga.AwardCurrency:
			ap := st.Payload.(sharedsaga.AwardCurrencyPayload)
			if ap.Amount < 0 {
				t.Errorf("settle contains a NEGATIVE AwardCurrency (%+v) — winner double-debited", ap)
			}
			if ap.Amount == 1000 && ap.AccountId == sellerAcctForBid {
				sawSellerCredit = true
			}
		case sharedsaga.MtsBidEscrow:
			ep := st.Payload.(sharedsaga.MtsBidEscrowPayload)
			t.Errorf("settle contains an MtsBidEscrow (%+v) — winner re-touched at settle", ep)
		case sharedsaga.MtsMoveListingToHolding:
			mp := st.Payload.(sharedsaga.MtsMoveListingToHoldingPayload)
			if mp.BuyerId == bidderForBid && mp.ListingId == listingId {
				sawMove = true
			}
		case sharedsaga.MtsSettlePurchase:
			t.Error("settle reused MtsSettlePurchase — that DEBITS the winner (double-debit)")
		}
	}
	if !sawSellerCredit {
		t.Error("expected a +1000 points credit to the seller (the base winning bid)")
	}
	if !sawMove {
		t.Error("expected a move-to-holding for the winner")
	}
}

// TestSettleAuctionTwiceCreditsSellerOnce is the regression guard for the
// double-credit money bug: the DB-driven expiration sweep discovers expired
// auctions by (state='active' AND ends_at<now). SettleAuction emits the
// seller-credit + move saga, but the listing only flips out of `active` later, in
// the async MtsMoveListingToHolding custody step. If a SECOND sweep tick fires
// before that async move completes, the listing is STILL active+expired and gets
// re-discovered — a naive settle would emit a SECOND seller credit, double-paying
// the seller. The fix transitions the listing active->settling SYNCHRONOUSLY under
// a CAS when the first settle emits, so the second settle's CAS loses and emits
// nothing. This test settles the SAME auction twice (move step deliberately not
// run between) and asserts exactly ONE seller credit was emitted.
func TestSettleAuctionTwiceCreditsSellerOnce(t *testing.T) {
	p, emitter, db, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	if _, err := p.PlaceBid(bidRequest(listingId, bidderForBid, bidderAcctForBid, 1000)); err != nil {
		t.Fatalf("bid: %v", err)
	}
	emitter.reset()

	settleReq := listing.SettleRequest{
		ListingId:       listingId,
		WorldId:         0,
		WinnerId:        bidderForBid,
		WinnerAccountId: bidderAcctForBid,
		SellerAccountId: sellerAcctForBid,
	}

	// First sweep tick: settle. Emits the seller-credit + move saga and flips the
	// listing out of the discovery set (active->settling).
	res1, err := p.SettleAuction(settleReq)
	if err != nil {
		t.Fatalf("first SettleAuction: %v", err)
	}
	if !res1.HadWinner {
		t.Fatal("expected the first settle to report a winner")
	}

	// The listing MUST no longer be discoverable as an expired active auction: the
	// settle synchronously moved it out of `active` so the next sweep tick cannot
	// re-discover it before the async move step runs.
	expired, err := listing.GetExpiredActive(time.Now().Add(2*time.Hour), 0)(db.WithContext(test.CreateTestContext()))()
	if err != nil {
		t.Fatalf("GetExpiredActive: %v", err)
	}
	for _, lm := range expired {
		if lm.Id() == listingId {
			t.Fatalf("listing %s still discoverable as expired-active after settle — a second sweep would re-settle it", listingId)
		}
	}

	// Second sweep tick BEFORE the async move step completes: settle the same row
	// again. The CAS must lose and emit nothing — no second seller credit.
	res2, err := p.SettleAuction(settleReq)
	if err != nil {
		t.Fatalf("second SettleAuction: %v", err)
	}
	if res2.HadWinner {
		t.Error("the second settle reported a winner — it should have lost the settle CAS and done nothing")
	}

	// Across BOTH settle calls, exactly ONE seller-points credit must have been
	// emitted. Two would double-pay the seller (the money bug).
	sellerCredits := 0
	for _, sg := range emitter.sagas() {
		for _, st := range sg.Steps {
			if st.Action == sharedsaga.AwardCurrency {
				ap := st.Payload.(sharedsaga.AwardCurrencyPayload)
				if ap.AccountId == sellerAcctForBid && ap.Amount == 1000 {
					sellerCredits++
				}
			}
		}
	}
	if sellerCredits != 1 {
		t.Errorf("seller credited %d times across two settle ticks, want exactly 1 (double-credit money bug)", sellerCredits)
	}
}

// TestSettleAuctionNoBidsReturnsToSeller asserts an auction that expires with NO
// bids returns the item to the SELLER holding (origin=expired) and emits no settle
// money-mover — i.e. it takes the existing Expire path, reported by HadWinner=false.
func TestSettleAuctionNoBidsReturnsToSeller(t *testing.T) {
	p, emitter, _, listingId, cleanup := newBidProcessor(t)
	defer cleanup()

	res, err := p.SettleAuction(listing.SettleRequest{
		ListingId: listingId,
		WorldId:   0,
	})
	if err != nil {
		t.Fatalf("SettleAuction (no bids): %v", err)
	}
	if res.HadWinner {
		t.Error("expected HadWinner=false for a no-bid auction")
	}
	if !res.Expired {
		t.Error("expected the no-bid auction to take the expire-to-seller path")
	}
	if emitter.called {
		t.Error("no money-mover saga should be emitted for a no-bid auction")
	}
}
