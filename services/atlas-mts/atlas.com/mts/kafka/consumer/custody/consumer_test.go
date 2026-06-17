package custody

import (
	"atlas-mts/holding"
	"atlas-mts/kafka/message/custody"
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

	// exactly one ACCEPTED ack carrying the same transactionId
	if len(rp.events) != 1 {
		t.Fatalf("expected 1 ack, got %d", len(rp.events))
	}
	if rp.events[0].eventType != custody.StatusEventTypeAccepted {
		t.Fatalf("expected ACCEPTED ack, got %s", rp.events[0].eventType)
	}
	if rp.events[0].transactionId != transactionId {
		t.Fatalf("ack transactionId mismatch: want %s got %s", transactionId, rp.events[0].transactionId)
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

	// both deliveries re-acked with ACCEPTED + same transactionId
	if len(rp.events) != 2 {
		t.Fatalf("expected 2 acks (original + replay), got %d", len(rp.events))
	}
	for i, ev := range rp.events {
		if ev.eventType != custody.StatusEventTypeAccepted {
			t.Fatalf("ack %d not ACCEPTED: %s", i, ev.eventType)
		}
		if ev.transactionId != transactionId {
			t.Fatalf("ack %d transactionId mismatch: %s", i, ev.transactionId)
		}
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
