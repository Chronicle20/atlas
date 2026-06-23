package custody

import (
	"atlas-mts/holding"
	"atlas-mts/kafka/message/custody"
	mtsmsg "atlas-mts/kafka/message/mts"
	"atlas-mts/listing"
	"atlas-mts/test"
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

// recordedEvent is a decoded custody status ack captured by the test producer.
type recordedEvent struct {
	transactionId uuid.UUID
	eventType     string
}

// recordingProducer is a test producer.Provider that decodes every emitted
// kafka message into a recordedEvent, so assertions can inspect the ack type
// and transactionId without a live broker.
type recordingProducer struct {
	mu     sync.Mutex
	events []recordedEvent
}

// provider returns the two-level per-context producer factory the handler
// expects (func(ctx) func(token) MessageProducer), matching the shape of
// producer.ProviderImpl(l). Every emitted message is decoded and recorded.
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
					var ev custody.StatusEvent[json.RawMessage]
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

// eventsOfType filters recorded events by their event-type string so assertions
// can isolate one event family (e.g. MOVED acks vs LISTING_SOLD notices) when a
// success path emits more than one.
func eventsOfType(events []recordedEvent, eventType string) []recordedEvent {
	var out []recordedEvent
	for _, e := range events {
		if e.eventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

func newAcceptCommand(transactionId uuid.UUID, listingId uuid.UUID) custody.Command[custody.AcceptToMtsListingCommandBody] {
	return custody.Command[custody.AcceptToMtsListingCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandAcceptToMtsListing,
		Body: custody.AcceptToMtsListingCommandBody{
			ListingId:      listingId,
			WorldId:        0,
			SellerId:       42,
			SellerName:     "Seller",
			SaleType:       string(listing.SaleTypeFixed),
			TemplateId:     1302000,
			Quantity:       1,
			WeaponAttack:   17,
			Slots:          7,
			Level:          1,
			ListValue:      1000,
			CommissionRate: 0.10,
			Category:       "equip",
			SubCategory:    "onehand",
			MinIncrement:   0,
		},
	}
}

func TestAcceptToMtsListing_CreatesListingAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	transactionId := uuid.New()
	listingId := uuid.New()
	cmd := newAcceptCommand(transactionId, listingId)

	handleAcceptToMtsListing(rp.provider())(db)(l, ctx, cmd)

	// the listing row was created with the carried id, in active state, snapshot persisted
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("expected listing row created, got error: %v", err)
	}
	if stored.Id() != listingId {
		t.Fatalf("expected listing id %s, got %s", listingId, stored.Id())
	}
	if stored.State() != listing.StateActive {
		t.Fatalf("expected state active, got %s", stored.State())
	}
	if stored.TemplateId() != 1302000 || stored.Quantity() != 1 || stored.WeaponAttack() != 17 {
		t.Fatalf("snapshot not persisted: tmpl=%d qty=%d watk=%d", stored.TemplateId(), stored.Quantity(), stored.WeaponAttack())
	}
	if stored.ListValue() != 1000 || stored.CommissionRate() != 0.10 || stored.Category() != "equip" {
		t.Fatalf("sale params not persisted: lv=%d rate=%v cat=%s", stored.ListValue(), stored.CommissionRate(), stored.Category())
	}

	// exactly one ACCEPTED ack (drives the saga) + one LISTING_CREATED (drives the
	// channel's RegisterSaleEntryDone), both carrying the same transactionId.
	accepted := eventsOfType(rp.events, custody.StatusEventTypeAccepted)
	if len(accepted) != 1 {
		t.Fatalf("expected 1 ACCEPTED ack, got %d (all: %v)", len(accepted), rp.events)
	}
	if accepted[0].transactionId != transactionId {
		t.Fatalf("ACCEPTED ack transactionId mismatch: want %s got %s", transactionId, accepted[0].transactionId)
	}
	created := eventsOfType(rp.events, mtsmsg.StatusEventTypeListingCreated)
	if len(created) != 1 {
		t.Fatalf("expected 1 LISTING_CREATED event, got %d (all: %v)", len(created), rp.events)
	}
	if created[0].transactionId != transactionId {
		t.Fatalf("LISTING_CREATED transactionId mismatch: want %s got %s", transactionId, created[0].transactionId)
	}
}

func TestAcceptToMtsListing_ReplayIsNoOpAndReacks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	transactionId := uuid.New()
	listingId := uuid.New()
	cmd := newAcceptCommand(transactionId, listingId)

	// first delivery
	handleAcceptToMtsListing(rp.provider())(db)(l, ctx, cmd)
	// replayed delivery (same listingId / transactionId)
	handleAcceptToMtsListing(rp.provider())(db)(l, ctx, cmd)

	// exactly one listing row with that id (no duplicate created)
	all, err := listing.GetAll()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("GetAll error: %v", err)
	}
	count := 0
	for _, m := range all {
		if m.Id() == listingId {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 listing row for id %s after replay, got %d (total rows=%d)", listingId, count, len(all))
	}

	// both deliveries re-emitted the ACCEPTED ack + the LISTING_CREATED notice with
	// the same transactionId (a replayed LISTING_CREATED is a harmless idempotent
	// seller notice, mirroring the replayed LISTING_SOLD on the move handler).
	accepted := eventsOfType(rp.events, custody.StatusEventTypeAccepted)
	if len(accepted) != 2 {
		t.Fatalf("expected 2 ACCEPTED acks (original + replay), got %d (all: %v)", len(accepted), rp.events)
	}
	created := eventsOfType(rp.events, mtsmsg.StatusEventTypeListingCreated)
	if len(created) != 2 {
		t.Fatalf("expected 2 LISTING_CREATED events (original + replay), got %d (all: %v)", len(created), rp.events)
	}
	for _, ev := range append(accepted, created...) {
		if ev.transactionId != transactionId {
			t.Fatalf("event transactionId mismatch: want %s got %s", transactionId, ev.transactionId)
		}
	}
}

// seedActiveListing persists an active listing row with a known snapshot so the
// move handler has something to move to a buyer holding.
func seedActiveListing(t *testing.T, db *gorm.DB, ctx context.Context, listingId uuid.UUID) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, 42).
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

// holdingsForBuyer returns the (non-deleted) holdings owned by buyerId. The
// package's cache=shared in-memory DB leaks rows across tests, so per-buyer
// filtering keeps the "exactly one holding" assertion isolated.
func holdingsForBuyer(t *testing.T, db *gorm.DB, ctx context.Context, buyerId uint32) []holding.Model {
	t.Helper()
	all, err := holding.GetAll()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("holding GetAll: %v", err)
	}
	var out []holding.Model
	for _, m := range all {
		if m.OwnerId() == buyerId {
			out = append(out, m)
		}
	}
	return out
}

func newMoveCommand(transactionId uuid.UUID, listingId uuid.UUID, buyerId uint32) custody.Command[custody.MtsMoveListingToHoldingCommandBody] {
	return custody.Command[custody.MtsMoveListingToHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandMtsMoveListingToHolding,
		Body: custody.MtsMoveListingToHoldingCommandBody{
			ListingId: listingId,
			BuyerId:   buyerId,
			WorldId:   0,
		},
	}
}

func TestMtsMoveListingToHolding_MarksSoldCreatesHoldingAndAcks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	// The package uses a cache=shared in-memory DB, so rows leak across tests;
	// scope holding assertions to this test's unique buyer id.
	const buyerId = uint32(7770001)
	seedActiveListing(t, db, ctx, listingId)

	rp := &recordingProducer{}
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, newMoveCommand(transactionId, listingId, buyerId))

	// listing marked sold
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing state sold, got %s", stored.State())
	}

	// exactly one holding for the buyer, origin purchased, snapshot copied
	buyerHoldings := holdingsForBuyer(t, db, ctx, buyerId)
	if len(buyerHoldings) != 1 {
		t.Fatalf("expected exactly 1 holding for buyer %d, got %d", buyerId, len(buyerHoldings))
	}
	h := buyerHoldings[0]
	if h.Origin() != holding.OriginPurchased {
		t.Fatalf("expected origin purchased, got %s", h.Origin())
	}
	if h.TemplateId() != 1302000 || h.Quantity() != 1 || h.WeaponAttack() != 17 || h.Slots() != 7 {
		t.Fatalf("holding snapshot not copied: tmpl=%d qty=%d watk=%d slots=%d", h.TemplateId(), h.Quantity(), h.WeaponAttack(), h.Slots())
	}

	// exactly one MOVED ack (drives the saga) + one LISTING_SOLD (drives the
	// channel's BuyItemDone), both carrying the transactionId.
	moved := eventsOfType(rp.events, custody.StatusEventTypeMoved)
	if len(moved) != 1 {
		t.Fatalf("expected 1 MOVED ack, got %d (all: %v)", len(moved), rp.events)
	}
	if moved[0].transactionId != transactionId {
		t.Fatalf("ack transactionId mismatch: want %s got %s", transactionId, moved[0].transactionId)
	}
	sold := eventsOfType(rp.events, mtsmsg.StatusEventTypeListingSold)
	if len(sold) != 1 {
		t.Fatalf("expected 1 LISTING_SOLD event, got %d (all: %v)", len(sold), rp.events)
	}
	if sold[0].transactionId != transactionId {
		t.Fatalf("LISTING_SOLD transactionId mismatch: want %s got %s", transactionId, sold[0].transactionId)
	}
}

func TestMtsMoveListingToHolding_ReplayCreatesNoSecondHoldingAndReacks(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	// Unique buyer id so the shared in-memory DB's leaked rows don't pollute the
	// per-buyer count.
	const buyerId = uint32(7770002)
	seedActiveListing(t, db, ctx, listingId)

	rp := &recordingProducer{}
	cmd := newMoveCommand(transactionId, listingId, buyerId)

	// first delivery
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, cmd)
	// replayed delivery (same listing/buyer/transaction)
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, cmd)

	// still exactly one holding for this buyer after replay (no second copy granted)
	if got := len(holdingsForBuyer(t, db, ctx, buyerId)); got != 1 {
		t.Fatalf("expected exactly 1 holding for buyer %d after replay, got %d", buyerId, got)
	}

	// listing remains sold
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing state sold, got %s", stored.State())
	}

	// both deliveries re-acked MOVED (and re-emitted LISTING_SOLD) with the same
	// transactionId. A replayed LISTING_SOLD is a harmless idempotent buyer notice.
	moved := eventsOfType(rp.events, custody.StatusEventTypeMoved)
	if len(moved) != 2 {
		t.Fatalf("expected 2 MOVED acks (original + replay), got %d (all: %v)", len(moved), rp.events)
	}
	for i, ev := range moved {
		if ev.transactionId != transactionId {
			t.Fatalf("ack %d transactionId mismatch: %s", i, ev.transactionId)
		}
	}
	if got := len(eventsOfType(rp.events, mtsmsg.StatusEventTypeListingSold)); got != 2 {
		t.Fatalf("expected 2 LISTING_SOLD events (original + replay), got %d (all: %v)", got, rp.events)
	}
}

func TestReleaseFromMtsHolding_SoftDeletesAndIsIdempotent(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	// seed a holding row to release
	holdingId := uuid.New()
	m, err := holding.NewBuilder(test.TestTenantId, 0, 99).
		SetId(holdingId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	if _, err := holding.CreateHolding(db.WithContext(ctx), m); err != nil {
		t.Fatalf("seed holding: %v", err)
	}

	rp := &recordingProducer{}
	transactionId := uuid.New()
	cmd := custody.Command[custody.ReleaseFromMtsHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandReleaseFromMtsHolding,
		Body:          custody.ReleaseFromMtsHoldingCommandBody{HoldingId: holdingId},
	}

	// first delivery: soft-deletes + acks
	handleReleaseFromMtsHolding(rp.provider())(db)(l, ctx, cmd)
	if _, err := holding.GetById(holdingId.String())(db.WithContext(ctx))(); err == nil {
		t.Fatalf("expected holding soft-deleted after release")
	}

	// replayed delivery: already released, re-acks without error
	handleReleaseFromMtsHolding(rp.provider())(db)(l, ctx, cmd)

	if len(rp.events) != 2 {
		t.Fatalf("expected 2 RELEASED acks (original + replay), got %d", len(rp.events))
	}
	for i, ev := range rp.events {
		if ev.eventType != custody.StatusEventTypeReleased {
			t.Fatalf("ack %d not RELEASED: %s", i, ev.eventType)
		}
		if ev.transactionId != transactionId {
			t.Fatalf("ack %d transactionId mismatch: %s", i, ev.transactionId)
		}
	}
}

// TestRestoreMtsHolding_UndoesReleaseAndIsIdempotent asserts the WithdrawFromMts
// compensation handler un-soft-deletes the holding (making it readable again)
// and is idempotent: a replayed restore on an already-live row re-acks RESTORED
// without error. This is the dupe-safety inverse of ReleaseFromMtsHolding.
func TestRestoreMtsHolding_UndoesReleaseAndIsIdempotent(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	// seed + release a holding row so there is something to restore
	holdingId := uuid.New()
	m, err := holding.NewBuilder(test.TestTenantId, 0, 99).
		SetId(holdingId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	if _, err := holding.CreateHolding(db.WithContext(ctx), m); err != nil {
		t.Fatalf("seed holding: %v", err)
	}
	if _, err := holding.SoftDelete(db.WithContext(ctx), holdingId.String()); err != nil {
		t.Fatalf("seed release: %v", err)
	}

	rp := &recordingProducer{}
	transactionId := uuid.New()
	cmd := custody.Command[custody.RestoreMtsHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandRestoreMtsHolding,
		Body:          custody.RestoreMtsHoldingCommandBody{HoldingId: holdingId},
	}

	// first delivery: un-soft-deletes + acks; the holding is readable again
	handleRestoreMtsHolding(rp.provider())(db)(l, ctx, cmd)
	if _, err := holding.GetById(holdingId.String())(db.WithContext(ctx))(); err != nil {
		t.Fatalf("expected holding restored (readable) after restore: %v", err)
	}

	// replayed delivery: already live, re-acks without error
	handleRestoreMtsHolding(rp.provider())(db)(l, ctx, cmd)

	if len(rp.events) != 2 {
		t.Fatalf("expected 2 RESTORED acks (original + replay), got %d", len(rp.events))
	}
	for i, ev := range rp.events {
		if ev.eventType != custody.StatusEventTypeRestored {
			t.Fatalf("ack %d not RESTORED: %s", i, ev.eventType)
		}
		if ev.transactionId != transactionId {
			t.Fatalf("ack %d transactionId mismatch: %s", i, ev.transactionId)
		}
	}
}
