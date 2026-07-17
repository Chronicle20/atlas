package custody

import (
	"atlas-mts/holding"
	consumer2 "atlas-mts/kafka/consumer"
	msg "atlas-mts/kafka/message"
	"atlas-mts/kafka/message/custody"
	mtsmsg "atlas-mts/kafka/message/mts"
	producer2 "atlas-mts/kafka/producer"
	custodyproducer "atlas-mts/kafka/producer/custody"
	mtsproducer "atlas-mts/kafka/producer/mts"
	"atlas-mts/listing"
	"atlas-mts/wish"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
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
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRestoreMtsHolding(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMtsMoveListingToHolding(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveMtsListing(producer2.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRestoreListingFromHolding(producer2.ProviderImpl(l))(db)))); err != nil {
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

			p := pf(ctx)

			// The row-create business logic (idempotency, category/subCategory
			// derivation, auction currentBid seeding, builder assembly) lives in the
			// listing processor; this handler maps the command body to the request and
			// owns only the Kafka acks. The create and its success acks commit in one
			// transaction: the ACCEPTED + LISTING_CREATED events are enqueued as outbox
			// rows in the same tx as the listing row, so a crash before commit emits
			// nothing (task-114 atomicity).
			terr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				if err := listing.NewProcessor(l, ctx, tx).Accept(listing.AcceptRequest{
					ListingId:        b.ListingId,
					WorldId:          b.WorldId,
					SellerId:         b.SellerId,
					SellerAccountId:  b.SellerAccountId,
					SellerName:       b.SellerName,
					SaleType:         b.SaleType,
					TemplateId:       b.TemplateId,
					Quantity:         b.Quantity,
					Strength:         b.Strength,
					Dexterity:        b.Dexterity,
					Intelligence:     b.Intelligence,
					Luck:             b.Luck,
					HP:               b.HP,
					MP:               b.MP,
					WeaponAttack:     b.WeaponAttack,
					MagicAttack:      b.MagicAttack,
					WeaponDefense:    b.WeaponDefense,
					MagicDefense:     b.MagicDefense,
					Accuracy:         b.Accuracy,
					Avoidability:     b.Avoidability,
					Hands:            b.Hands,
					Speed:            b.Speed,
					Jump:             b.Jump,
					Slots:            b.Slots,
					Level:            b.Level,
					ItemLevel:        b.ItemLevel,
					ItemExp:          b.ItemExp,
					RingId:           b.RingId,
					ViciousCount:     b.ViciousCount,
					Flags:            b.Flags,
					ListValue:        b.ListValue,
					BuyNowPrice:      b.BuyNowPrice,
					CommissionRate:   b.CommissionRate,
					Category:         b.Category,
					SubCategory:      b.SubCategory,
					EndsAt:           b.EndsAt,
					MinIncrement:     b.MinIncrement,
					OfferWishSerial:  b.OfferWishSerial,
					OfferWishOwnerId: b.OfferWishOwnerId,
				}); err != nil {
					return err
				}

				// On success emit BOTH the custody ACCEPTED ack (drives the saga
				// forward — the orchestrator needs it to advance the listing-create
				// saga) AND the high-level LISTING_CREATED MTS status event so the
				// channel writes RegisterSaleEntryDone to the seller. This mirrors the
				// MOVED+LISTING_SOLD dual-emit in handleMtsMoveListingToHolding: the
				// custody ack is the saga machinery, the MTS status event is the
				// player-facing notice. Without the LISTING_CREATED emit the seller's
				// client hangs (no RegisterSaleEntryDone).
				return msg.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *msg.Buffer) error {
					if perr := buf.Put(custody.EnvStatusEventTopic, custodyproducer.AcceptedStatusEventProvider(c.TransactionId, b.ListingId)); perr != nil {
						return perr
					}
					return buf.Put(mtsmsg.EnvStatusEventTopic, mtsproducer.ListingCreatedStatusEventProvider(c.TransactionId, b.WorldId, b.ListingId, b.SellerId, b.TemplateId, b.SaleType))
				})
			})
			if terr != nil {
				l.WithError(terr).Errorf("Failed to accept listing [%s] for transaction [%s].", b.ListingId.String(), c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, terr.Error()))
				})
			}
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
			p := pf(ctx)

			// The capture-then-soft-delete tx lives in the holding processor; this
			// handler owns only the Kafka acks. Released + (optional) ITEM_TAKEN_HOME
			// are enqueued as outbox rows in the same tx as the soft-delete, so they
			// publish iff the release commits.
			terr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				res, err := holding.NewProcessor(l, ctx, tx).Release(c.Body.HoldingId.String())
				if err != nil {
					return err
				}
				return msg.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *msg.Buffer) error {
					if perr := buf.Put(custody.EnvStatusEventTopic, custodyproducer.ReleasedStatusEventProvider(c.TransactionId, c.Body.HoldingId)); perr != nil {
						return perr
					}
					// ITEM_TAKEN_HOME drives the channel's re-push of the owner's
					// "Transfer Inventory" panel so the just-withdrawn holding disappears
					// without re-entering the MTS. Release is the take-home soft-delete
					// boundary (WithdrawFromMts), so this is the natural emission point.
					if res.EmitTakenHome {
						return buf.Put(mtsmsg.EnvStatusEventTopic, mtsproducer.ItemTakenHomeStatusEventProvider(c.TransactionId, byte(res.Taken.WorldId()), res.Taken.Id(), res.Taken.OwnerId(), res.Taken.TemplateId()))
					}
					return nil
				})
			})
			if terr != nil {
				l.WithError(terr).Errorf("Failed to release holding [%s] for transaction [%s].", c.Body.HoldingId.String(), c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, terr.Error()))
				})
			}
		}
	}
}

// handleRestoreMtsHolding un-soft-deletes the holding row by id — the inverse of
// handleReleaseFromMtsHolding, dispatched by the saga compensator when a
// WithdrawFromMts saga fails after the holding was already released. Restore is
// idempotent: clearing deleted_at on an already-live row affects 0 rows and is
// success, not an error. The whole restore runs in one local DB transaction.
func handleRestoreMtsHolding(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreMtsHoldingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreMtsHoldingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.RestoreMtsHoldingCommandBody]) {
			if c.Type != custody.CommandRestoreMtsHolding {
				return
			}
			p := pf(ctx)

			// The idempotent un-soft-delete tx lives in the holding processor; this
			// handler owns only the Kafka acks. RESTORED is enqueued as an outbox row
			// in the same tx as the restore, so it publishes iff the restore commits.
			terr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				if err := holding.NewProcessor(l, ctx, tx).RestoreHolding(c.Body.HoldingId.String()); err != nil {
					return err
				}
				return msg.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.RestoredStatusEventProvider(c.TransactionId, c.Body.HoldingId))
				})
			})
			if terr != nil {
				l.WithError(terr).Errorf("Failed to restore holding [%s] for transaction [%s].", c.Body.HoldingId.String(), c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, terr.Error()))
				})
			}
		}
	}
}

// handleMtsMoveListingToHolding settles a purchase: in ONE local DB transaction it
// (a) loads the listing, (b) conditionally marks it sold via the listing
// administrator's UpdateState(active→sold), and (c) creates the buyer's holding row
// (origin=purchased) copying the listing's item snapshot. Idempotency: the buyer
// holding id is derived deterministically from (listingId, buyerId); a replayed
// delivery finds that holding already present and is a no-op that still re-acks
// MOVED. The conditional UpdateState affecting 0 rows on a replay (already sold) is
// likewise success, not an error.
func handleMtsMoveListingToHolding(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.MtsMoveListingToHoldingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.MtsMoveListingToHoldingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.MtsMoveListingToHoldingCommandBody]) {
			if c.Type != custody.CommandMtsMoveListingToHolding {
				return
			}
			b := c.Body

			p := pf(ctx)

			// The settle tx (load listing, conditional ->sold transition, single-custody
			// race guard, buyer-holding create, both parties' history rows) lives in the
			// listing processor; here it is wrapped so the MOVED ack + LISTING_SOLD
			// status event enqueue as outbox rows in the same tx as the settle — they
			// publish iff the sale commits. The result carries what the post-commit
			// offer/escrow side-effects below need (sale type + fulfilled want-ad serial).
			var res listing.SettleMoveResult
			terr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				r, err := listing.NewProcessor(l, ctx, tx).SettleMove(listing.SettleMoveRequest{
					ListingId: b.ListingId,
					BuyerId:   b.BuyerId,
					WorldId:   b.WorldId,
					Price:     b.Price,
				})
				if err != nil {
					return err
				}
				res = r
				// On success emit BOTH the custody MOVED ack (drives the saga forward) AND
				// the high-level LISTING_SOLD MTS status event so the channel writes
				// BuyItemDone to the buyer. The buyer (or auction winner) is b.BuyerId.
				return msg.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *msg.Buffer) error {
					if perr := buf.Put(custody.EnvStatusEventTopic, custodyproducer.MovedStatusEventProvider(c.TransactionId, b.ListingId, r.HoldingId)); perr != nil {
						return perr
					}
					return buf.Put(mtsmsg.EnvStatusEventTopic, mtsproducer.ListingSoldStatusEventProvider(c.TransactionId, b.WorldId, b.ListingId, r.SellerId, b.BuyerId, r.ItemId, r.SoldSaleType, b.ResultKind, b.Price))
				})
			})
			if terr != nil {
				l.WithError(terr).Errorf("Failed to move listing [%s] to holding for buyer [%d], transaction [%s].", b.ListingId.String(), b.BuyerId, c.TransactionId.String())
				_ = msg.Emit(p)(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, terr.Error()))
				})
				return
			}
			soldSaleType := res.SoldSaleType
			soldOfferWishSerial := res.SoldOfferWishSerial

			// Offer purchase (BUY_WISH): the want-ad has been fulfilled. Consume it
			// and return every LOSING offer to its offerer's Transfer Inventory. Both
			// are best-effort post-commit: the settle (money + item to the buyer) has
			// already committed above, and an un-released sibling stays safely
			// escrowed (reclaimable via Not-Yet-Sold cancel or the expiry sweep).
			//
			// The sibling LISTING_CANCELLED notices stay on the DIRECT producer (not
			// the outbox): ReleaseSiblingOffers fans out N independent per-sibling
			// Cancel transactions and deliberately swallows per-sibling failures, so
			// there is no single transaction to bind these notices to. They are a
			// best-effort courtesy notice — the offerer's item is already in their
			// holding and the escrow is sweep-recoverable — the same best-effort
			// exclusion class task-114 documents in inventory.md.
			if soldSaleType == string(listing.SaleTypeOffer) && soldOfferWishSerial != 0 {
				if _, derr := wish.NewProcessor(l, ctx, db).DeleteBySerial(world.Id(b.WorldId), soldOfferWishSerial); derr != nil {
					l.WithError(derr).Warnf("Unable to consume fulfilled want-ad (serial [%d]) for buyer [%d].", soldOfferWishSerial, b.BuyerId)
				}
				released := listing.NewProcessor(l, ctx, db).ReleaseSiblingOffers(world.Id(b.WorldId), soldOfferWishSerial, b.ListingId)
				if len(released) > 0 {
					_ = msg.Emit(p)(func(buf *msg.Buffer) error {
						for _, r := range released {
							if perr := buf.Put(mtsmsg.EnvStatusEventTopic, mtsproducer.ListingCancelledStatusEventProvider(c.TransactionId, b.WorldId, r.ListingId, r.HoldingId, r.SellerId, r.ItemId)); perr != nil {
								return perr
							}
						}
						return nil
					})
				}
			}

			// A BUY-NOW that settles an auction with an outstanding high bid must
			// refund that bidder's escrow — they never won. ReleaseHighBidEscrow is
			// self-guarding (releases only a still-Held bid), so a settle-to-WINNER
			// (whose bid is StateWon before the move ran) no-ops here; only a buy-now
			// releases. Best-effort post-commit + idempotent (the bid is marked
			// released), so a replay of this move does not double-refund.
			if soldSaleType == string(listing.SaleTypeAuction) {
				if rerr := listing.NewProcessor(l, ctx, db).ReleaseHighBidEscrow(world.Id(b.WorldId), b.ListingId); rerr != nil {
					l.WithError(rerr).Warnf("Unable to release high-bid escrow on settle for listing [%s].", b.ListingId.String())
				}
			}
		}
	}
}

// handleRemoveMtsListing hard-deletes a spurious ACTIVE listing by id — the
// late-compensation inverse of AcceptToMtsListing (a list saga that timed out,
// re-granted the item to the seller, then had its accept land late, duplicating
// the item). listing.DeleteActive is guarded to state=active, so a listing
// bought/cancelled/settled in the interim is left untouched (0 rows = success).
// These late-inverse handlers emit only an ERROR ack on failure: no consumer
// awaits a compensation success (dispatchLateInverse is fire-and-forget), and a
// new success event kind would cascade the whole event pipeline for no reader.
func handleRemoveMtsListing(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.RemoveMtsListingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.RemoveMtsListingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.RemoveMtsListingCommandBody]) {
			if c.Type != custody.CommandRemoveMtsListing {
				return
			}
			// The guarded active-only delete tx lives in the listing processor; this
			// handler owns only the ERROR ack + the removed-vs-noop logging.
			affected, err := listing.NewProcessor(l, ctx, db).RemoveSpuriousActive(c.Body.ListingId.String())
			if err != nil {
				l.WithError(err).Errorf("Failed to remove spurious listing [%s] for transaction [%s].", c.Body.ListingId.String(), c.TransactionId.String())
				_ = msg.Emit(pf(ctx))(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				})
				return
			}
			if affected == 0 {
				l.Infof("RemoveMtsListing: listing [%s] not active (already bought/cancelled/removed); nothing to remove, transaction [%s].", c.Body.ListingId.String(), c.TransactionId.String())
				return
			}
			l.Infof("RemoveMtsListing: removed spurious active listing [%s], transaction [%s].", c.Body.ListingId.String(), c.TransactionId.String())
		}
	}
}

// handleRestoreListingFromHolding reverses a settlement move — the
// late-compensation inverse of MtsMoveListingToHolding. It delegates to the listing
// processor's RestoreFromHolding, which in one tx soft-deletes the deterministic
// buyer holding (listing.MoveHoldingId(listingId, buyerId)) and transitions the
// listing sold->active, so a buy that landed late after the buyer was refunded
// returns the item to the marketplace and leaves the buyer nothing.
// Both mutations are idempotent (0 rows on replay = success). See
// handleRemoveMtsListing for the emit-on-error-only rationale.
func handleRestoreListingFromHolding(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreListingFromHoldingCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreListingFromHoldingCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.RestoreListingFromHoldingCommandBody]) {
			if c.Type != custody.CommandRestoreListingFromHolding {
				return
			}
			hid := listing.MoveHoldingId(c.Body.ListingId, c.Body.BuyerId)

			// The soft-delete-buyer-holding + listing sold->active tx lives in the
			// listing processor; this handler owns only the ERROR ack + the logging.
			err := listing.NewProcessor(l, ctx, db).RestoreFromHolding(c.Body.ListingId.String(), c.Body.BuyerId)
			if err != nil {
				l.WithError(err).Errorf("Failed to reverse move for listing [%s] buyer [%d], transaction [%s].", c.Body.ListingId.String(), c.Body.BuyerId, c.TransactionId.String())
				_ = msg.Emit(pf(ctx))(func(buf *msg.Buffer) error {
					return buf.Put(custody.EnvStatusEventTopic, custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				})
				return
			}
			l.Infof("RestoreListingFromHolding: reversed move for listing [%s] buyer [%d] (holding [%s] removed, listing restored to active), transaction [%s].", c.Body.ListingId.String(), c.Body.BuyerId, hid.String(), c.TransactionId.String())
		}
	}
}
