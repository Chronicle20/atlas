package mts

import (
	consumer2 "atlas-channel/kafka/consumer"
	mtsmsg "atlas-channel/kafka/message/mts"
	"atlas-channel/listener"
	"atlas-channel/cashshop/wallet"
	mtsholding "atlas-channel/mts/holding"
	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

// mtsRegisterSaleGenericReason is the NoticeFailReason byte for the bare
// RegisterSaleEntryFailed arm used when a create-failure's reasonKey does not
// resolve in the tenant table (0 -> the client's generic MTS-failed notice).
const mtsRegisterSaleGenericReason byte = 0

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
				if err := register(message.AdaptHandler(message.PersistentConfig(handleBidPlaced(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleOutbid(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleWishAdded(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleWishRemoved(sc, wp)))); err != nil {
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

// announceUserSaleList re-pushes the seller's "Not Yet Sold" panel
// (GetUserSaleItemDone) by re-querying their active listings. The v83 client
// only loads this list once at MTS entry and never re-requests it after a
// registration/cancellation (RegisterSaleEntryDone just shows a notice and
// re-selects a tab — it does not re-query), so the server must push the fresh
// list for the panel to reflect a just-created or just-cancelled listing.
func announceUserSaleList(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, worldId byte, sellerId uint32) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(world.Id(worldId), mtslisting.BrowseFilter{SellerId: sellerId})
	if err != nil {
		l.WithError(err).Errorf("Unable to refresh MTS sale list for seller [%d]; leaving the Not-Yet-Sold panel stale.", sellerId)
		return
	}
	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtslisting.ToMtsItem(m))
	}
	announceTo(l, ctx, sc, wp, sellerId, fieldpkt.MtsOperationGetUserSaleItemDoneBody(items))
}

// announceUserPurchaseList re-pushes the character's "Transfer Inventory" panel
// (GetUserPurchaseItemDone) by re-querying their take-home holdings. Like the sale
// list, the v83 client loads this once at MTS entry and never re-requests it after
// a take-home, so a just-retrieved item lingers in the panel until the player
// re-enters MTS unless the server pushes the fresh list.
func announceUserPurchaseList(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32) {
	hs, err := mtsholding.NewProcessor(l, ctx).GetByCharacter(characterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to refresh MTS purchase list for character [%d]; leaving the Transfer-Inventory panel stale.", characterId)
		return
	}
	items := make([]fieldcb.MtsItem, 0, len(hs))
	for _, h := range hs {
		items = append(items, mtsholding.ToMtsItem(h))
	}
	announceTo(l, ctx, sc, wp, characterId, fieldpkt.MtsOperationGetUserPurchaseItemDoneBody(items, 0, 0))
}

// announceWalletRefresh re-pushes the MTS wallet display (MTS_OPERATION2: prepaid
// NX + maple points) to the character so the on-screen NX/points counter updates
// after a money-moving operation (a buy debit / sale credit). The v83 client only
// reads the wallet at entry, so without this the counter stays stale until
// re-entry. Resolves the character's session for its accountId (the wallet key).
func announceWalletRefresh(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	if !t.Is(sc.Tenant()) {
		return
	}
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to refresh MTS wallet for character [%d]; leaving the NX counter stale.", characterId)
			return nil
		}
		return session.Announce(l)(ctx)(wp)(fieldcb.MtsOperation2Writer)(fieldcb.NewMtsOperation2(w.Prepaid(), w.Points()).Encode)(s)
	})
}

// MTS browse sections (CITC category tab), mirrored from the socket handler's
// itcSection* constants. The Cart view (section 4 / sub 0) renders type=cart
// wishes; the Wanted view (section 2) renders type=wanted wishes.
const (
	mtsSectionWanted uint32 = 2
	mtsSectionCart   uint32 = 4
)

// announceWishList re-pushes the character's Cart or Wanted view as a
// GetItcListDone so the list reflects a just-added/removed wish AND the trailing
// requestSent=1 clears the client's m_bITCRequestSent latch. The latch matters
// because DeleteZzimDone (cart remove) shows its success notice but — unlike
// SetZzimDone — never clears the latch itself (CITC sub_5A4E66 vs sub_5A4DFC), so
// without this re-push the client freezes after a successful cart removal (IDA:
// CITC::OnGetITCListDone v83 0x5a48af clears this[6] only when requestSent != 0).
// The v83 client also never re-requests the wish list after a mutation, so the
// re-push is the only way the Cart/Wanted view updates without re-entering MTS.
func announceWishList(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, section uint32, wishType string) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, wishType)
	if err != nil {
		l.WithError(err).Errorf("Unable to refresh MTS %s list for character [%d]; leaving the view stale.", wishType, characterId)
		return
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		items = append(items, mtswish.ToMtsItem(w))
	}
	// section as the browse category, sub 0 (all), page 0, sortType/sortColumn 1,
	// requestSent 1 (mirrors the entry browse — and clears the latch, see above).
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), section, 0, 0, 1, 1, items, 1)
	announceTo(l, ctx, sc, wp, characterId, body)
}

// wishSectionForOrigin maps a wish-mutation origin to the MTS section + wish type
// whose view should be re-pushed: SET_ZZIM/DELETE_ZZIM act on the Cart, while
// REGISTER_WISH/CANCEL_WISH act on the Wanted ads. An unknown origin returns
// ok=false so the caller skips the re-push rather than guessing a section.
func wishSectionForOrigin(origin string) (section uint32, wishType string, ok bool) {
	switch origin {
	case mtsmsg.WishOriginSetZzim, mtsmsg.WishOriginDeleteZzim:
		return mtsSectionCart, mtswish.TypeCart, true
	case mtsmsg.WishOriginRegisterWish, mtsmsg.WishOriginCancelWish:
		return mtsSectionWanted, mtswish.TypeWanted, true
	default:
		return 0, "", false
	}
}

func handleListingCreated(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreatedBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCreated {
			return
		}
		l.Debugf("MTS listing created for seller [%d] (item [%d]).", e.Body.SellerId, e.Body.ItemId)
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationRegisterSaleEntryDoneBody())
		// Refresh the seller's "Not Yet Sold" panel so the new listing appears
		// without re-entering MTS (the client does not re-query it itself).
		announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
	}
}

func handleListingCreateFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreateFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCreateFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCreateFailed {
			return
		}
		l.Debugf("MTS listing create failed for seller [%d] (reasonKey [%s]).", e.Body.SellerId, e.Body.ReasonKey)
		announceTo(l, ctx, sc, wp, e.Body.SellerId, failNoticeOr(e.Body.ReasonKey, fieldpkt.MtsOperationRegisterSaleEntryFailedBody(mtsRegisterSaleGenericReason, mtsRegisterSaleNoSaleLimit)))
	}
}

func handleListingCancelled(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelledBody]) {
		if e.Type != mtsmsg.StatusEventTypeListingCancelled {
			return
		}
		l.Debugf("MTS listing cancelled for seller [%d] (item [%d] -> holding [%s]).", e.Body.SellerId, e.Body.ItemId, e.Body.HoldingId.String())
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationCancelSaleItemDoneBody())
		// Refresh the seller's "Not Yet Sold" panel so the cancelled listing drops
		// off without re-entering MTS.
		announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
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
		// Do NOT write MoveItcPurchaseItemLtoSDone here. ITEM_TAKEN_HOME fires when
		// atlas-mts soft-deletes the holding — BEFORE accept_to_character has granted
		// the item to inventory — so a Done sent now confirms a take-home whose
		// inventory grant has not landed (the client shows the confirmation but the
		// item slot is still empty). The authoritative Done is written once, from the
		// saga-COMPLETED path (saga/consumer.go announceMtsTakeHomeDone), which fires
		// only after both release + accept_to_character succeed. Emitting from both
		// paths double-confirmed the take-home (the first update no-op'd, the second
		// stuck) — task-102 live finding.
		//
		// This handler still refreshes the "Transfer Inventory" panel: the holding is
		// gone as of ITEM_TAKEN_HOME, so the retrieved item should drop off the panel
		// now without waiting for (or re-entering) MTS.
		announceUserPurchaseList(l, ctx, sc, wp, e.Body.CharacterId)
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
		// Refresh the buyer's view of the purchase: the NX/points counter (debited)
		// and the Transfer Inventory (the bought item now sits in their holdings,
		// ready to take home). Without these the client shows the buy succeeded but
		// the counter and panel stay stale until re-entry.
		announceWalletRefresh(l, ctx, sc, wp, e.Body.BuyerId)
		announceUserPurchaseList(l, ctx, sc, wp, e.Body.BuyerId)
		// Refresh the seller's view: the sold item leaves "Not Yet Sold" and their
		// wallet gains the sale credit (points). If the seller is offline/elsewhere
		// these no-op; their panels also re-push on their next browse.
		if e.Body.SellerId != 0 {
			announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
			announceWalletRefresh(l, ctx, sc, wp, e.Body.SellerId)
		}
	}
}

// failNoticeOr routes a SEMANTIC failure key (e.g. "NOT_ENOUGH_NX",
// "ITEM_SOLD" — set by atlas-mts on the *_FAILED event) through the tenant
// writer options table "noticeFailReasons" to the client's reason-notice arm —
// GetSearchItcListFailed (mode 24), whose sub-handler is Decode1(reason) ->
// CITC::NoticeFailReason -> latch clear, IDA-verified uniform across gms v83
// (0x5A49E3) / v84 (0x5B4ED3) / v87 / v95. Both the reason CODE and the arm's
// MODE byte are config-driven per version, like every other dispatcher value
// (the task-103 uniformity ruling). An empty key, a tenant without the table,
// or a key the table lacks all fall back to the operation's bare *Failed arm
// (its fixed generic notice) — never a 99-crash resolve.
func failNoticeOr(reasonKey string, bare func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if reasonKey == "" {
		return bare
	}
	return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			code, ok := noticeFailReasonCode(options, reasonKey)
			if !ok {
				l.Debugf("Tenant noticeFailReasons table lacks key [%s]; writing the bare failed arm.", reasonKey)
				return bare(l, ctx)(options)
			}
			return fieldpkt.MtsOperationGetSearchItcListFailedBody(code)(l, ctx)(options)
		}
	}
}

// noticeFailReasonCode soft-resolves options["noticeFailReasons"][key] (JSON
// numbers decode as float64, mirroring ResolveCode's accepted shapes) without
// ResolveCode's 99-on-miss contract: a missing table or key reports !ok so the
// caller can fall back instead of crashing the client.
func noticeFailReasonCode(options map[string]interface{}, key string) (byte, bool) {
	raw, ok := options["noticeFailReasons"]
	if !ok {
		return 0, false
	}
	table, ok := raw.(map[string]interface{})
	if !ok {
		return 0, false
	}
	v, ok := table[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return byte(n), true
	case int:
		return byte(n), true
	}
	return 0, false
}

// handleBuyFailed writes the BuyItemFailed result to the buyer when a buy / buy-now
// is rejected (serial unresolved, listing not active, or insufficient prepaid).
func handleBuyFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBuyFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBuyFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBuyFailed {
			return
		}
		l.Debugf("MTS buy failed for buyer [%d] serial [%d] (reasonKey [%s]).", e.Body.BuyerId, e.Body.Serial, e.Body.ReasonKey)
		announceTo(l, ctx, sc, wp, e.Body.BuyerId, failNoticeOr(e.Body.ReasonKey, fieldpkt.MtsOperationBuyItemFailedBody()))
	}
}

// handleBidPlaced refreshes the bidder's NX counter after a successful bid. Placing
// a bid escrows (debits) the bidder's prepaid in atlas-mts; the v83 client only
// reads the wallet at MTS entry, so without this push the on-screen NX stays stale
// until re-entry (task-102 live finding). No BidAuctionDone is written here — the
// bid's own client flow handles the dialog; this is purely the wallet refresh.
func handleBidPlaced(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBidPlacedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBidPlacedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBidPlaced {
			return
		}
		l.Debugf("MTS bid placed by bidder [%d] on listing [%s] (amount [%d]); refreshing wallet.", e.Body.BidderId, e.Body.ListingId.String(), e.Body.Amount)
		announceWalletRefresh(l, ctx, sc, wp, e.Body.BidderId)
	}
}

// handleOutbid refreshes the outbid bidder's NX counter: being outbid releases their
// escrow back to prepaid in atlas-mts, and the client won't reflect the refund until
// re-entry without this push (task-102 live finding — the refund "definitely" didn't
// show).
func handleOutbid(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventOutbidBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventOutbidBody]) {
		if e.Type != mtsmsg.StatusEventTypeOutbid {
			return
		}
		l.Debugf("MTS previous bidder [%d] outbid on listing [%s]; refreshing wallet (escrow released).", e.Body.PreviousBidderId, e.Body.ListingId.String())
		announceWalletRefresh(l, ctx, sc, wp, e.Body.PreviousBidderId)
	}
}

// handleBidFailed writes the BidAuctionFailed result to the bidder when a place-bid
// is rejected (serial unresolved, not an active auction, below floor, or lost race).
func handleBidFailed(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBidFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBidFailedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBidFailed {
			return
		}
		l.Debugf("MTS bid failed for bidder [%d] serial [%d] (reasonKey [%s]).", e.Body.BidderId, e.Body.Serial, e.Body.ReasonKey)
		announceTo(l, ctx, sc, wp, e.Body.BidderId, failNoticeOr(e.Body.ReasonKey, fieldpkt.MtsOperationBidAuctionFailedBody()))
	}
}

// mtsNotifyCancelWishCount is the count passed to NotifyCancelWishResult on a
// successful CANCEL_WISH. The clientbound codec gates a StringPool notice on each
// count being >0; 1 cancelled / 0 other shows the single "wish cancelled" notice.
const (
	mtsNotifyCancelWishCountA uint32 = 1
	mtsNotifyCancelWishCountB uint32 = 0
)

// handleWishAdded writes the wish-add result to the originating character. WISH_ADDED
// is emitted by atlas-mts's handleRegisterWish; Origin discriminates which ITC arm
// initiated the add so the channel writes the matching clientbound result
// (SET_ZZIM -> SetZzimDone, REGISTER_WISH -> RegisterWishItemDone). An unknown Origin
// is logged (no result written) rather than guessing a mode.
func handleWishAdded(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventWishAddedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventWishAddedBody]) {
		if e.Type != mtsmsg.StatusEventTypeWishAdded {
			return
		}
		l.Debugf("MTS wish added for character [%d] (item [%d], origin [%s]).", e.Body.CharacterId, e.Body.ItemId, e.Body.Origin)
		switch e.Body.Origin {
		case mtsmsg.WishOriginSetZzim:
			announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationSetZzimDoneBody())
		case mtsmsg.WishOriginRegisterWish:
			announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationRegisterWishItemDoneBody())
		default:
			l.Warnf("MTS WISH_ADDED for character [%d] has unknown origin [%s]; no result written.", e.Body.CharacterId, e.Body.Origin)
		}
		// Re-push the Cart/Wanted view so the just-added wish appears without the
		// player re-entering MTS (the v83 client never re-requests the list after a
		// SetZzimDone/RegisterWishItemDone notice).
		if section, wishType, ok := wishSectionForOrigin(e.Body.Origin); ok {
			announceWishList(l, ctx, sc, wp, e.Body.CharacterId, section, wishType)
		}
	}
}

// handleWishRemoved writes the wish-remove result to the originating character.
// WISH_REMOVED is emitted by atlas-mts's handleRemoveWish; Origin discriminates which
// ITC arm initiated the remove so the channel writes the matching clientbound result
// (DELETE_ZZIM -> DeleteZzimDone, CANCEL_WISH -> NotifyCancelWishResult). An unknown
// Origin is logged rather than guessing a mode.
func handleWishRemoved(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventWishRemovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventWishRemovedBody]) {
		if e.Type != mtsmsg.StatusEventTypeWishRemoved {
			return
		}
		l.Debugf("MTS wish removed for character [%d] (origin [%s]).", e.Body.CharacterId, e.Body.Origin)
		switch e.Body.Origin {
		case mtsmsg.WishOriginDeleteZzim:
			announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationDeleteZzimDoneBody())
		case mtsmsg.WishOriginCancelWish:
			announceTo(l, ctx, sc, wp, e.Body.CharacterId, fieldpkt.MtsOperationNotifyCancelWishResultBody(mtsNotifyCancelWishCountA, mtsNotifyCancelWishCountB))
		default:
			l.Warnf("MTS WISH_REMOVED for character [%d] has unknown origin [%s]; no result written.", e.Body.CharacterId, e.Body.Origin)
		}
		// Re-push the Cart/Wanted view so the removed wish disappears and — critically
		// for DELETE_ZZIM — the trailing requestSent=1 clears the client's request
		// latch (DeleteZzimDone never clears it itself), which otherwise freezes the
		// client after a successful cart removal.
		if section, wishType, ok := wishSectionForOrigin(e.Body.Origin); ok {
			announceWishList(l, ctx, sc, wp, e.Body.CharacterId, section, wishType)
		}
	}
}
