package mts

import (
	"atlas-mts/bid"
	"atlas-mts/holding"
	"atlas-mts/kafka/message/mts"
	msgsaga "atlas-mts/kafka/message/saga"
	"atlas-mts/listing"
	"atlas-mts/test"
	"atlas-mts/transaction"
	"atlas-mts/wish"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// recordedEvent is a decoded MTS status event captured by the test producer.
type recordedEvent struct {
	transactionId uuid.UUID
	eventType     string
	reason        string
}

// recordingProducer is a test producer.Provider that decodes every emitted kafka
// message into a recordedEvent, so assertions can inspect the event type and
// transactionId without a live broker.
type recordingProducer struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (r *recordingProducer) provider() func(ctx context.Context) kprod.Provider {
	return func(ctx context.Context) kprod.Provider {
		return func(token string) kprod.MessageProducer {
			return func(p model.Provider[[]kafka.Message]) error {
				ms, err := p()
				if err != nil {
					return err
				}
				r.mu.Lock()
				defer r.mu.Unlock()
				for _, m := range ms {
					var ev mts.StatusEvent[json.RawMessage]
					if err := json.Unmarshal(m.Value, &ev); err != nil {
						return err
					}
					r.events = append(r.events, recordedEvent{transactionId: ev.TransactionId, eventType: ev.Type, reason: reasonFromBody(ev.Body)})
				}
				return nil
			}
		}
	}
}

// seedActiveListing persists an active listing row with a known snapshot so the
// cancel handler has something to move to a seller holding.
// seedActiveListing persists an active listing row with a known snapshot. The
// per-(tenant, world) serial is assigned by CreateListing (serial.Next) and read
// from the returned model, so callers address the row by its real nITCSN.
func seedActiveListing(t *testing.T, db *gorm.DB, ctx context.Context, listingId uuid.UUID, sellerId uint32) listing.Model {
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
	stored, err := listing.CreateListing(db.WithContext(ctx), m)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	return stored
}

// holdingsForOwner returns the (non-deleted) holdings owned by ownerId. The
// package's cache=shared in-memory DB leaks rows across tests, so per-owner
// filtering keeps the "exactly one holding" assertion isolated.
func holdingsForOwner(t *testing.T, db *gorm.DB, ctx context.Context, ownerId uint32) []holding.Model {
	t.Helper()
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

func newCancelCommand(transactionId uuid.UUID, serial uint32, sellerId uint32) mts.Command[mts.CancelListingCommandBody] {
	return mts.Command[mts.CancelListingCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandCancelListing,
		Body: mts.CancelListingCommandBody{
			WorldId:  0,
			Serial:   serial,
			SellerId: sellerId,
		},
	}
}

func TestCancelListing_MovesActiveToSellerHoldingAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, transaction.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	const sellerId = uint32(8880001)
	seeded := seedActiveListing(t, db, ctx, listingId, sellerId)

	rp := &recordingProducer{}
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(transactionId, seeded.Serial(), sellerId))

	// listing transitioned to cancelled
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateCancelled {
		t.Fatalf("expected listing state cancelled, got %s", stored.State())
	}

	// exactly one seller holding, origin cancelled, snapshot copied
	sellerHoldings := holdingsForOwner(t, db, ctx, sellerId)
	if len(sellerHoldings) != 1 {
		t.Fatalf("expected exactly 1 holding for seller %d, got %d", sellerId, len(sellerHoldings))
	}
	h := sellerHoldings[0]
	if h.Origin() != holding.OriginCancelled {
		t.Fatalf("expected origin cancelled, got %s", h.Origin())
	}
	if h.TemplateId() != 1302000 || h.Quantity() != 1 || h.WeaponAttack() != 17 || h.Slots() != 7 {
		t.Fatalf("holding snapshot not copied: tmpl=%d qty=%d watk=%d slots=%d", h.TemplateId(), h.Quantity(), h.WeaponAttack(), h.Slots())
	}

	// exactly one LISTING_CANCELLED event carrying the transactionId
	evts := allEvents(t, db, rp, transactionId)
	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}
	if evts[0].eventType != mts.StatusEventTypeListingCancelled {
		t.Fatalf("expected LISTING_CANCELLED, got %s", evts[0].eventType)
	}
	if evts[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, evts[0].transactionId)
	}

	// exactly one cancelled history row for the seller (task-102 #4).
	var cancelledRows int64
	if err := db.WithContext(ctx).Table("mts_transactions").
		Where("character_id = ? AND kind = ?", sellerId, transaction.KindCancelled).
		Count(&cancelledRows).Error; err != nil {
		t.Fatalf("count cancelled transactions: %v", err)
	}
	if cancelledRows != 1 {
		t.Fatalf("expected exactly 1 cancelled history row for seller %d, got %d", sellerId, cancelledRows)
	}
}

// TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsFailed asserts the cancel
// handler that loses the cancel-vs-buy race (the listing is already not active)
// creates no seller holding and emits LISTING_CANCEL_FAILED (so the channel writes
// CancelSaleItemFailed to the seller) rather than LISTING_CANCELLED.
func TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	const sellerId = uint32(8880002)
	seeded := seedActiveListing(t, db, ctx, listingId, sellerId)

	// Simulate a concurrent buy winning the race: the listing is already sold.
	if _, err := listing.UpdateState(db.WithContext(ctx), listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate concurrent buy: %v", err)
	}

	rp := &recordingProducer{}
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(transactionId, seeded.Serial(), sellerId))

	// listing remains sold (cancel did not clobber it)
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing to remain sold, got %s", stored.State())
	}

	// no seller holding created
	if got := len(holdingsForOwner(t, db, ctx, sellerId)); got != 0 {
		t.Fatalf("expected no holding for race-losing seller %d, got %d", sellerId, got)
	}

	// exactly one LISTING_CANCEL_FAILED event emitted
	if len(rp.events) != 1 {
		t.Fatalf("expected 1 event for race loser, got %d (%v)", len(rp.events), rp.events)
	}
	if rp.events[0].eventType != mts.StatusEventTypeListingCancelFailed {
		t.Fatalf("expected LISTING_CANCEL_FAILED, got %s", rp.events[0].eventType)
	}
}

// TestCancelListing_OwnerMismatch_EmitsFailed asserts a cancel whose SellerId does
// not match the listing's seller is rejected with LISTING_CANCEL_FAILED and leaves
// the listing active.
func TestCancelListing_OwnerMismatch_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	const sellerId = uint32(8880003)
	seeded := seedActiveListing(t, db, ctx, listingId, sellerId)

	rp := &recordingProducer{}
	// A different character (sellerId+999) attempts the cancel.
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(transactionId, seeded.Serial(), sellerId+999))

	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateActive {
		t.Fatalf("expected listing to remain active after owner mismatch, got %s", stored.State())
	}
	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeListingCancelFailed {
		t.Fatalf("expected 1 LISTING_CANCEL_FAILED, got %v", rp.events)
	}
}

// TestCancelListing_SerialUnresolved_EmitsFailed asserts a cancel whose serial does
// not resolve to any listing is rejected with LISTING_CANCEL_FAILED.
func TestCancelListing_SerialUnresolved_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(uuid.New(), 99999, 8880004))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeListingCancelFailed {
		t.Fatalf("expected 1 LISTING_CANCEL_FAILED for unresolved serial, got %v", rp.events)
	}
}

func newTakeHomeCommand(transactionId uuid.UUID, serial uint32, characterId uint32) mts.Command[mts.TakeHomeCommandBody] {
	return mts.Command[mts.TakeHomeCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandTakeHome,
		Body: mts.TakeHomeCommandBody{
			WorldId:       0,
			Serial:        serial,
			CharacterId:   characterId,
			InventoryType: 1,
			Slot:          0,
		},
	}
}

// seedHolding persists a holding row with a known owner. The per-(tenant, world)
// serial is assigned by CreateHolding (serial.Next) and read from the returned
// model, so callers address the row by its real nITCSN.
func seedHolding(t *testing.T, db *gorm.DB, ctx context.Context, holdingId uuid.UUID, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(test.TestTenantId, 0, ownerId).
		SetId(holdingId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	stored, err := holding.CreateHolding(db.WithContext(ctx), m)
	if err != nil {
		t.Fatalf("seed holding: %v", err)
	}
	return stored
}

// TestTakeHome_SerialUnresolved_EmitsFailed asserts a take-home whose serial does
// not resolve to any holding is rejected with TAKE_HOME_FAILED.
func TestTakeHome_SerialUnresolved_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	handleTakeHome(rp.provider())(db)(l, ctx, newTakeHomeCommand(uuid.New(), 88888, 7770001))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeTakeHomeFailed {
		t.Fatalf("expected 1 TAKE_HOME_FAILED for unresolved serial, got %v", rp.events)
	}
}

// TestTakeHome_OwnerMismatch_EmitsFailed asserts a take-home whose CharacterId does
// not match the holding's owner is rejected with TAKE_HOME_FAILED.
func TestTakeHome_OwnerMismatch_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	holdingId := uuid.New()
	const ownerId = uint32(7770002)
	seeded := seedHolding(t, db, ctx, holdingId, ownerId)

	rp := &recordingProducer{}
	handleTakeHome(rp.provider())(db)(l, ctx, newTakeHomeCommand(uuid.New(), seeded.Serial(), ownerId+999))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeTakeHomeFailed {
		t.Fatalf("expected 1 TAKE_HOME_FAILED for owner mismatch, got %v", rp.events)
	}
}

func newBuyCommand(transactionId uuid.UUID, serial uint32, buyerId uint32, buyNow bool) mts.Command[mts.BuyCommandBody] {
	return mts.Command[mts.BuyCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandBuy,
		Body: mts.BuyCommandBody{
			WorldId:        0,
			Serial:         serial,
			BuyerId:        buyerId,
			BuyerAccountId: buyerId + 1000,
			BuyNow:         buyNow,
		},
	}
}

// TestBuy_SerialUnresolved_EmitsFailed asserts a buy whose serial does not resolve
// to any listing is rejected with BUY_FAILED (so the channel writes BuyItemFailed).
func TestBuy_SerialUnresolved_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	handleBuy(rp.provider())(db)(l, ctx, newBuyCommand(uuid.New(), 77777, 6660001, false))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeBuyFailed {
		t.Fatalf("expected 1 BUY_FAILED for unresolved serial, got %v", rp.events)
	}
	if rp.events[0].reason != mts.FailReasonItemSold {
		t.Fatalf("BUY_FAILED reason = %q, want FailReasonItemSold", rp.events[0].reason)
	}
}

// TestBuy_NonActiveListing_EmitsFailed asserts a buy against a non-active listing
// (already sold) is rejected with BUY_FAILED. The serial resolves but the Buy
// processor rejects the non-active state before any balance read.
func TestBuy_NonActiveListing_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	listingId := uuid.New()
	const sellerId = uint32(6660002)
	seeded := seedActiveListing(t, db, ctx, listingId, sellerId)
	if _, err := listing.UpdateState(db.WithContext(ctx), listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate already-sold: %v", err)
	}

	rp := &recordingProducer{}
	handleBuy(rp.provider())(db)(l, ctx, newBuyCommand(uuid.New(), seeded.Serial(), 6660003, false))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeBuyFailed {
		t.Fatalf("expected 1 BUY_FAILED for non-active listing, got %v", rp.events)
	}
	if rp.events[0].reason != mts.FailReasonItemSold {
		t.Fatalf("BUY_FAILED reason = %q, want FailReasonItemSold", rp.events[0].reason)
	}
}

func newBidCommand(transactionId uuid.UUID, serial uint32, bidderId uint32, amount uint32) mts.Command[mts.PlaceBidCommandBody] {
	return mts.Command[mts.PlaceBidCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandPlaceBid,
		Body: mts.PlaceBidCommandBody{
			WorldId:         0,
			Serial:          serial,
			BidderId:        bidderId,
			BidderAccountId: bidderId + 1000,
			Amount:          amount,
		},
	}
}

// seedActiveAuction persists an active AUCTION listing with no prior bidder, so a
// first bid at listValue clears the floor and wins.
func seedActiveAuction(t *testing.T, db *gorm.DB, ctx context.Context, listingId uuid.UUID, sellerId uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
		SetId(listingId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetWeaponAttack(17).
		SetSlots(7).
		SetLevel(1).
		SetListValue(1000).
		SetCurrentBid(900).
		SetMinIncrement(100).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("onehand").
		Build()
	if err != nil {
		t.Fatalf("build auction: %v", err)
	}
	stored, err := listing.CreateListing(db.WithContext(ctx), m)
	if err != nil {
		t.Fatalf("seed auction: %v", err)
	}
	return stored
}

// outboxRowsOnTopic counts outbox rows enqueued for a given (env-token) topic.
// The env vars are unset in tests, so outbox.EmitProvider stores the raw token
// (e.g. "COMMAND_TOPIC_SAGA") as the row's topic.
func outboxRowsOnTopic(t *testing.T, db *gorm.DB, topic string) int {
	t.Helper()
	var n int64
	if err := db.Model(&outbox.Entity{}).Where("topic = ?", topic).Count(&n).Error; err != nil {
		t.Fatalf("count outbox rows on %s: %v", topic, err)
	}
	return int(n)
}

// TestPlaceBid_Success_RoutesEscrowSagaAndBidPlacedThroughOutbox pins the
// task-114 fix for the money-critical escrow path: a winning bid enqueues BOTH
// its BID_PLACED status event AND its cross-service escrow-hold saga command as
// outbox rows on the bid's transaction, and emits NOTHING on the direct producer.
// Pre-fix, wrapping listing.PlaceBid in the handler's outer tx pulled its escrow
// saga (which PlaceBid fires after its own write) onto the direct producer INSIDE
// the tx — so a rolled-back bid would orphan an escrow move against it once
// ExecuteTransaction is real (task-119). Routing the saga through the tx-bound
// outbox keeps it in lockstep with the commit.
func TestPlaceBid_Success_RoutesEscrowSagaAndBidPlacedThroughOutbox(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, bid.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	listingId := uuid.New()
	const sellerId = uint32(5551001)
	const bidderId = uint32(5551002)
	seeded := seedActiveAuction(t, db, ctx, listingId, sellerId)

	transactionId := uuid.New()
	rp := &recordingProducer{}
	handlePlaceBid(rp.provider())(db)(l, ctx, newBidCommand(transactionId, seeded.Serial(), bidderId, 1000))

	// Nothing on the DIRECT producer: a winning first bid has no failure ack, and
	// both BID_PLACED and the escrow saga are now transactional (outbox).
	if len(rp.events) != 0 {
		t.Fatalf("expected no direct-producer emits on a winning bid, got %v", rp.events)
	}

	// BID_PLACED landed in the outbox (scoped to the bid's transactionId; the
	// escrow saga carries its own escrowTxnId and is excluded here).
	evts := allEvents(t, db, rp, transactionId)
	if len(evts) != 1 || evts[0].eventType != mts.StatusEventTypeBidPlaced {
		t.Fatalf("expected exactly 1 BID_PLACED outbox event, got %v", evts)
	}

	// The escrow-hold saga command ALSO landed in the outbox (command topic), so
	// it publishes iff the bid commits — the money-critical fix.
	if got := outboxRowsOnTopic(t, db, msgsaga.EnvCommandTopic); got != 1 {
		t.Fatalf("expected exactly 1 escrow saga command in the outbox, got %d", got)
	}
}

// TestPlaceBid_SerialUnresolved_EmitsFailed asserts a bid whose serial does not
// resolve to any listing is rejected with BID_FAILED (so the channel writes
// BidAuctionFailed).
func TestPlaceBid_SerialUnresolved_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	handlePlaceBid(rp.provider())(db)(l, ctx, newBidCommand(uuid.New(), 66666, 5550001, 2000))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeBidFailed {
		t.Fatalf("expected 1 BID_FAILED for unresolved serial, got %v", rp.events)
	}
}

// TestPlaceBid_NonAuctionListing_EmitsFailed asserts a bid against a fixed-price
// (non-auction) listing is rejected with BID_FAILED. The serial resolves but the
// PlaceBid processor rejects the non-auction sale type.
func TestPlaceBid_NonAuctionListing_EmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	listingId := uuid.New()
	const sellerId = uint32(5550002)
	// seedActiveListing seeds a FIXED-price listing; bidding on it must fail.
	seeded := seedActiveListing(t, db, ctx, listingId, sellerId)

	rp := &recordingProducer{}
	handlePlaceBid(rp.provider())(db)(l, ctx, newBidCommand(uuid.New(), seeded.Serial(), 5550003, 2000))

	if len(rp.events) != 1 || rp.events[0].eventType != mts.StatusEventTypeBidFailed {
		t.Fatalf("expected 1 BID_FAILED for non-auction listing, got %v", rp.events)
	}
}

func TestRegisterWish_CreatesEntryAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	wishId := uuid.New()
	const characterId = uint32(9990001)
	const itemId = uint32(1302000)
	const price = uint32(12345)

	rp := &recordingProducer{}
	cmd := mts.Command[mts.RegisterWishCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandRegisterWish,
		Body: mts.RegisterWishCommandBody{
			WishId:      wishId,
			WorldId:     0,
			CharacterId: characterId,
			ItemId:      itemId,
			Price:       price,
		},
	}
	handleRegisterWish(rp.provider())(db)(l, ctx, cmd)

	// the wish row was created with the carried id
	stored, err := wish.GetById(wishId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("expected wish row created, got error: %v", err)
	}
	if stored.CharacterId() != characterId || stored.ItemId() != itemId {
		t.Fatalf("wish not persisted: char=%d item=%d", stored.CharacterId(), stored.ItemId())
	}
	if stored.Price() != price {
		t.Fatalf("wish price not persisted: want %d got %d", price, stored.Price())
	}

	evts := allEvents(t, db, rp, transactionId)
	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}
	if evts[0].eventType != mts.StatusEventTypeWishAdded {
		t.Fatalf("expected WISH_ADDED, got %s", evts[0].eventType)
	}
	if evts[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, evts[0].transactionId)
	}
}

func TestRemoveWish_DeletesEntryAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration, outbox.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	// seed a wish row to remove
	wishId := uuid.New()
	const characterId = uint32(9990002)
	m, err := wish.NewBuilder(test.TestTenantId, characterId, 1302000).
		SetId(wishId).
		Build()
	if err != nil {
		t.Fatalf("build wish: %v", err)
	}
	if _, err := wish.CreateWish(db.WithContext(ctx), m); err != nil {
		t.Fatalf("seed wish: %v", err)
	}

	rp := &recordingProducer{}
	transactionId := uuid.New()
	cmd := mts.Command[mts.RemoveWishCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandRemoveWish,
		Body:          mts.RemoveWishCommandBody{WishId: wishId, WorldId: 0},
	}
	handleRemoveWish(rp.provider())(db)(l, ctx, cmd)

	// the wish row is gone
	if _, err := wish.GetById(wishId.String())(db.WithContext(ctx))(); err == nil {
		t.Fatalf("expected wish deleted after remove")
	}

	evts := allEvents(t, db, rp, transactionId)
	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}
	if evts[0].eventType != mts.StatusEventTypeWishRemoved {
		t.Fatalf("expected WISH_REMOVED, got %s", evts[0].eventType)
	}
	if evts[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, evts[0].transactionId)
	}
}

// allEvents merges the two sinks an MTS handler now emits to: the DIRECT
// producer captured by rp (failure-path *_FAILED acks) and the transactional
// outbox rows the migrated SUCCESS paths enqueue (task-114 — LISTING_CANCELLED,
// WISH_ADDED, WISH_REMOVED, BID_PLACED, OUTBID). Both decode to the same status
// envelope, so assertions cover the whole emit surface after the migration.
//
// Results are scoped to txId: the package's cache=shared in-memory DB leaks
// outbox_entries rows across sibling tests (unlike the per-test recordingProducer),
// so filtering on the test's transactionId isolates the assertion. Rows are read
// in id order to mirror publish order.
func allEvents(t *testing.T, db *gorm.DB, rp *recordingProducer, txId uuid.UUID) []recordedEvent {
	t.Helper()
	var out []recordedEvent
	rp.mu.Lock()
	for _, e := range rp.events {
		if e.transactionId == txId {
			out = append(out, e)
		}
	}
	rp.mu.Unlock()
	var rows []outbox.Entity
	if err := db.Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("read outbox rows: %v", err)
	}
	for _, r := range rows {
		var ev mts.StatusEvent[json.RawMessage]
		if err := json.Unmarshal(r.MessageValue, &ev); err != nil {
			t.Fatalf("decode outbox row: %v", err)
		}
		if ev.TransactionId != txId {
			continue
		}
		out = append(out, recordedEvent{transactionId: ev.TransactionId, eventType: ev.Type, reason: reasonFromBody(ev.Body)})
	}
	return out
}

// reasonFromBody extracts the optional semantic "reasonKey" from a raw status-event
// body (empty when absent), so failure tests can assert the resolved client
// NoticeFailReason key without per-event typed decoding.
func reasonFromBody(raw json.RawMessage) string {
	var b struct {
		ReasonKey string `json:"reasonKey"`
	}
	_ = json.Unmarshal(raw, &b)
	return b.ReasonKey
}

// TestFailReasonMapping pins the error -> client NoticeFailReason code mapping
// (IDA-verified codes; see the NoticeFailReason* docs in the message package).
func TestFailReasonMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"insufficient prepaid", listing.ErrInsufficientPrepaid, mts.FailReasonNotEnoughNX},
		{"wrapped insufficient prepaid", fmt.Errorf("ctx: %w", listing.ErrInsufficientPrepaid), mts.FailReasonNotEnoughNX},
		{"listing unavailable", listing.ErrListingUnavailable, mts.FailReasonItemSold},
		{"wrapped unavailable", fmt.Errorf("ctx: %w", listing.ErrListingUnavailable), mts.FailReasonItemSold},
		{"record not found", gorm.ErrRecordNotFound, mts.FailReasonItemSold},
		{"anything else", errors.New("boom"), mts.FailReasonGeneric},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := failReasonFor(c.err); got != c.want {
				t.Fatalf("failReasonFor(%v) = %q, want %q", c.err, got, c.want)
			}
		})
	}
}
