package custody

import (
	"atlas-mts/holding"
	consumer2 "atlas-mts/kafka/consumer"
	msg "atlas-mts/kafka/message"
	"atlas-mts/kafka/message/custody"
	producer2 "atlas-mts/kafka/producer"
	custodyproducer "atlas-mts/kafka/producer/custody"
	"atlas-mts/listing"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitConsumers registers the MTS custody command consumer (the saga custody
// channel), mirroring the cash-compartment consumer.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mts_custody_command")(custody.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

// InitHandlers wires the accept/release custody command handlers onto the
// custody command topic. The producer.Provider is constructed per delivery from
// the message context so emitted acks carry the right tenant/span headers.
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(custody.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptToMtsListing(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleaseFromMtsHolding(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// providerFn is the shape of the per-context producer factory returned by
// producer2.ProviderImpl(l): func(ctx) func(token) MessageProducer.
type providerFn = func(ctx context.Context) func(token string) kprod.MessageProducer

// handleAcceptToMtsListing CREATES the listing row in active state from the
// carried snapshot, using the caller-supplied ListingId so the create is
// deterministic and idempotent. A replayed delivery (same ListingId) finds the
// row already present and is a no-op that still re-acks ACCEPTED. The whole
// row-create runs in one local DB transaction.
func handleAcceptToMtsListing(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.AcceptToMtsListingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.AcceptToMtsListingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.AcceptToMtsListingCommandBody]) {
			if c.Type != custody.CommandAcceptToMtsListing {
				return
			}
			b := c.Body
			tdb := db.WithContext(ctx)

			err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
				// Idempotency: if a row already exists for this listing id, the
				// command has already been applied — no-op, do not create a
				// duplicate.
				if existing, gerr := listing.GetById(b.ListingId.String())(tx)(); gerr == nil && existing.Id() == b.ListingId {
					return nil
				}

				t := tenant.MustFromContext(ctx)
				tid := t.Id()

				m, berr := listing.NewBuilder(tid, world.Id(b.WorldId), b.SellerId).
					SetId(b.ListingId).
					SetSellerName(b.SellerName).
					SetSaleType(listing.SaleType(b.SaleType)).
					SetState(listing.StateActive).
					SetTemplateId(b.TemplateId).
					SetQuantity(b.Quantity).
					SetStrength(b.Strength).
					SetDexterity(b.Dexterity).
					SetIntelligence(b.Intelligence).
					SetLuck(b.Luck).
					SetHP(b.HP).
					SetMP(b.MP).
					SetWeaponAttack(b.WeaponAttack).
					SetMagicAttack(b.MagicAttack).
					SetWeaponDefense(b.WeaponDefense).
					SetMagicDefense(b.MagicDefense).
					SetAccuracy(b.Accuracy).
					SetAvoidability(b.Avoidability).
					SetHands(b.Hands).
					SetSpeed(b.Speed).
					SetJump(b.Jump).
					SetSlots(b.Slots).
					SetLevel(b.Level).
					SetItemLevel(b.ItemLevel).
					SetItemExp(b.ItemExp).
					SetRingId(b.RingId).
					SetViciousCount(b.ViciousCount).
					SetFlags(b.Flags).
					SetListValue(b.ListValue).
					SetBuyNowPrice(b.BuyNowPrice).
					SetCommissionRate(b.CommissionRate).
					SetCategory(b.Category).
					SetSubCategory(b.SubCategory).
					SetEndsAt(b.EndsAt).
					SetMinIncrement(b.MinIncrement).
					Build()
				if berr != nil {
					return berr
				}
				_, cerr := listing.CreateListing(tx, m)
				return cerr
			})

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to accept listing [%s] for transaction [%s].", b.ListingId.String(), c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				})
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(custody.EnvStatusEventTopic, custodyproducer.AcceptedStatusEventProvider(c.TransactionId, b.ListingId))
			})
		}
	}
}

// handleReleaseFromMtsHolding soft-deletes the holding row by id. Soft-delete is
// idempotent: a replayed delivery affects 0 rows (already gone) and still acks
// RELEASED. The whole delete runs in one local DB transaction.
func handleReleaseFromMtsHolding(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.ReleaseFromMtsHoldingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.ReleaseFromMtsHoldingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.ReleaseFromMtsHoldingCommandBody]) {
			if c.Type != custody.CommandReleaseFromMtsHolding {
				return
			}
			tdb := db.WithContext(ctx)

			err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
				// SoftDelete is idempotent: 0 rows affected on a replay (already
				// released) is success, not an error.
				_, derr := holding.SoftDelete(tx, c.Body.HoldingId.String())
				return derr
			})

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to release holding [%s] for transaction [%s].", c.Body.HoldingId.String(), c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				})
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ReleasedStatusEventProvider(c.TransactionId, c.Body.HoldingId))
			})
		}
	}
}
