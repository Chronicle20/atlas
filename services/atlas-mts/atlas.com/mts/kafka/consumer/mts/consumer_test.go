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

func newCancelCommand(transactionId uuid.UUID, listingId uuid.UUID) mts.Command[mts.CancelListingCommandBody] {
	return mts.Command[mts.CancelListingCommandBody]{
		TransactionId: transactionId,
		Type:          mts.CommandCancelListing,
		Body: mts.CancelListingCommandBody{
			ListingId: listingId,
			WorldId:   0,
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
	seedActiveListing(t, db, ctx, listingId, sellerId)

	rp := &recordingProducer{}
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(transactionId, listingId))

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

// TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsNothing asserts the cancel
// handler that loses the cancel-vs-buy race (the listing is already not active)
// neither creates a seller holding nor emits a LISTING_CANCELLED event.
func TestCancelListing_RaceLoserCreatesNoHoldingAndEmitsNothing(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, wish.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	const sellerId = uint32(8880002)
	seedActiveListing(t, db, ctx, listingId, sellerId)

	// Simulate a concurrent buy winning the race: the listing is already sold.
	if _, err := listing.UpdateState(db.WithContext(ctx), listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate concurrent buy: %v", err)
	}

	rp := &recordingProducer{}
	handleCancelListing(rp.provider())(db)(l, ctx, newCancelCommand(transactionId, listingId))

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

	// no LISTING_CANCELLED event emitted
	if len(rp.events) != 0 {
		t.Fatalf("expected no events for race loser, got %d (%v)", len(rp.events), rp.events)
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
