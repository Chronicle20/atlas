package mts

import (
	"atlas-mts/holding"
	consumer2 "atlas-mts/kafka/consumer"
	msg "atlas-mts/kafka/message"
	"atlas-mts/kafka/message/mts"
	producer2 "atlas-mts/kafka/producer"
	mtsproducer "atlas-mts/kafka/producer/mts"
	"atlas-mts/listing"
	"atlas-mts/transaction"
	"atlas-mts/wish"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
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

// InitHandlers wires the locally-handled MTS command handlers (create, cancel,
// take-home, buy, bid, register wish, remove wish) onto the command topic. The
// remaining ticker-driven command type (expire) is intentionally NOT routed here.
// The producer.Provider is constructed per delivery from the message context so
// emitted events carry the right tenant/span headers.
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(mts.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateListing(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelListing(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleTakeHome(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuy(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePlaceBid(producer2.ProviderImpl(l))(db)))); err != nil {
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


// failReasonFor maps a buy/bid rejection error to the SEMANTIC failure key
// the channel resolves against the tenant "noticeFailReasons" writer table
// (see the FailReason* docs in the mts message package). Unmapped errors stay
// generic (empty key -> the operation's bare *Failed arm).
func failReasonFor(err error) string {
	switch {
	case errors.Is(err, listing.ErrInsufficientPrepaid):
		return mts.FailReasonNotEnoughNX
	case errors.Is(err, listing.ErrListingUnavailable), errors.Is(err, gorm.ErrRecordNotFound):
		return mts.FailReasonItemSold
	default:
		return mts.FailReasonGeneric
	}
}

// handleCreateListing initiates a listing from the channel's register-sale /
// register-auction / sale-current-item arm. It maps the command body to a
// listing.ListRequest and runs the server-authoritative List flow (price-floor,
// active-cap, auction-duration validation + TransferToMts saga). On a validation
// or emit failure it emits LISTING_CREATE_FAILED so the channel writes
// RegisterSaleEntryFailed to the seller. The LISTING_CREATED success event is
// emitted by the custody consumer's AcceptToMtsListing (the listing row, and thus
// its serial, only exists then) — not here.
func handleCreateListing(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.CreateListingCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.CreateListingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.CreateListingCommandBody]) {
			if c.Type != mts.CommandCreateListing {
				return
			}
			b := c.Body

			_, err := listing.NewProcessor(l, ctx, db).List(listing.ListRequest{
				WorldId:             world.Id(b.WorldId),
				SellerId:            b.SellerId,
				SellerAccountId:     b.SellerAccountId,
				SellerName:          b.SellerName,
				SaleType:            listing.SaleType(b.SaleType),
				SourceInventoryType: b.SourceInventoryType,
				AssetId:             b.AssetId,
				Quantity:            b.Quantity,
				ListValue:           b.ListValue,
				BuyNowPrice:         b.BuyNowPrice,
				DurationHours:       b.DurationHours,
				MinIncrement:        b.MinIncrement,
				Category:            b.Category,
				SubCategory:         b.SubCategory,
				OfferWishSerial:     b.OfferWishSerial,
				OfferWishOwnerId:    b.OfferWishOwnerId,
			})
			if err != nil {
				l.WithError(err).Errorf("Failed to initiate listing for seller [%d], transaction [%s].", b.SellerId, c.TransactionId.String())
				p := pf(ctx)
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					// The synchronous registration validations (auction duration out of
					// range, price below floor, too many active listings) have no
					// registration-specific v83 client string, so they all resolve to the
					// generic "the request for MTS has failed" notice — but through the
					// config-driven reasonKey path (like buy/bid), not a hardcoded byte.
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.ListingCreateFailedStatusEventProvider(c.TransactionId, b.WorldId, b.SellerId, mts.FailReasonRegisterFailed))
				})
				return
			}
		}
	}
}

// handleCancelListing performs the seller's race-safe cancel. It resolves the
// wire serial (nITCSN) to the listing UUID (listing.GetBySerial), owner-checks the
// command's SellerId against the listing's seller, then — in ONE local DB
// transaction — conditionally transitions the row active->cancelled and, only if
// that affected exactly one row, creates the seller's holding (origin=cancelled)
// copying the listing's item snapshot. The conditional transition is the race
// arbiter: a concurrent buy that already moved the row out of active makes this
// the cancel-vs-buy LOSER (no holding, LISTING_CANCEL_FAILED emitted). A serial
// that does not resolve or an owner-check mismatch likewise emits
// LISTING_CANCEL_FAILED so the channel writes CancelSaleItemFailed to the seller.
func handleCancelListing(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.CancelListingCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.CancelListingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.CancelListingCommandBody]) {
			if c.Type != mts.CommandCancelListing {
				return
			}
			b := c.Body
			p := pf(ctx)

			emitFail := func() {
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.ListingCancelFailedStatusEventProvider(c.TransactionId, b.WorldId, b.Serial, b.SellerId, mts.FailReasonGeneric))
				})
			}

			// The serial owner-check, race-safe cancel, and cancelled-history row all
			// live in the listing processor (shared with the REST DELETE); the consumer
			// only maps the wire serial and emits the result.
			res, err := listing.NewProcessor(l, ctx, db).CancelBySerial(world.Id(b.WorldId), b.Serial, b.SellerId)
			if err != nil {
				l.WithError(err).Errorf("Failed to cancel serial [%d] in world [%d], transaction [%s].", b.Serial, b.WorldId, c.TransactionId.String())
				emitFail()
				return
			}
			if !res.Won {
				// Lost the cancel-vs-buy race: the buy path owns the holding; the seller
				// still gets the cancel-failed notice.
				l.Debugf("Cancel for serial [%d] lost the cancel-vs-buy race (already not active); no holding created.", b.Serial)
				emitFail()
				return
			}
			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.ListingCancelledStatusEventProvider(c.TransactionId, b.WorldId, res.ListingId, res.HoldingId, res.SellerId, res.ItemId))
			})
		}
	}
}

// handleTakeHome withdraws a holding into the owner's inventory. It resolves the
// wire serial (nITCSN) to the holding UUID (holding.GetBySerial), owner-checks the
// command's CharacterId against the holding's owner, then runs the WithdrawFromMts
// saga (release_from_mts_holding + accept_to_character). On a serial-not-resolved,
// owner-check, or emit failure it emits TAKE_HOME_FAILED so the channel writes
// MoveItcPurchaseItemLtoSFailed to the character. The ITEM_TAKEN_HOME success
// event is emitted by the saga's accept step (a later phase), not here.
func handleTakeHome(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.TakeHomeCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.TakeHomeCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.TakeHomeCommandBody]) {
			if c.Type != mts.CommandTakeHome {
				return
			}
			b := c.Body
			p := pf(ctx)

			emitFail := func() {
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.TakeHomeFailedStatusEventProvider(c.TransactionId, b.WorldId, b.Serial, b.CharacterId, mts.FailReasonGeneric))
				})
			}

			proc := holding.NewProcessor(l, ctx, db)

			// Resolve the wire serial -> holding UUID.
			hm, err := proc.GetBySerial(world.Id(b.WorldId), b.Serial)
			if err != nil {
				l.WithError(err).Errorf("Failed to resolve serial [%d] for take-home in world [%d], transaction [%s].", b.Serial, b.WorldId, c.TransactionId.String())
				emitFail()
				return
			}

			// Owner-check: only the holding's owner may take it home.
			if hm.OwnerId() != b.CharacterId {
				l.Errorf("Character [%d] attempted to take home holding [%s] (serial [%d]) owned by [%d]; forbidden.", b.CharacterId, hm.Id().String(), b.Serial, hm.OwnerId())
				emitFail()
				return
			}

			if _, err := proc.TakeHome(hm.Id().String(), b.CharacterId, world.Id(b.WorldId), b.InventoryType, b.Slot); err != nil {
				l.WithError(err).Errorf("Failed to take home holding [%s] for transaction [%s].", hm.Id().String(), c.TransactionId.String())
				emitFail()
				return
			}
		}
	}
}

// handleBuy settles a buy / buy-now. It resolves the wire serial (nITCSN) to the
// listing UUID (listing.GetBySerial), reads the seller account from the resolved
// listing row, then asks the listing processor to load the (active) listing,
// compute the marked-up price (from listValue for a plain buy, or buyNowPrice for a
// buy-now), pre-check the buyer's NX Prepaid balance, and — when sufficient — emit a
// debit-first MtsSettlePurchase saga. On a serial-not-resolved, non-active, or
// insufficient-funds rejection it emits BUY_FAILED so the channel writes
// BuyItemFailed to the buyer. The BUY success notice (BuyItemDone) is driven by the
// LISTING_SOLD event the saga's move step emits — not here.
func handleBuy(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.BuyCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.BuyCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.BuyCommandBody]) {
			if c.Type != mts.CommandBuy {
				return
			}
			b := c.Body
			p := pf(ctx)

			emitFail := func(reason string) {
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.BuyFailedStatusEventProvider(c.TransactionId, b.WorldId, b.Serial, b.BuyerId, reason, b.ResultKind))
				})
			}

			proc := listing.NewProcessor(l, ctx, db)

			// Resolve the wire serial -> listing UUID; the seller account is read from
			// the resolved row (captured at list time), never carried on the wire.
			lm, err := proc.GetBySerial(world.Id(b.WorldId), b.Serial)
			if err != nil {
				l.WithError(err).Errorf("Failed to resolve serial [%d] for buy in world [%d], transaction [%s].", b.Serial, b.WorldId, c.TransactionId.String())
				emitFail(failReasonFor(err))
				return
			}

			if err := proc.Buy(listing.BuyRequest{
				WorldId:         world.Id(b.WorldId),
				ListingId:       lm.Id(),
				BuyerId:         b.BuyerId,
				BuyerAccountId:  b.BuyerAccountId,
				SellerAccountId: lm.SellerAccountId(),
				BuyNow:          b.BuyNow,
				ResultKind:      b.ResultKind,
			}); err != nil {
				l.WithError(err).Errorf("Failed to settle buy for listing [%s] (serial [%d]), buyer [%d], transaction [%s].", lm.Id().String(), b.Serial, b.BuyerId, c.TransactionId.String())
				emitFail(failReasonFor(err))
				return
			}
		}
	}
}

// handlePlaceBid places a bid on an auction listing. It resolves the wire serial
// (nITCSN) to the listing UUID (listing.GetBySerial), then asks the listing
// processor to validate the listing is an active auction and the bid clears the
// floor (listValue for the first bid, else currentBid + minIncrement), and — under a
// race-safe compare-and-swap on the listing row — record a held bid, advance the
// listing's currentBid/highBidder, and emit an MtsBidEscrow{-markedUp} saga to hold
// the bidder's prepaid (the MARKED-UP amount). On an outbid it releases the prior
// bidder's escrow. On a serial-not-resolved, non-auction, below-floor, or lost-race
// rejection it emits BID_FAILED so the channel writes BidAuctionFailed to the
// bidder. The success/settle notice (SuccessBidInfoResult) is emitted at auction
// settle (the ticker), not here.
func handlePlaceBid(pf providerFn) func(db *gorm.DB) message.Handler[mts.Command[mts.PlaceBidCommandBody]] {
	return func(db *gorm.DB) message.Handler[mts.Command[mts.PlaceBidCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c mts.Command[mts.PlaceBidCommandBody]) {
			if c.Type != mts.CommandPlaceBid {
				return
			}
			b := c.Body
			p := pf(ctx)

			emitFail := func(reason string) {
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.BidFailedStatusEventProvider(c.TransactionId, b.WorldId, b.Serial, b.BidderId, reason))
				})
			}

			proc := listing.NewProcessor(l, ctx, db)

			// Resolve the wire serial -> listing UUID.
			lm, err := proc.GetBySerial(world.Id(b.WorldId), b.Serial)
			if err != nil {
				l.WithError(err).Errorf("Failed to resolve serial [%d] for bid in world [%d], transaction [%s].", b.Serial, b.WorldId, c.TransactionId.String())
				emitFail(failReasonFor(err))
				return
			}

			res, err := proc.PlaceBid(listing.BidRequest{
				WorldId:         world.Id(b.WorldId),
				ListingId:       lm.Id(),
				BidderId:        b.BidderId,
				BidderAccountId: b.BidderAccountId,
				Amount:          b.Amount,
			})
			if err != nil {
				l.WithError(err).Errorf("Failed to place bid for listing [%s] (serial [%d]), bidder [%d], transaction [%s].", lm.Id().String(), b.Serial, b.BidderId, c.TransactionId.String())
				emitFail(failReasonFor(err))
				return
			}

			// Emit BID_PLACED so the channel refreshes the bidder's NX (the escrow
			// debit just left their prepaid). On an outbid, also emit OUTBID so the
			// displaced bidder's NX is refreshed (their escrow was released).
			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				if perr := buf.Put(mts.EnvStatusEventTopic, mtsproducer.BidPlacedStatusEventProvider(c.TransactionId, b.WorldId, lm.Id(), b.BidderId, b.Amount)); perr != nil {
					return perr
				}
				if res.HadPrior {
					return buf.Put(mts.EnvStatusEventTopic, mtsproducer.OutbidStatusEventProvider(c.TransactionId, b.WorldId, lm.Id(), res.PreviousBidderId))
				}
				return nil
			})

			// Record the outbid bidder's bid-lost history row (nProcessStatus 2). Each
			// outbid is a distinct lost bid, so one row per outbid. Best-effort: a
			// failure leaves history a row short but does not undo the (committed) bid.
			if res.HadPrior {
				t := tenant.MustFromContext(ctx)
				lostTxn, berr := transaction.NewBuilder(t.Id(), world.Id(b.WorldId), res.PreviousBidderId).
					SetId(uuid.New()).
					SetCounterpartyId(res.SellerId).
					SetItemId(res.ItemId).
					SetQuantity(res.Quantity).
					SetTotalPrice(res.PreviousBidAmount).
					SetKind(transaction.KindBidLost).
					Build()
				if berr != nil {
					l.WithError(berr).Warnf("Failed to build bid-lost history row for outbid bidder [%d] on listing [%s].", res.PreviousBidderId, lm.Id().String())
				} else if _, terr := transaction.CreateTransaction(db.WithContext(ctx), lostTxn); terr != nil {
					l.WithError(terr).Warnf("Failed to record bid-lost history row for outbid bidder [%d] on listing [%s].", res.PreviousBidderId, lm.Id().String())
				}
			}
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

			// Origin determines the wish kind (a wire concern resolved here):
			// REGISTER_WISH_ENTRY posts a "wanted" item; SET_ZZIM (and the rest) are
			// "cart" additions. Cart and wanted are disjoint stores so the Cart and
			// Wanted views never bleed together. The row-create tx (and the wanted
			// price/expiry derivation) lives in the wish processor; this handler owns
			// only the WISH_ADDED emission.
			wishType := wish.TypeCart
			if b.Origin == mts.WishOriginRegisterWish {
				wishType = wish.TypeWanted
			}
			err := wish.NewProcessor(l, ctx, db).RegisterWish(wish.RegisterWishRequest{
				WishId:        b.WishId,
				WorldId:       world.Id(b.WorldId),
				CharacterId:   b.CharacterId,
				ItemId:        b.ItemId,
				WishType:      wishType,
				ListingSerial: b.ListingSerial,
				Count:         b.Count,
				Price:         b.Price,
			})

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to register wish [%s] for character [%d], transaction [%s].", b.WishId.String(), b.CharacterId, c.TransactionId.String())
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.WishAddedStatusEventProvider(c.TransactionId, b.WorldId, b.WishId, b.CharacterId, b.ItemId, b.Origin))
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

			// The read-then-delete tx lives in the wish processor; this handler owns
			// only the WISH_REMOVED emission (which echoes the owning characterId).
			characterId, err := wish.NewProcessor(l, ctx, db).RemoveWish(b.WishId.String())

			p := pf(ctx)
			if err != nil {
				l.WithError(err).Errorf("Failed to remove wish [%s] for transaction [%s].", b.WishId.String(), c.TransactionId.String())
				return
			}

			_ = msg.Emit(p)(func(buf *msg.Buffer) error {
				return buf.Put(mts.EnvStatusEventTopic, mtsproducer.WishRemovedStatusEventProvider(c.TransactionId, b.WorldId, b.WishId, characterId, b.Origin))
			})
		}
	}
}
