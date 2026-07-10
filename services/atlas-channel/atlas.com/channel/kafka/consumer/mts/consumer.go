package mts

import (
	consumer2 "atlas-channel/kafka/consumer"
	mtsmsg "atlas-channel/kafka/message/mts"
	"atlas-channel/listener"
	mtsproc "atlas-channel/mts"
	mtscart "atlas-channel/mts/cart"
	mtsholding "atlas-channel/mts/holding"
	mtslisting "atlas-channel/mts/listing"
	mtswanted "atlas-channel/mts/wanted"
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
	"github.com/google/uuid"
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

// mtsSaleTypeOffer is the sale-type wire string for a want-ad OFFER, mirroring
// atlas-mts's listing.SaleTypeOffer. A LISTING_CREATED/LISTING_SOLD carrying it
// routes to the want-ad clientbound results (SaleCurrentItemToWishDone / BuyWishDone)
// instead of the normal register/buy results.
const mtsSaleTypeOffer = "offer"

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
				if err := register(message.AdaptHandler(message.PersistentConfig(handleBidPlaced(sc, wp)))); err != nil {
					return nil, err
				}
				if err := register(message.AdaptHandler(message.PersistentConfig(handleBidFailed(sc, wp)))); err != nil {
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

// mtsSectionAuction is the CITC top-tab category for auctions (3 = Auction,
// 1 = For Sale). mtsBrowsePageSize is the client's 16-plates-per-page window,
// mirrored from the socket handler's paging so an unsolicited browse refresh
// windows to page 0 exactly as a client-requested browse does.
const (
	mtsSectionAuction uint32 = 3
	mtsBrowsePageSize        = 16
)

// announceBidderAuctionBrowse re-pushes the auction browse page (GET_ITC_LIST_DONE,
// category 3, page 0, the bidder's own listings excluded) to a bidder after their
// bid lands, so the new high bid and incremented bid count show without re-entering
// the MTS. The v83 bid dialog closes itself after sending and never re-requests the
// list, so the server pushes the refreshed page. categoryItemCnt carries the TOTAL
// match count (drives the client's page selector, ceil(total/16)); the packet's item
// list carries page 0's 16-item window. requestSent=1 is harmless here — no client
// request latch is set for a server-initiated push.
func announceBidderAuctionBrowse(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, worldId byte, bidderId uint32) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(world.Id(worldId), mtslisting.BrowseFilter{Category: "3", ExcludeSellerId: bidderId, Page: 0, PageSize: -1})
	if err != nil {
		l.WithError(err).Errorf("Unable to refresh MTS auction list for bidder [%d]; leaving the browse view stale.", bidderId)
		return
	}
	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtslisting.ToMtsItem(m))
	}
	window := items
	if len(window) > mtsBrowsePageSize {
		window = window[:mtsBrowsePageSize]
	}
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), mtsSectionAuction, 0, 0, 1, 1, window, 1)
	announceTo(l, ctx, sc, wp, bidderId, body)
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
func announceWishList(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, worldId byte, characterId uint32, section uint32, wishType string) {
	var items []fieldcb.MtsItem
	if wishType == mtswish.TypeCart {
		// The Cart renders each favorited item's live LISTING (nITCSN = listing
		// serial, all-in price) so BUY_ZZIM / DELETE_ZZIM address a real listing —
		// see mts/cart.Items. The re-push must match the browse arm's rendering.
		items = mtscart.Items(l, ctx, world.Id(worldId), characterId)
	} else {
		// The Wanted view (section 2) is the cross-character list MINUS the viewer's
		// own want-ads — identical to the browse arm. Rendering the viewer's OWN
		// want-ads here (the old behavior) made a poster see their own ad in the
		// Wanted tab after posting/cancelling (task-102 live finding).
		items = mtswanted.WorldItems(l, ctx, world.Id(worldId), characterId)
	}
	// section as the browse category, sub 0 (all), page 0, sortType/sortColumn 1,
	// requestSent 1 (mirrors the entry browse — and clears the latch, see above).
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), section, 0, 0, 1, 1, items, 1)
	announceTo(l, ctx, sc, wp, characterId, body)
}

// announceOwnWantAds re-pushes the character's OWN want-ads (My Page -> Offers,
// section 4 / sub 1) as a GetItcListDone so the panel reflects a just-consumed
// want-ad without the poster re-entering MTS. Unlike announceWishList's Wanted arm
// (the cross-character world list MINUS the viewer), this renders the character's
// OWN wanted wishes (mtswish.TypeWanted via ToMtsItem). The trailing requestSent=1
// clears the client's m_bITCRequestSent latch, mirroring the entry browse. On a
// REST error the panel is left stale rather than pushing an empty list.
func announceOwnWantAds(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, worldId byte, characterId uint32) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, mtswish.TypeWanted)
	if err != nil {
		l.WithError(err).Errorf("Unable to refresh MTS own want-ads for character [%d]; leaving the My-Page Offers panel stale.", characterId)
		return
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		items = append(items, mtswish.ToMtsItem(w))
	}
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), mtsSectionCart, 1, 0, 1, 1, items, 1)
	announceTo(l, ctx, sc, wp, characterId, body)
}

// wishSectionForOrigin maps a wish-mutation origin to the MTS section + wish type
// whose view should be re-pushed: SET_ZZIM/DELETE_ZZIM act on the Cart, while
// REGISTER_WISH/CANCEL_WISH act on the Wanted ads. An unknown origin returns
// ok=false so the caller skips the re-push rather than guessing a section.
func wishSectionForOrigin(origin string) (section uint32, wishType string, ok bool) {
	switch origin {
	case mtsmsg.WishOriginSetZzim, mtsmsg.WishOriginDeleteZzim, mtsmsg.WishOriginPurchased:
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
		l.Debugf("MTS listing created for seller [%d] (item [%d], saleType [%s]).", e.Body.SellerId, e.Body.ItemId, e.Body.SaleType)
		if e.Body.SaleType == mtsSaleTypeOffer {
			// A want-ad OFFER was escrowed (SALE_CURRENT_ITEM): confirm it to the
			// offerer with SaleCurrentItemToWishDone (the register dialog listens for a
			// different result) and refresh their Not-Yet-Sold panel where the escrowed
			// offer now sits.
			announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationSaleCurrentItemToWishDoneBody())
			announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
			return
		}
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
		// The register dialog listens ONLY for RegisterSaleEntryFailed — routing this
		// through failNoticeOr -> GetSearchItcListFailed (the search-list notice arm,
		// correct for buy/bid) leaves the register dialog stuck with no response
		// (task-102 live finding). Always send the register dialog's own failed arm so
		// it un-hangs; the client renders its own registration-failure message.
		announceTo(l, ctx, sc, wp, e.Body.SellerId, fieldpkt.MtsOperationRegisterSaleEntryFailedBody(mtsRegisterSaleGenericReason, mtsRegisterSaleNoSaleLimit))
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
		// off, AND the "Transfer Inventory" panel where the cancelled item now sits
		// (ready to take home) — both without re-entering MTS.
		announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
		announceUserPurchaseList(l, ctx, sc, wp, e.Body.SellerId)
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
		l.Debugf("MTS listing sold to buyer [%d] (item [%d], saleType [%s]).", e.Body.BuyerId, e.Body.ItemId, e.Body.SaleType)
		if e.Body.SaleType == mtsSaleTypeOffer {
			// A want-ad OFFER was accepted (BUY_WISH): the poster paid, the offered item
			// is now in their Transfer Inventory. Confirm with BuyWishDone (not
			// BuyItemDone) and refresh the buyer's purchase panel. The want-ad consume +
			// sibling-offer release are handled server-side by atlas-mts (the losing
			// offerers' panels are re-pushed by their per-offer LISTING_CANCELLED events).
			announceTo(l, ctx, sc, wp, e.Body.BuyerId, fieldpkt.MtsOperationBuyWishDoneBody())
			announceUserPurchaseList(l, ctx, sc, wp, e.Body.BuyerId)
			// Refresh the accepted OFFERER's Not-Yet-Sold: their offer listing is now
			// sold, so it drops off that panel.
			announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
			// Refresh the POSTER's My Page -> Offers: the want-ad they posted was
			// consumed by the accept, so it drops off that panel.
			announceOwnWantAds(l, ctx, sc, wp, e.Body.WorldId, e.Body.BuyerId)
			return
		}
		announceTo(l, ctx, sc, wp, e.Body.BuyerId, fieldpkt.MtsOperationBuyItemDoneBody())
		// Refresh the buyer's Transfer Inventory panel (the bought item now sits in
		// their holdings, ready to take home). The NX/points counter is refreshed
		// separately by the wallet-status consumer once the debit actually lands
		// (scene-gated), so it is not pushed here.
		announceUserPurchaseList(l, ctx, sc, wp, e.Body.BuyerId)
		// If the buyer had the purchased item in their Cart (SET_ZZIM favorite),
		// remove that cart entry now that they own it — a bought item should leave
		// the Cart. The removal round-trips through REMOVE_WISH(PURCHASED); the
		// resulting WISH_REMOVED re-pushes the Cart view (and drops the row) without a
		// client notice. A buy from the browse (no cart entry) is a no-op here.
		removeCartWishForPurchase(l, ctx, e.Body.WorldId, e.Body.BuyerId, e.Body.ItemId)
		// Refresh the seller's "Not Yet Sold" panel (the sold item leaves it). Their
		// wallet credit is handled by the wallet-status consumer. If the seller is
		// offline/elsewhere these no-op; their panels also re-push on their next browse.
		if e.Body.SellerId != 0 {
			announceUserSaleList(l, ctx, sc, wp, e.Body.WorldId, e.Body.SellerId)
		}
	}
}

// removeCartWishForPurchase removes the buyer's Cart (SET_ZZIM) entry for a
// just-purchased item so the bought item leaves the Cart. It resolves the buyer's
// cart wish for itemId and emits REMOVE_WISH tagged PURCHASED — a silent,
// server-initiated removal: handleWishRemoved writes no client notice for that
// origin but re-pushes the Cart view (dropping the row). A buy from the browse,
// where the buyer has no cart entry for the item, is a no-op.
func removeCartWishForPurchase(l logrus.FieldLogger, ctx context.Context, worldId byte, characterId uint32, itemId uint32) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, mtswish.TypeCart)
	if err != nil {
		l.WithError(err).Warnf("Unable to load cart entries for buyer [%d] to prune purchased item [%d]; leaving cart as-is.", characterId, itemId)
		return
	}
	for _, w := range ws {
		if w.ItemId() != itemId {
			continue
		}
		wishId, perr := uuid.Parse(w.Id())
		if perr != nil {
			l.WithError(perr).Errorf("Cart wish id [%s] is not a valid uuid (buyer [%d]); cannot prune purchased item.", w.Id(), characterId)
			return
		}
		if rerr := mtsproc.NewProcessor(l, ctx).RemoveWish(world.Id(worldId), wishId, characterId, mtsmsg.WishOriginPurchased); rerr != nil {
			l.WithError(rerr).Errorf("Unable to emit PURCHASED RemoveWish [%s] for buyer [%d].", wishId.String(), characterId)
		}
		return
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

// handleBidPlaced re-pushes the bidder's auction browse page after a bid is
// recorded (BID_PLACED), so the new high bid and incremented bid count appear
// in place. The v83 bid dialog closes itself after sending and never re-requests
// the list. The NX debit is refreshed separately by the wallet-status consumer
// once the escrow actually lands.
func handleBidPlaced(sc server.Model, wp writer.Producer) message.Handler[mtsmsg.StatusEvent[mtsmsg.StatusEventBidPlacedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mtsmsg.StatusEvent[mtsmsg.StatusEventBidPlacedBody]) {
		if e.Type != mtsmsg.StatusEventTypeBidPlaced {
			return
		}
		l.Debugf("MTS bid placed by bidder [%d] on listing [%s] (amount [%d]); refreshing auction browse.", e.Body.BidderId, e.Body.ListingId.String(), e.Body.Amount)
		announceBidderAuctionBrowse(l, ctx, sc, wp, e.Body.WorldId, e.Body.BidderId)
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
			announceWishList(l, ctx, sc, wp, e.Body.WorldId, e.Body.CharacterId, section, wishType)
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
		case mtsmsg.WishOriginPurchased:
			// Server-initiated cart prune after a purchase — no notice: BuyItemDone
			// already confirmed the buy. The Cart re-push below drops the bought row.
		default:
			l.Warnf("MTS WISH_REMOVED for character [%d] has unknown origin [%s]; no result written.", e.Body.CharacterId, e.Body.Origin)
		}
		// Re-push the Cart/Wanted view so the removed wish disappears and — critically
		// for DELETE_ZZIM — the trailing requestSent=1 clears the client's request
		// latch (DeleteZzimDone never clears it itself), which otherwise freezes the
		// client after a successful cart removal.
		if section, wishType, ok := wishSectionForOrigin(e.Body.Origin); ok {
			announceWishList(l, ctx, sc, wp, e.Body.WorldId, e.Body.CharacterId, section, wishType)
		}
	}
}
