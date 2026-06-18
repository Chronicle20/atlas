package mts

import (
	consumer2 "atlas-channel/kafka/consumer"
	mtsmsg "atlas-channel/kafka/message/mts"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// mtsTakeHomePurchaseTab is the tab value the MoveItcPurchaseItemLtoSDone arm
// passes to CCtrlTab::SetTab(tab-1). The purchase ("taken home") items live in the
// first MTS tab, so tab=1 selects index 0. selectedNo 0 leaves the selection at
// the top of the list.
const (
	mtsTakeHomePurchaseTab uint32 = 1
	mtsTakeHomeSelectedNo  uint32 = 0
)

// mtsRegisterSaleNoSaleLimit is the saleLimit short for a RegisterSaleEntryFailed
// whose reason is NOT the sale-limit reason (0x48). The clientbound codec only
// writes the trailing Decode2 when reason==0x48, so any value is inert for the
// generic-reason failures the command path emits.
const mtsRegisterSaleNoSaleLimit uint16 = 0

// InitConsumers registers the EVENT_TOPIC_MTS_STATUS consumer (mirrors the
// cash-shop compartment status-event consumer): tenant/span header parsers + start
// at the latest offset (status events are live notifications, not replayed).
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mts_status_event")(mtsmsg.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers wires the per-event-type handlers onto the status topic. Each
// handler writes the matching clientbound MtsOperation result to the originating
// character's session (resolved by the event's seller/character id), mirroring the
// cash-shop compartment status consumer.
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(mtsmsg.EnvStatusEventTopic)()

				register := func(h handler.Handler) error {
					id, err := rf(t, h)
					if err != nil {
						return err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					return nil
				}

				if err := register(message.AdaptHandler(message.PersistentConfig(handleListingCreated(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleListingCreateFailed(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleListingCancelled(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleListingCancelFailed(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleItemTakenHome(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleTakeHomeFailed(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleListingSold(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleBuyFailed(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleBidFailed(sc, wp)))); err != nil {
					return nil, err
				}
				return handles, nil
			}
		}
	}
}

// announceTo resolves the target character's session on this channel and writes the
// supplied clientbound MtsOperation body to it. A missing session (the character
// is not on this channel) is a graceful no-op; a wrong-tenant event is dropped.
func announceTo(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
	t := tenant.MustFromContext(ctx)
	if !t.Is(sc.Tenant()) {
		return
	}
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		return session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s)
	})
}

func handleListingCreated(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreatedBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCreated {
			return
		}
		l.Debugf("MTS listing created for seller [%d] (item [%d]).", e.Body.SellerId, e.Body.ItemId)
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationRegisterSaleEntryDoneBody())
	}
}

func handleListingCreateFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreateFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreateFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCreateFailed {
			return
		}
		l.Debugf("MTS listing create failed for seller [%d] (reason [%d]).", e.Body.SellerId, e.Body.Reason)
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationRegisterSaleEntryFailedBody(e.Body.Reason, mtsRegisterSaleNoSaleLimit))
	}
}

func handleListingCancelled(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelledBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCancelled {
			return
		}
		l.Debugf("MTS listing cancelled for seller [%d] (item [%d] -> holding [%s]).", e.Body.SellerId, e.Body.ItemId, e.Body.HoldingId.String())
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationCancelSaleItemDoneBody())
	}
}

func handleListingCancelFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCancelFailed {
			return
		}
		l.Debugf("MTS listing cancel failed for seller [%d] serial [%d] (reason [%d]).", e.Body.SellerId, e.Body.Serial, e.Body.Reason)
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationCancelSaleItemFailedBody(e.Body.Reason))
	}
}

func handleItemTakenHome(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventItemTakenHomeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventItemTakenHomeBody]) {
		if e.Type != mtsmsg.StatusEventTypeItemTakenHome {
			return
		}
		l.Debugf("MTS item taken home for character [%d] (item [%d]).", e.Body.CharacterId, e.Body.ItemId)
		announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationMoveItcPurchaseItemLtoSDoneBody(mtsTakeHomePurchaseTab, mtsTakeHomeSelectedNo))
	}
}

func handleTakeHomeFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventTakeHomeFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventTakeHomeFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeTakeHomeFailed {
			return
		}
		l.Debugf("MTS take-home failed for character [%d] serial [%d] (reason [%d]).", e.Body.CharacterId, e.Body.Serial, e.Body.Reason)
		announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationMoveItcPurchaseItemLtoSFailedBody(e.Body.Reason))
	}
}

// handleListingSold writes the BuyItemDone result to the buyer when a listing
// settles to a purchase (the buy / buy-now success notice). LISTING_SOLD is emitted
// by atlas-mts's settle path (the saga move step / auction settle), carrying the
// buyer (or auction winner) as BuyerId.
func handleListingSold(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingSoldBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingSoldBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingSold {
			return
		}
		l.Debugf("MTS listing sold to buyer [%d] (item [%d]).", e.Body.BuyerId, e.Body.ItemId)
		announceTo(l, ctx, sc, wp, e.Body.BuyerId, fieldpkt.MtsOperationBuyItemDoneBody())
	}
}

// handleBuyFailed writes the BuyItemFailed result to the buyer when a buy / buy-now
// is rejected (serial unresolved, listing not active, or insufficient prepaid).
func handleBuyFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBuyFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBuyFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBuyFailed {
			return
		}
		l.Debugf("MTS buy failed for buyer [%d] serial [%d] (reason [%d]).", e.Body.BuyerId, e.Body.Serial, e.Body.Reason)
		announceTo(l, ctx, sc, wp, e.Body.BuyerId, fieldpkt.MtsOperationBuyItemFailedBody())
	}
}

// handleBidFailed writes the BidAuctionFailed result to the bidder when a place-bid
// is rejected (serial unresolved, not an active auction, below floor, or lost race).
func handleBidFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBidFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBidFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBidFailed {
			return
		}
		l.Debugf("MTS bid failed for bidder [%d] serial [%d] (reason [%d]).", e.Body.BidderId, e.Body.Serial, e.Body.Reason)
		announceTo(l, ctx, sc, wp, e.Body.BidderId, fieldpkt.MtsOperationBidAuctionFailedBody())
	}
}
