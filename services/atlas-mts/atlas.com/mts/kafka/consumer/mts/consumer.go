package mts

import (
	consumer2 "atlas-mts/kafka/consumer"
	msg "atlas-mts/kafka/message"
	"atlas-mts/kafka/message/mts"
	producer2 "atlas-mts/kafka/producer"
	mtsproducer "atlas-mts/kafka/producer/mts"
	"atlas-mts/listing"
	"atlas-mts/wish"
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

// InitConsumers registers the high-level MTS command consumer, mirroring the
// custody command consumer.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mts_command")(mts.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

// InitHandlers wires the locally-handled MTS command handlers (cancel, buy,
// register wish, remove wish) onto the command topic. The remaining
// saga/ticker-driven command types (create, bid, take-home, expire) are
// intentionally NOT routed here — they are dispatched in their own phases, like a
// not-yet-routed message type. The producer.Provider is constructed per delivery
// from the message context so emitted events carry the right tenant/span headers.
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(mts.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelListing(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuy(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRegisterWish(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveWish(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// providerFn is the shape of the per-context producer factory returned by
// producer2.ProviderImpl(l): func(ctx) func(token) MessageProducer.
type providerFn = func(ctx context.Context) func(token string) kprod.MessageProducer

// handleCancelListing performs the seller's race-safe cancel: in ONE local DB
// transaction it (a) loads the listing snapshot, (b) conditionally transitions
// the row active->cancelled, and (c) — only if the transition affected exactly
// one row — creates the seller's holding (origin=cancelled) copying the listing's
// item snapshot. The conditional transition is the race arbiter: if a concurrent
// buy already moved the row out of active, the transition affects 0 rows, the
// handler is the cancel-vs-buy LOSER, no holding is created, and NO event is
// emitted (the buy path owns the outcome). Composing the transition AND the
// holding-create in the same ExecuteTransaction guarantees the cancel can never
// half-complete (a cancelled row without its seller holding, or vice versa).
func handleCancelListing(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.CancelListingCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.CancelListingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.CancelListingCommandBody]) {
			if c.Type != mts.CommandCancelListing {
				return
			}
			b := c.Body

			// The race-safe active->holding(seller) transition lives in the listing
			// processor so it is shared verbatim with the REST DELETE; this handler
			// only adds the event emission on a win.
			res, err := listing.NewProcessor(l, ctx, db).Cancel(b.ListingId.String())

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to cancel listing [%s] for transaction [%s].", b.ListingId.String(), c.TransactionId.String())
				return
			}
			if !res.Won {
				// Cancel-vs-buy loser: the buy path owns the outcome; emit nothing.
				l.Debugf("Cancel for listing [%s] lost the cancel-vs-buy race (already not active); no holding created.", b.ListingId.String())
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.ListingCancelledStatusEventProvider(c.TransactionId, b.WorldId, b.ListingId, res.HoldingId, res.SellerId, res.ItemId))
			})
		}
	}
}

// handleBuy settles a buy / buy-now: it asks the listing processor to load the
// (active) listing, compute the marked-up price from the listing's captured
// listValue/commissionRate, pre-check the buyer's NX Prepaid balance, and — when
// sufficient — emit a debit-first MtsSettlePurchase saga to COMMAND_TOPIC_SAGA.
// It emits NO MTS status event itself: the listing-sold / item-moved-to-holding
// outcome is driven by the saga's move step (a later phase wires that ack path).
// An under-funded or non-active buy is logged and dropped (no saga, no effect).
func handleBuy(db *gorm.DB) message.Handler[mts.Command[mts.BuyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.BuyCommandBody]) {
		if c.Type != mts.CommandBuy {
			return
		}
		b := c.Body

		err := listing.NewProcessor(l, ctx, db).Buy(listing.BuyRequest{
			WorldId:         world.Id(b.WorldId),
			ListingId:       b.ListingId,
			BuyerId:         b.BuyerId,
			BuyerAccountId:  b.BuyerAccountId,
			SellerAccountId: b.SellerAccountId,
		})
		if err != nil {
			l.WithError(err).Errorf("Failed to settle buy for listing [%s], buyer [%d], transaction [%s].", b.ListingId.String(), b.BuyerId, c.TransactionId.String())
			return
		}
	}
}

// handleRegisterWish creates a wish-list entry for a character using the
// caller-supplied WishId, then emits WISH_ADDED. The create runs in one local DB
// transaction.
func handleRegisterWish(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.RegisterWishCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.RegisterWishCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.RegisterWishCommandBody]) {
			if c.Type != mts.CommandRegisterWish {
				return
			}
			b := c.Body
			tdb := db.WithContext(ctx)

			err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
				t := tenant.MustFromContext(ctx)
				wm, berr := wish.NewBuilder(t.Id(), b.CharacterId, b.ItemId).
					SetId(b.WishId).
					Build()
				if berr != nil {
					return berr
				}
				_, cerr := wish.CreateWish(tx, wm)
				return cerr
			})

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to register wish [%s] for character [%d], transaction [%s].", b.WishId.String(), b.CharacterId, c.TransactionId.String())
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.WishAddedStatusEventProvider(c.TransactionId, b.WorldId, b.WishId, b.CharacterId, b.ItemId))
			})
		}
	}
}

// handleRemoveWish deletes a wish-list entry, then emits WISH_REMOVED. The wish
// row is read inside the transaction before the delete so the emitted event can
// echo the owning characterId. The delete runs in one local DB transaction.
func handleRemoveWish(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.RemoveWishCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.RemoveWishCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.RemoveWishCommandBody]) {
			if c.Type != mts.CommandRemoveWish {
				return
			}
			b := c.Body
			tdb := db.WithContext(ctx)

			var characterId uint32

			err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
				// Read the row first so the event can echo the owning characterId.
				// A missing row (already removed) leaves characterId 0 and the
				// delete affects 0 rows — both are success, not errors.
				if wm, gerr := wish.GetById(b.WishId.String())(tx)(); gerr == nil {
					characterId = wm.CharacterId()
				}
				_, derr := wish.DeleteWish(tx, b.WishId.String())
				return derr
			})

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to remove wish [%s] for transaction [%s].", b.WishId.String(), c.TransactionId.String())
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.WishRemovedStatusEventProvider(c.TransactionId, b.WorldId, b.WishId, characterId))
			})
		}
	}
}
