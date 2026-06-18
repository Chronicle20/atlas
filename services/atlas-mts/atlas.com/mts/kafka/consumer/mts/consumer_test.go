package mts

import (
	"atlas-mts/holding"
	"atlas-mts/kafka/message/mts"
	"atlas-mts/listing"
	"atlas-mts/test"
	"atlas-mts/wish"
	"context"
	"encoding/json"
	"sync"
	"testing"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// recordedEvent is a decoded MTS status event captured by the test producer.
type recordedEvent struct {
	transactionId uuid.UUID
	eventType     string
}

// recordingProducer is a test producer.Provider that decodes every emitted kafka
// message into a recordedEvent, so assertions can inspect the event type and
// transactionId without a live broker.
type recordingProducer struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (r *recordingProducer) provider() func(ctx context.Context) func(token string) kprod.MessageProducer {
	return func(ctx context.Context) func(token string) kprod.MessageProducer {
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
					r.events = append(r.events, recordedEvent{transactionId: ev.TransactionId, eventType: ev.Type})
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
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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
	if len(rp.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rp.events))
	}
	if rp.events[0].eventType != mts.StatusEventTypeListingCancelled {
		t.Fatalf("expected LISTING_CANCELLED, got %s", rp.events[0].eventType)
	}
	if rp.events[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, rp.events[0].transactionId)
	}
}

// TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsFailed asserts the cancel
// handler that loses the cancel-vs-buy race (the listing is already not active)
// creates no seller holding and emits LISTING_CANCEL_FAILED (so the channel writes
// CancelSaleItemFailed to the seller) rather than LISTING_CANCELLED.
func TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsFailed(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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

func TestRegisterWish_CreatesEntryAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	wishId := uuid.New()
	const characterId = uint32(9990001)
	const itemId = uint32(1302000)

	rp := &recordingProducer{}
	cmd := mts.Command[mts.RegisterWishCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandRegisterWish,
		Body: mts.RegisterWishCommandBody{
			WishId:      wishId,
			WorldId:     0,
			CharacterId: characterId,
			ItemId:      itemId,
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

	if len(rp.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rp.events))
	}
	if rp.events[0].eventType != mts.StatusEventTypeWishAdded {
		t.Fatalf("expected WISH_ADDED, got %s", rp.events[0].eventType)
	}
	if rp.events[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, rp.events[0].transactionId)
	}
}

func TestRemoveWish_DeletesEntryAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
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

	if len(rp.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rp.events))
	}
	if rp.events[0].eventType != mts.StatusEventTypeWishRemoved {
		t.Fatalf("expected WISH_REMOVED, got %s", rp.events[0].eventType)
	}
	if rp.events[0].transactionId != transactionId {
		t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, rp.events[0].transactionId)
	}
}
