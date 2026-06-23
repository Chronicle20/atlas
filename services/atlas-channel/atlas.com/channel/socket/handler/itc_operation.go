package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	mtsmsg "atlas-channel/kafka/message/mts"
	mtsproc "atlas-channel/mts"
	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ITC_OPERATION operation KEYs. These are the reverse-lookup keys into the
// tenant "operations" table (options["operations"][KEY] -> mode byte). The mode
// bytes themselves are NEVER hard-coded here: the incoming dispatcher mode byte
// is reverse-resolved to one of these KEYs via the config table (mirroring
// isMessengerShopOperation in messenger_operation.go), then dispatched. Verified
// table (template_gms_*_1.json ITC_OPERATION options.operations):
//
//	REGISTER_SALE:2 SALE_CURRENT_ITEM:3 REGISTER_WISH_ENTRY:4 GET_ITC_LIST:5
//	SEARCH_ITC_LIST:6 CANCEL_SALE:7 TAKE_HOME:8 SET_ZZIM:9 DELETE_ZZIM:10
//	VIEW_WISH:11 BUY_WISH:12 CANCEL_WISH:13 BUY:16 BUY_ZZIM:17 REGISTER_AUCTION:18
//	PLACE_BID:19 BUY_AUCTION_IMM:20
const (
	ItcOperationRegisterSale      = "REGISTER_SALE"
	ItcOperationSaleCurrentItem   = "SALE_CURRENT_ITEM"
	ItcOperationRegisterWishEntry = "REGISTER_WISH_ENTRY"
	ItcOperationGetItcList        = "GET_ITC_LIST"
	ItcOperationSearchItcList     = "SEARCH_ITC_LIST"
	ItcOperationCancelSale        = "CANCEL_SALE"
	ItcOperationTakeHome          = "TAKE_HOME"
	ItcOperationSetZzim           = "SET_ZZIM"
	ItcOperationDeleteZzim        = "DELETE_ZZIM"
	ItcOperationViewWish          = "VIEW_WISH"
	ItcOperationBuyWish           = "BUY_WISH"
	ItcOperationCancelWish        = "CANCEL_WISH"
	ItcOperationBuy               = "BUY"
	ItcOperationBuyZzim           = "BUY_ZZIM"
	ItcOperationRegisterAuction   = "REGISTER_AUCTION"
	ItcOperationPlaceBid          = "PLACE_BID"
	ItcOperationBuyAuctionImm     = "BUY_AUCTION_IMM"
)

// itcOperationKeys is the full set of routable KEYs. The dispatcher reverse-
// resolves the incoming mode byte against the tenant table for each of these and
// dispatches to the first match.
var itcOperationKeys = []string{
	ItcOperationRegisterSale,
	ItcOperationSaleCurrentItem,
	ItcOperationRegisterWishEntry,
	ItcOperationGetItcList,
	ItcOperationSearchItcList,
	ItcOperationCancelSale,
	ItcOperationTakeHome,
	ItcOperationSetZzim,
	ItcOperationDeleteZzim,
	ItcOperationViewWish,
	ItcOperationBuyWish,
	ItcOperationCancelWish,
	ItcOperationBuy,
	ItcOperationBuyZzim,
	ItcOperationRegisterAuction,
	ItcOperationPlaceBid,
	ItcOperationBuyAuctionImm,
}

// itcSaleType mirrors atlas-mts's listing.SaleType wire string ("fixed"/"auction").
// The channel sends the string; atlas-mts maps it onto its SaleType enum. Kept as
// string constants (not a new domain type) since this is the wire contract only.
const (
	itcSaleTypeFixed   = "fixed"
	itcSaleTypeAuction = "auction"
)

// resolveItcOperationKey reverse-resolves a dispatcher mode byte to its
// operation KEY via the tenant "operations" table (options["operations"]). It is
// the inverse of the config-driven mode resolution used by the clientbound
// writers (WithResolvedCode) and mirrors isMessengerShopOperation's forward
// lookup — NO mode byte is hard-coded. Returns ("", false) when the byte does
// not map to any configured KEY (an unrouted/unknown mode).
func resolveItcOperationKey(l logrus.FieldLogger) func(options map[string]interface{}, mode byte) (string, bool) {
	return func(options map[string]interface{}, mode byte) (string, bool) {
		genericCodes, ok := options["operations"]
		if !ok {
			l.Errorf("ITC_OPERATION has no configured operations table.")
			return "", false
		}
		codes, ok := genericCodes.(map[string]interface{})
		if !ok {
			l.Errorf("ITC_OPERATION operations table is malformed.")
			return "", false
		}
		for _, key := range itcOperationKeys {
			res, ok := codes[key].(float64)
			if !ok {
				continue
			}
			if byte(res) == mode {
				return key, true
			}
		}
		return "", false
	}
}

// CreateListingArgs is the resolved, processor-ready argument set produced from a
// register-sale / register-auction / sale-current-item packet plus the session
// identity. Extracting the mapping into a pure function keeps the decode->command
// translation unit-testable without a session/Kafka.
type CreateListingArgs struct {
	WorldId         world.Id
	SellerId        uint32
	SellerAccountId uint32
	SellerName      string
	SaleType        string
	// TemplateId/CashId identify the item the seller is listing. The register
	// packet's GW_ItemSlotBase blob carries the templateId (and, for cash items,
	// the cashId) but NOT the inventory slot/asset id — GW_ItemSlotBase has no
	// position field on the wire (see model.Asset.Decode). emitCreateListing
	// resolves these into SourceInventoryType + AssetId from the seller's live
	// inventory, so those two fields are populated downstream, not here.
	TemplateId          uint32
	CashId              int64
	SourceInventoryType byte
	AssetId             uint32
	Quantity            uint32
	ListValue           uint32
	BuyNowPrice         *uint32
	DurationHours       int
	Category            string
	SubCategory         string
}

// buildCreateListingFromRegisterSale maps the verified ItcOperationRegisterSale
// (mode 2, fixed-price) onto CreateListingArgs. The item is identified by the
// blob's templateId (and cashId, for cash items); the inventory type and real
// asset id are resolved from the seller's live inventory in emitCreateListing —
// the packet's itemType byte is the MTS category, not an inventory type, and the
// blob carries no slot. Category/SubCategory are left empty — the register packet
// carries no category, and atlas-mts categorizes server-side from the item; the
// browse filter treats empty as unfiltered. Price is the packet's price (NX list
// value).
func buildCreateListingFromRegisterSale(p fieldsb.ItcOperationRegisterSale, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	return CreateListingArgs{
		WorldId:         worldId,
		SellerId:        sellerId,
		SellerAccountId: sellerAccountId,
		SellerName:      sellerName,
		SaleType:        itcSaleTypeFixed,
		TemplateId:      p.Item().TemplateId(),
		CashId:          p.Item().CashId(),
		Quantity:        p.Quantity(),
		ListValue:       p.Price(),
		BuyNowPrice:     nil,
		DurationHours:   0,
	}
}

// buildCreateListingFromRegisterAuction maps the verified
// ItcOperationRegisterAuction (mode 0x12) onto CreateListingArgs. The auction
// packet carries a buy-now price and a duration (hours) but NO separate
// starting/list price field (verified read order: quantity, commodityId, selector,
// buyNowPrice, itemType, flag, durationHrs). The buy-now price doubles as the
// list/reserve value (atlas-mts uses ListValue as the auction's starting floor and
// the seller credit on settle). atlas-mts validates the duration against its
// [min,max] range.
func buildCreateListingFromRegisterAuction(p fieldsb.ItcOperationRegisterAuction, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	buyNow := p.BuyNowPrice()
	return CreateListingArgs{
		WorldId:         worldId,
		SellerId:        sellerId,
		SellerAccountId: sellerAccountId,
		SellerName:      sellerName,
		SaleType:        itcSaleTypeAuction,
		TemplateId:      p.Item().TemplateId(),
		CashId:          p.Item().CashId(),
		Quantity:        p.Quantity(),
		ListValue:       buyNow,
		BuyNowPrice:     &buyNow,
		DurationHours:   int(p.DurationHrs()),
	}
}

// buildCreateListingFromSaleCurrentItem maps the verified
// ItcOperationSaleCurrentItem (mode 3, sell currently-selected item) onto
// CreateListingArgs. The item is identified by the blob's templateId/cashId and
// resolved against the seller's live inventory in emitCreateListing (the wire's
// slotPos/itemType are not an asset id / inventory type). The packet carries no
// price field — SaleCurrentItem is the "list at the previously-entered price"
// follow-up; with no price on the wire the channel sends ListValue 0 and
// atlas-mts rejects it against the price floor (a clean RegisterSaleEntryFailed)
// rather than guessing a price.
func buildCreateListingFromSaleCurrentItem(p fieldsb.ItcOperationSaleCurrentItem, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	return CreateListingArgs{
		WorldId:         worldId,
		SellerId:        sellerId,
		SellerAccountId: sellerAccountId,
		SellerName:      sellerName,
		SaleType:        itcSaleTypeFixed,
		TemplateId:      p.Item().TemplateId(),
		CashId:          p.Item().CashId(),
		Quantity:        p.Item().Quantity(),
		ListValue:       0,
		BuyNowPrice:     nil,
		DurationHours:   0,
	}
}

// browseFilterFromGetItcList maps the verified ItcOperationChangedPage (mode 5,
// the full 8-field GET_ITC_LIST browse request) onto the channel REST BrowseFilter.
// Only the page is carried into the filter; the wire category/subCategory are
// numeric indices with no verified string mapping, so they are not used as
// equality filters (an unmatched numeric->string filter would return an empty page
// rather than the catalog). They are still echoed back into the result page.
func browseFilterFromGetItcList(p fieldsb.ItcOperationChangedPage) mtslisting.BrowseFilter {
	return mtslisting.BrowseFilter{
		Page: int(p.Page()),
	}
}

// browseFilterFromSearchItcList maps the verified ItcOperationTabSearch (mode 6,
// the SEARCH_ITC_LIST request) onto the REST BrowseFilter. The search name is the
// seller-name search term.
func browseFilterFromSearchItcList(p fieldsb.ItcOperationTabSearch) mtslisting.BrowseFilter {
	return mtslisting.BrowseFilter{
		SellerName: p.SearchName(),
	}
}

// mtsItemFromListing maps one channel-side listing.Model to a clientbound MtsItem
// (ITCITEM) for the GetItcListDone page. The item-slot blob carries the template
// id, quantity, and slot; the MTS trailer carries itcSn (= the listing's serial),
// price, and the auction bid metadata. The contract-fee / rollback / user-id
// strings are empty (the channel surfaces no such state) and the date-expired
// FILETIME is zero.
func mtsItemFromListing(m mtslisting.Model) fieldcb.MtsItem {
	// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83
	// client's GW_ItemSlotBase::Decode reads the item type byte first, with NO
	// leading inventory-slot byte). Passing false prepends a slot byte that the
	// client misreads as the item type, mis-decodes the rest of the item, and
	// overruns a later DecodeStr -> client crash on browse.
	item := packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	var dateExpired [8]byte
	return fieldpkt.MtsOperationNewItem(
		item,            // GW_ItemSlotBase blob
		m.ItcSn(),       // nITCSN = the listing serial (addresses buy/cancel/bid)
		m.ListValue(),   // nPrice
		0,               // nContractFee
		"",              // sContractFeeTxId
		"",              // sRollbackUsageID
		dateExpired,     // ftITCDateExpired
		"",              // sUserID
		m.SellerName(),  // sGameID (seller display name)
		"",              // sComment
		0,               // nBidCount
		m.MinIncrement(), // nBidRange
		m.CurrentBid(),  // nBidPrice
		m.ListValue(),   // nMinPrice
		m.BuyNowPrice(), // nMaxPrice
		m.ListValue(),   // nUnitPrice
		0,               // nProcessStatus
	)
}

// sellerName resolves the requesting character's display name for the listing
// (atlas-mts stores it for the browse seller column). A lookup failure yields ""
// rather than blocking the list — atlas-mts treats an empty seller name as
// acceptable and the browse column simply shows blank.
func sellerName(l logrus.FieldLogger, ctx context.Context, s session.Model) string {
	c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve seller name for character [%d]; listing with empty name.", s.CharacterId())
		return ""
	}
	return c.Name()
}

// ItcOperationHandleFunc is the ITC_OPERATION mode dispatcher. It peeks the leading
// mode byte, reverse-resolves it to an operation KEY against the tenant
// "operations" table, then re-seeks the reader to the mode byte so the chosen
// per-arm body codec (whose Decode reads mode-then-body in one pass) decodes from
// the right position. Wired arms (task-102): list (REGISTER_SALE/REGISTER_AUCTION/
// SALE_CURRENT_ITEM), CANCEL_SALE, TAKE_HOME (each emits a COMMAND_TOPIC_MTS
// command; the result is written by the channel EVENT_TOPIC_MTS_STATUS consumer),
// and the synchronous browse arms (GET_ITC_LIST/SEARCH_ITC_LIST) which query
// atlas-mts REST and write GetItcListDone inline. The remaining arms
// (wish/zzim/buy/bid) are routed-but-unimplemented seams for sibling tasks; an
// unconfigured mode byte is logged.
func ItcOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		// Record the mode-byte position so each arm's body codec (which re-reads the
		// mode byte first) decodes from the correct offset after we peek the mode.
		start := r.Position()
		p := fieldsb.ItcOperation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		key, ok := resolveItcOperationKey(l)(readerOptions, p.Mode())
		if !ok {
			l.Warnf("Character [%d] sent ITC_OPERATION with unconfigured/unknown mode byte [%d].", s.CharacterId(), p.Mode())
			return
		}

		// Re-seek to the mode byte: the per-arm body codecs Decode mode-then-body in
		// one pass (the dispatcher-family contract), so they must start at the mode.
		r.Seek(start)

		switch key {
		case ItcOperationRegisterSale:
			body := &fieldsb.ItcOperationRegisterSale{}
			body.Decode(l, ctx)(r, readerOptions)
			emitCreateListing(l, ctx, s, buildCreateListingFromRegisterSale(*body, s.WorldId(), s.CharacterId(), s.AccountId(), sellerName(l, ctx, s)))
		case ItcOperationRegisterAuction:
			body := &fieldsb.ItcOperationRegisterAuction{}
			body.Decode(l, ctx)(r, readerOptions)
			emitCreateListing(l, ctx, s, buildCreateListingFromRegisterAuction(*body, s.WorldId(), s.CharacterId(), s.AccountId(), sellerName(l, ctx, s)))
		case ItcOperationSaleCurrentItem:
			body := &fieldsb.ItcOperationSaleCurrentItem{}
			body.Decode(l, ctx)(r, readerOptions)
			emitCreateListing(l, ctx, s, buildCreateListingFromSaleCurrentItem(*body, s.WorldId(), s.CharacterId(), s.AccountId(), sellerName(l, ctx, s)))
		case ItcOperationCancelSale:
			body := &fieldsb.ItcOperationCancelSale{}
			body.Decode(l, ctx)(r, readerOptions)
			if err := mtsproc.NewProcessor(l, ctx).CancelListing(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId()); err != nil {
				l.WithError(err).Errorf("Unable to emit CANCEL_LISTING for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationTakeHome:
			body := &fieldsb.ItcOperationMoveLtoS{}
			body.Decode(l, ctx)(r, readerOptions)
			// The wire carries only nITCSN; the destination inventory type/slot are
			// resolved by atlas-mts's accept_to_character expansion (InventoryType 0 /
			// slot 0 are advisory placeholders — the saga assigns a free slot).
			if err := mtsproc.NewProcessor(l, ctx).TakeHome(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), 0, 0); err != nil {
				l.WithError(err).Errorf("Unable to emit TAKE_HOME for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationGetItcList:
			body := &fieldsb.ItcOperationChangedPage{}
			body.Decode(l, ctx)(r, readerOptions)
			// requestSent=1: this GetItcListDone answers a latching client request
			// (CITC::OnChangedCategory/Sub/Page set m_bITCRequestSent=this[6]=1 before
			// SendPacket and refuse any further ITC request until it clears).
			// CITC::OnGetITCListDone (v83 0x5a48af) clears the latch ONLY when its
			// trailing Decode1 byte is nonzero (`result=Decode1; if(result) this[6]=0`).
			// Sending 0 left the latch set, freezing the next tab — the reported bug.
			writeBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), body.Page(), body.SortType(), body.SortColumn(), 1, browseFilterFromGetItcList(*body))
		case ItcOperationSearchItcList:
			body := &fieldsb.ItcOperationTabSearch{}
			body.Decode(l, ctx)(r, readerOptions)
			// SEARCH surfaces hits in the same GetItcListDone result view. Same latch
			// contract as GET_ITC_LIST: send requestSent=1 so OnGetITCListDone clears
			// m_bITCRequestSent and the next tab/search is not blocked.
			writeBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), 0, 0, 0, 1, browseFilterFromSearchItcList(*body))
		case ItcOperationBuy:
			body := &fieldsb.ItcOperationBuy{}
			body.Decode(l, ctx)(r, readerOptions)
			// Plain fixed-price buy (mode 0x10): BuyNow=false. atlas-mts resolves the
			// serial->listing and settles at the listing's listValue.
			if err := mtsproc.NewProcessor(l, ctx).Buy(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), false); err != nil {
				l.WithError(err).Errorf("Unable to emit BUY for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationBuyAuctionImm:
			body := &fieldsb.ItcOperationBuyAuctionImm{}
			body.Decode(l, ctx)(r, readerOptions)
			// Buy-now / immediate-buyout of an auction (mode 0x14): identical wire
			// shape to BUY (serial only), distinguished by BuyNow=true so atlas-mts
			// settles at the listing's buyNowPrice (not its auction list/starting
			// value) — grounded in listing.Processor.Buy's BuyNow price-basis branch.
			if err := mtsproc.NewProcessor(l, ctx).Buy(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), true); err != nil {
				l.WithError(err).Errorf("Unable to emit BUY_AUCTION_IMM for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationPlaceBid:
			body := &fieldsb.ItcOperationPlaceBid{}
			body.Decode(l, ctx)(r, readerOptions)
			// PLACE_BID (mode 0x13): the wire carries itcSn + bidPrice + bidRange. The
			// bid amount is bidPrice (the player's base bid); bidRange is the client's
			// increment hint and is not part of the server-authoritative floor (atlas-mts
			// validates against currentBid + the listing's configured minIncrement).
			if err := mtsproc.NewProcessor(l, ctx).PlaceBid(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), body.BidPrice()); err != nil {
				l.WithError(err).Errorf("Unable to emit PLACE_BID for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationSetZzim:
			body := &fieldsb.ItcOperationSetZzim{}
			body.Decode(l, ctx)(r, readerOptions)
			// SET_ZZIM favorites a listing: resolve the wire serial -> listing to read
			// its templateId, then register a wish for that item. The SetZzimDone/Failed
			// result is written by the status consumer from the WISH_ADDED event
			// (Origin SET_ZZIM). A serial that does not resolve writes SetZzimFailed.
			lm, err := mtslisting.NewProcessor(l, ctx).GetBySerial(s.WorldId(), body.ItcSn())
			if err != nil {
				l.WithError(err).Errorf("Unable to resolve serial [%d] for SET_ZZIM, character [%d].", body.ItcSn(), s.CharacterId())
				writeWishFailure(l, ctx, wp, s, fieldpkt.MtsOperationSetZzimFailedBody())
				return
			}
			if err := mtsproc.NewProcessor(l, ctx).RegisterWish(s.WorldId(), s.CharacterId(), lm.TemplateId(), mtsmsg.WishOriginSetZzim); err != nil {
				l.WithError(err).Errorf("Unable to emit SET_ZZIM RegisterWish for character [%d] item [%d].", s.CharacterId(), lm.TemplateId())
				writeWishFailure(l, ctx, wp, s, fieldpkt.MtsOperationSetZzimFailedBody())
			}
		case ItcOperationDeleteZzim:
			body := &fieldsb.ItcOperationDeleteZzim{}
			body.Decode(l, ctx)(r, readerOptions)
			emitRemoveWishBySerial(l, ctx, wp, s, body.ItcSn(), mtsmsg.WishOriginDeleteZzim, fieldpkt.MtsOperationDeleteZzimFailedBody())
		case ItcOperationCancelWish:
			body := &fieldsb.ItcOperationCancelWish{}
			body.Decode(l, ctx)(r, readerOptions)
			// CANCEL_WISH is sent from the WISH sub-tab (IDA: sub_5BC1D5 case 4,
			// v4[27]!=0 -> CITC::OnCancelWish, which Encode4s the wish ITCITEM's
			// nITCSN). VIEW_WISH renders each wish entry's own per-(tenant, world)
			// serial into that nITCSN, so the wire serial resolves DIRECTLY to a wish
			// entry (NOT a listing) — unlike DELETE_ZZIM, whose favorites tab shows
			// real listings. emitRemoveWishByWishSerial does that wish-identity resolve.
			emitRemoveWishByWishSerial(l, ctx, wp, s, body.ItcSn(), mtsmsg.WishOriginCancelWish, fieldpkt.MtsOperationCancelWishFailedBody())
		case ItcOperationViewWish:
			body := &fieldsb.ItcOperationViewWish{}
			body.Decode(l, ctx)(r, readerOptions)
			// VIEW_WISH is a synchronous read: query the character's wishlist over REST
			// and write LoadWishSaleListDone inline (no status event).
			writeWishList(l, ctx, wp, s)
		case ItcOperationRegisterWishEntry:
			body := &fieldsb.ItcOperationRegisterWishEntry{}
			body.Decode(l, ctx)(r, readerOptions)
			// REGISTER_WISH_ENTRY creates a wish request by criteria. The wish domain
			// (task 1.6) models only (characterId, itemId), so the wire price/count/
			// duration/feeOption/desc fields are intentionally NOT persisted — only the
			// itemId is stored. The RegisterWishItemDone/Failed result is written by the
			// status consumer from the WISH_ADDED event (Origin REGISTER_WISH).
			if body.Price() != 0 || body.Count() != 0 || body.Duration() != 0 || body.FeeOption() != 0 || body.Description() != "" {
				l.Debugf("REGISTER_WISH_ENTRY for character [%d] item [%d] dropped unmodeled fields (price [%d] count [%d] duration [%d] fee [%d] desc [%q]).", s.CharacterId(), body.ItemId(), body.Price(), body.Count(), body.Duration(), body.FeeOption(), body.Description())
			}
			if err := mtsproc.NewProcessor(l, ctx).RegisterWish(s.WorldId(), s.CharacterId(), body.ItemId(), mtsmsg.WishOriginRegisterWish); err != nil {
				l.WithError(err).Errorf("Unable to emit REGISTER_WISH_ENTRY RegisterWish for character [%d] item [%d].", s.CharacterId(), body.ItemId())
				writeWishFailure(l, ctx, wp, s, fieldpkt.MtsOperationRegisterWishItemFailedBody())
			}
		case ItcOperationBuyZzim:
			body := &fieldsb.ItcOperationBuyZzim{}
			body.Decode(l, ctx)(r, readerOptions)
			// BUY_ZZIM buys a favorited listing (mode 0x11): same serial-only wire shape
			// as BUY, routed into the Buy flow (BuyNow=false). atlas-mts resolves the
			// serial -> listing and settles at listValue. The success/failure result is
			// the shared LISTING_SOLD/BUY_FAILED status path (BuyItemDone/Failed).
			if err := mtsproc.NewProcessor(l, ctx).Buy(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), false); err != nil {
				l.WithError(err).Errorf("Unable to emit BUY_ZZIM for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		case ItcOperationBuyWish:
			body := &fieldsb.ItcOperationBuyWish{}
			body.Decode(l, ctx)(r, readerOptions)
			// BUY_WISH (mode 0x0C) is sent from the wish-list modal (IDA: sub_5AB5DE
			// -> CITC::OnBuyWish, v83 0x59fa94, Encode4 of the wish ITCITEM's nITCSN).
			// That modal renders the character's OWN wish entries — items they WANT,
			// with no listing behind them — so the echoed nITCSN is now a WISH serial,
			// not a listing serial. There is therefore no buyable listing to settle
			// against from the wish tab; routing the wish serial through the Buy flow
			// makes listing.GetBySerial miss and emit a clean BuyFailed (the correct
			// outcome: a wished item is not a listing you can buy from the wish view).
			// Buying an actual listing of a wished item happens from the browse/favorites
			// tabs (BUY / BUY_ZZIM), which carry real listing serials.
			if err := mtsproc.NewProcessor(l, ctx).Buy(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), false); err != nil {
				l.WithError(err).Errorf("Unable to emit BUY_WISH for character [%d] serial [%d].", s.CharacterId(), body.ItcSn())
			}
		default:
			l.Warnf("Character [%d] sent ITC_OPERATION with resolved-but-unrouted KEY [%s] (mode [%d]).", s.CharacterId(), key, p.Mode())
		}
	}
}

// emitCreateListing emits the CREATE_LISTING command from the resolved args. A
// produce error is logged (the client receives no result; atlas-mts emits the
// RegisterSaleEntryFailed via the status consumer only when it actually processes
// and rejects the command).
func emitCreateListing(l logrus.FieldLogger, ctx context.Context, s session.Model, args CreateListingArgs) {
	// Resolve the item to a concrete inventory type + asset id from the seller's
	// live inventory. The register packet conveys only the item's templateId
	// (and cashId for cash items) — never the inventory slot or the inventory
	// type — so the channel must look the item up to give atlas-mts/the saga a
	// real asset to release. Without this the saga's transfer_to_mts step fails
	// to find the asset (id 0 / category byte as type) and the listing is lost.
	invType, ok := inventory.TypeFromItemId(item.Id(args.TemplateId))
	if !ok {
		l.Errorf("Unable to derive inventory type for MTS listing template [%d] for character [%d]; aborting list.", args.TemplateId, s.CharacterId())
		return
	}
	assetId, found := resolveSellerAssetId(l, ctx, s.CharacterId(), invType, args.TemplateId, args.CashId)
	if !found {
		l.Errorf("Character [%d] tried to list template [%d] (cashId [%d]) not present in inventory type [%d]; aborting list.", s.CharacterId(), args.TemplateId, args.CashId, invType)
		return
	}
	args.SourceInventoryType = byte(invType)
	args.AssetId = assetId

	if err := mtsproc.NewProcessor(l, ctx).CreateListing(
		uuid.New(),
		args.WorldId,
		args.SellerId,
		args.SellerAccountId,
		args.SellerName,
		args.SaleType,
		args.SourceInventoryType,
		args.AssetId,
		args.Quantity,
		args.ListValue,
		args.BuyNowPrice,
		args.DurationHours,
		args.Category,
		args.SubCategory,
	); err != nil {
		l.WithError(err).Errorf("Unable to emit CREATE_LISTING for character [%d].", s.CharacterId())
	}
}

// resolveSellerAssetId looks up the seller's inventory compartment of the given
// type and returns the real asset id (the DB asset id atlas-mts releases) of the
// item the seller is listing. The item is matched by templateId; for cash items
// (cashId != 0) the cashId disambiguates a specific cash asset. Returns (0, false)
// if the compartment lookup fails or no matching asset is present.
func resolveSellerAssetId(l logrus.FieldLogger, ctx context.Context, characterId uint32, invType inventory.Type, templateId uint32, cashId int64) (uint32, bool) {
	comp, err := compartment.NewProcessor(l, ctx).GetByType(characterId, invType)
	if err != nil {
		l.WithError(err).Errorf("Unable to load inventory type [%d] for character [%d] while resolving MTS listing asset.", invType, characterId)
		return 0, false
	}
	for _, a := range comp.Assets() {
		if a.TemplateId() == templateId && (cashId == 0 || a.CashId() == cashId) {
			return a.Id(), true
		}
	}
	return 0, false
}

// writeBrowsePage queries atlas-mts REST for the listing page and writes the
// synchronous GetItcListDone result to the requesting session. Each MtsItem.itcSn
// is the listing's serial (from the REST itcSn). On a REST error an empty page is
// written so the client UI is not left hanging.
//
// requestSent is the trailing Decode1 byte read by CITC::OnGetITCListDone
// (v83 0x5a48af: `result = Decode1(a2); if (result) this[6] = 0`). The client
// LATCHES m_bITCRequestSent (this[6]=1) the moment it sends a GET_ITC_LIST /
// SEARCH request (CITC::OnChangedCategory/Sub/Page, v83 0x59f297/0x59f376/
// 0x59f465) and refuses to send any further ITC request until that latch clears.
// The ONLY thing that clears it for this response is a nonzero trailing byte, so
// EVERY GetItcListDone that answers a latching client request MUST pass 1 — a 0
// leaves the latch set and freezes the next tab. The entry path also passes 1
// (cosmetic there, since entry is server-initiated and never sets the latch).
func writeBrowsePage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, requestSent byte, f mtslisting.BrowseFilter) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(s.WorldId(), f)
	if err != nil {
		l.WithError(err).Errorf("Unable to browse MTS listings for character [%d]; writing empty page.", s.CharacterId())
		ms = nil
	}

	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtsItemFromListing(m))
	}

	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), category, subCategory, page, sortType, sortColumn, items, requestSent)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS browse page to character [%d].", s.CharacterId())
	}
}

// writeWishFailure announces a wish/zzim *Failed clientbound result inline. The
// command-driven wish arms (SET_ZZIM / REGISTER_WISH_ENTRY) rely on the status
// consumer for their success/fail result; this writes the failure synchronously
// when the channel cannot even emit the command (serial unresolved / produce
// error), so the client is not left waiting on a status event that will never come.
func writeWishFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS wish failure to character [%d].", s.CharacterId())
	}
}

// emitRemoveWishBySerial resolves a wire listing serial to the wish entry the
// character has for that listing's item, then emits a REMOVE_WISH command tagged
// with the originating arm (DELETE_ZZIM or CANCEL_WISH) so the status consumer
// writes the matching result. The wire carries only the serial, never the wish
// UUID, so the channel resolves serial -> listing.templateId -> the character's
// wish entry for that item. A resolution failure writes the supplied *Failed body.
func emitRemoveWishBySerial(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, serial uint32, origin string, failBody func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
	lm, err := mtslisting.NewProcessor(l, ctx).GetBySerial(s.WorldId(), serial)
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve serial [%d] for wish-remove (origin [%s]), character [%d].", serial, origin, s.CharacterId())
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	wm, err := mtswish.NewProcessor(l, ctx).GetByCharacterItem(s.CharacterId(), lm.TemplateId())
	if err != nil {
		l.WithError(err).Errorf("Character [%d] has no wish entry for item [%d] (serial [%d], origin [%s]).", s.CharacterId(), lm.TemplateId(), serial, origin)
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	wishId, err := uuid.Parse(wm.Id())
	if err != nil {
		l.WithError(err).Errorf("Wish entry id [%s] is not a valid uuid (character [%d], origin [%s]).", wm.Id(), s.CharacterId(), origin)
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	if err := mtsproc.NewProcessor(l, ctx).RemoveWish(s.WorldId(), wishId, s.CharacterId(), origin); err != nil {
		l.WithError(err).Errorf("Unable to emit RemoveWish [%s] for character [%d] (origin [%s]).", wishId.String(), s.CharacterId(), origin)
		writeWishFailure(l, ctx, wp, s, failBody)
	}
}

// emitRemoveWishByWishSerial resolves a WISH serial (the nITCSN the client echoed
// from the wish-tab ITCITEM that VIEW_WISH populated with the wish entry's own
// serial) directly to the wish entry, then emits REMOVE_WISH tagged with the
// origin (CANCEL_WISH). Unlike emitRemoveWishBySerial (DELETE_ZZIM), the wire
// serial here is a wish serial, NOT a listing serial — a wish has no listing — so
// it resolves via the wishlist (GetByCharacterSerial) with no listing lookup. A
// resolution failure (no wish for that serial, or the stale itcSn=0) writes the
// supplied *Failed body.
func emitRemoveWishByWishSerial(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, serial uint32, origin string, failBody func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
	wm, err := mtswish.NewProcessor(l, ctx).GetByCharacterSerial(s.CharacterId(), serial)
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve wish serial [%d] for CANCEL_WISH (origin [%s]), character [%d].", serial, origin, s.CharacterId())
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	wishId, err := uuid.Parse(wm.Id())
	if err != nil {
		l.WithError(err).Errorf("Wish entry id [%s] is not a valid uuid (character [%d], origin [%s]).", wm.Id(), s.CharacterId(), origin)
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	if err := mtsproc.NewProcessor(l, ctx).RemoveWish(s.WorldId(), wishId, s.CharacterId(), origin); err != nil {
		l.WithError(err).Errorf("Unable to emit RemoveWish [%s] for character [%d] (origin [%s]).", wishId.String(), s.CharacterId(), origin)
		writeWishFailure(l, ctx, wp, s, failBody)
	}
}

// writeWishList queries the character's wishlist over REST and writes the
// synchronous LoadWishSaleListDone result. Each wish entry renders as a minimal
// ITCITEM carrying only the wished item template (the wish domain stores no price/
// sale metadata). On a REST error an empty list is written so the client UI is not
// left hanging.
func writeWishList(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacter(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Unable to load wishlist for character [%d]; writing empty list.", s.CharacterId())
		ws = nil
	}

	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		items = append(items, mtsItemFromWish(w))
	}

	body := fieldpkt.MtsOperationLoadWishSaleListDoneBody(items)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS wish list to character [%d].", s.CharacterId())
	}
}

// mtsItemFromWish maps one wish entry to a minimal clientbound MtsItem for the
// LoadWishSaleListDone list. The wish domain carries the item template plus its
// own per-(tenant, world) ITC serial: the item-slot blob carries the template id
// (quantity 1), the MTS trailer carries the wish serial as nITCSN (the rest is
// zeroed — a wish has no price/bid metadata).
//
// Writing the wish serial as nITCSN is the H5 fix: the client echoes nITCSN back
// verbatim on CANCEL_WISH (IDA: CITC::OnCancelWish, v83 0x59fb07, Encode4 of the
// item's offset-0x20 nITCSN field), so a nonzero wish serial lets the channel
// resolve the cancel back to this wish entry. Writing 0 (the old behavior) meant
// the client always sent 0 and the cancel never resolved.
func mtsItemFromWish(w mtswish.Model) fieldcb.MtsItem {
	// zeroPosition=true: bare GW_ItemSlotBase blob, no leading slot byte (see
	// mtsItemFromListing — the v83 client crashes on a slot-prefixed ITCITEM).
	item := packetmodel.NewAsset(true, 0, w.ItemId(), time.Time{}).SetStackableInfo(1, 0, 0)
	var dateExpired [8]byte
	return fieldpkt.MtsOperationNewItem(
		item,        // GW_ItemSlotBase blob
		w.Serial(),  // nITCSN (the wish entry's per-(tenant, world) ITC serial)
		0,           // nPrice
		0,           // nContractFee
		"",          // sContractFeeTxId
		"",          // sRollbackUsageID
		dateExpired, // ftITCDateExpired
		"",          // sUserID
		"",          // sGameID
		"",          // sComment
		0,           // nBidCount
		0,           // nBidRange
		0,           // nBidPrice
		0,           // nMinPrice
		0,           // nMaxPrice
		0,           // nUnitPrice
		0,           // nProcessStatus
	)
}
