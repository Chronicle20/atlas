package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	dataitem "atlas-channel/data/item"
	mtsmsg "atlas-channel/kafka/message/mts"
	mtsproc "atlas-channel/mts"
	mtscart "atlas-channel/mts/cart"
	mtslisting "atlas-channel/mts/listing"
	mtstransaction "atlas-channel/mts/transaction"
	mtswish "atlas-channel/mts/wish"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
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
	SlotPos             uint32 // inventory slot of the listed item (from the register packet) — used to resolve the exact asset
	SourceInventoryType byte
	AssetId             uint32
	Quantity            uint32
	ListValue           uint32
	BuyNowPrice         *uint32
	DurationHours       int
	MinIncrement        uint32 // auction only; the seller's bid increment (0 => atlas-mts uses the tenant default)
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
		SlotPos:         p.SlotPos(),
		Quantity:        p.Quantity(),
		ListValue:       p.Price(),
		BuyNowPrice:     nil,
		DurationHours:   0,
	}
}

// buildCreateListingFromRegisterAuction maps the verified
// ItcOperationRegisterAuction (mode 0x12) onto CreateListingArgs. The auction
// packet carries TWO prices (a starting bid and a buy-now price) plus a duration
// (hours); the read order is quantity, startingBid (the "selector" int), buyNow,
// itemType, flag, durationHrs. The lower price becomes the auction's ListValue
// (first-bid floor / seller credit on settle) and the higher its buy-now ceiling
// — see the body for why. atlas-mts validates the duration against its [min,max]
// range.
func buildCreateListingFromRegisterAuction(p fieldsb.ItcOperationRegisterAuction, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	// The auction register dialog (CITC::OnRegisterSaleEntry -> sub_5AD76B,
	// v83 0x59ec36 / 0x5ad76b) collects TWO prices — a starting bid and a buy-now
	// price — and the client guarantees buyNow > startingBid (the SP_4765 "buy now
	// price is lower than the starting bid" validation). Both arrive on the wire:
	// the "selector" int (formerly assumed a constant ==1, but sub_5AD76B overwrites
	// it with a dialog price) and the buyNowPrice int are those two prices. The
	// lower is the starting bid — the auction's listValue / first-bid floor — and
	// the higher is the buy-now ceiling. Sending the buyNowPrice as the listValue
	// made the first-bid floor equal the buyout, so atlas-mts rejected every bid
	// below the buyout and the client showed SP_4760 "you cannot make a consecutive
	// bid" (the fixed BidAuctionFailed message). min/max is order-independent and
	// respects the client's buyNow>startingBid invariant.
	startingBid, buyNow := p.Selector(), p.BuyNowPrice()
	if startingBid > buyNow {
		startingBid, buyNow = buyNow, startingBid
	}
	return CreateListingArgs{
		WorldId:         worldId,
		SellerId:        sellerId,
		SellerAccountId: sellerAccountId,
		SellerName:      sellerName,
		SaleType:        itcSaleTypeAuction,
		TemplateId:      p.Item().TemplateId(),
		CashId:          p.Item().CashId(),
		SlotPos:         p.SlotPos(),
		Quantity:        p.Quantity(),
		ListValue:       startingBid,
		BuyNowPrice:     &buyNow,
		DurationHours:   int(p.DurationHrs()), // the Encode1 byte — the auction DURATION (task-102 field-label fix)
		MinIncrement:    p.MinIncrement(),     // the trailing Encode4 — the seller's bid increment
	}
}

// buildCreateListingFromSaleCurrentItem maps the verified
// ItcOperationSaleCurrentItem (mode 3, the want-ad OFFER) onto CreateListingArgs.
// The item is identified by the blob's templateId/cashId and resolved against the
// seller's live inventory in emitCreateListing (the wire's slotPos/itemType are
// not an asset id / inventory type). listValue is the want-ad's asking price
// (resolved from the offer's target wishSerial), so the offered item is listed as
// a fixed sale at the price the want-ad poster offered to pay; the poster then
// buys it through the normal browse/buy path.
func buildCreateListingFromSaleCurrentItem(p fieldsb.ItcOperationSaleCurrentItem, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string, listValue uint32) CreateListingArgs {
	return CreateListingArgs{
		WorldId:         worldId,
		SellerId:        sellerId,
		SellerAccountId: sellerAccountId,
		SellerName:      sellerName,
		SaleType:        itcSaleTypeFixed,
		TemplateId:      p.Item().TemplateId(),
		CashId:          p.Item().CashId(),
		SlotPos:         p.SlotPos(),
		Quantity:        p.Item().Quantity(),
		ListValue:       listValue,
		BuyNowPrice:     nil,
		DurationHours:   0,
	}
}

// resolveWantedPrice looks up the asking price of a want-ad by its per-(tenant,
// world) ITC serial (the wishSerial a SALE_CURRENT_ITEM offer targets). It scans
// the world's want-ads for the matching serial. Returns (0, false) when the serial
// does not resolve (a stale/invalid offer target) so the caller skips the listing
// rather than creating a free (price 0) sale.
func resolveWantedPrice(l logrus.FieldLogger, ctx context.Context, worldId world.Id, wishSerial uint32) (uint32, bool) {
	ws, err := mtswish.NewProcessor(l, ctx).GetWantedByWorld(byte(worldId))
	if err != nil {
		l.WithError(err).Errorf("Unable to load world want-ads to resolve offer target serial [%d].", wishSerial)
		return 0, false
	}
	for _, w := range ws {
		if w.Serial() == wishSerial {
			return w.Price(), true
		}
	}
	return 0, false
}

// Marketplace sections (client browse "tab"). Section 2 (Wanted) and the section
// 4 / sub 0 Cart view are wish-backed (the character's own wanted/cart entries),
// not public listings; the rest are listing browses.
const (
	itcSectionWanted uint32 = 2
	itcSectionCart   uint32 = 4
)

// applyItcViewFilters maps the client's GET_ITC_LIST/SEARCH browse selectors onto
// the REST listing filter. The wire carries two values that mirror the marketplace
// data model (listings are stored with the same two fields at AcceptToMtsListing):
//
//	category    -> the marketplace SECTION / top tab: 1=For Sale (fixed), 3=Auction.
//	               Listings store this in `category`, so section 1 shows fixed sales,
//	               section 3 auctions, and sections 2/4/5 (wanted/my-page/cart) hold
//	               no sale listings -> empty.
//	categorySub -> the item sub-tab: 0=all, 1=equip, 2=use, 3=setup, 4=etc. Listings
//	               store the item's inventory category in `subCategory`, so a USE item
//	               only surfaces under For Sale -> Use (and All).
//
// A straight (category, subCategory) equality filter — no per-view special-casing.
func applyItcViewFilters(f *mtslisting.BrowseFilter, category uint32, categorySub uint32) {
	f.Category = strconv.FormatUint(uint64(category), 10)
	if categorySub != 0 {
		f.SubCategory = strconv.FormatUint(uint64(categorySub), 10)
	}
}

// browseFilterFromGetItcList maps the verified ItcOperationChangedPage (mode 5,
// the full 8-field GET_ITC_LIST browse request) onto the channel REST BrowseFilter:
// the page plus the view (category -> saleType) and item sub-tab (categorySub ->
// inventory-type category) filters.
func browseFilterFromGetItcList(p fieldsb.ItcOperationChangedPage) mtslisting.BrowseFilter {
	f := mtslisting.BrowseFilter{Page: int(p.Page())}
	applyItcViewFilters(&f, p.Category(), p.CategorySub())
	return f
}

// browseFilterFromSearchItcList maps the verified ItcOperationTabSearch (mode 6,
// the SEARCH_ITC_LIST request) onto the REST BrowseFilter. The search term is an
// ITEM NAME: it is resolved to the matching item template ids via atlas-data's
// item-string search index, and the browse is filtered on those ids (scoped to the
// same view + sub-tab filters as a normal browse).
//
// The bool result is hasResults: false means the search term was non-empty but
// matched no items, so the caller MUST short-circuit to an empty page rather than
// running the (unfiltered) browse — an empty TemplateIds slice would otherwise
// return all listings. true means either the search resolved to one or more ids
// (TemplateIds populated) or the term was empty (an unfiltered browse of the view).
// A resolve error is treated as zero matches (false) so a transient atlas-data
// failure shows no results instead of leaking the whole marketplace.
func browseFilterFromSearchItcList(l logrus.FieldLogger, ctx context.Context, p fieldsb.ItcOperationTabSearch) (mtslisting.BrowseFilter, bool) {
	f := mtslisting.BrowseFilter{}
	applyItcViewFilters(&f, p.Category(), p.CategorySub())

	name := p.SearchName()
	if name == "" {
		// No search term: browse the view unfiltered.
		return f, true
	}

	ids, err := dataitem.NewProcessor(l, ctx).GetIdsByName(name)
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve MTS search term [%q] to item ids; showing no results.", name)
		return f, false
	}
	if len(ids) == 0 {
		return f, false
	}
	f.TemplateIds = ids
	return f, true
}

// mtsItemFromListing maps one channel-side listing.Model to a clientbound MtsItem
// (ITCITEM) for the GetItcListDone page. The item-slot blob carries the template
// id, quantity, and slot; the MTS trailer carries itcSn (= the listing's serial),
// price, the auction bid metadata, and the "Sold Until" date (auction end or a
// far-future sentinel for fixed listings — see mtslisting.ToMtsItem).
func mtsItemFromListing(m mtslisting.Model) fieldcb.MtsItem {
	// Delegates to the shared mtslisting.ToMtsItem so the browse arm and the
	// consumer's post-event "Not Yet Sold" re-push produce identical wire bytes
	// (including the zeroPosition=true bare item blob — see ToMtsItem).
	return mtslisting.ToMtsItem(m)
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
			// SALE_CURRENT_ITEM (mode 3) is the want-ad OFFER (CITC picker "SEND ME
			// AN OFFER"): the player offers one of their items to fulfill want-ad
			// #WishSerial. Resolve the want-ad's asking price and list the offered
			// item at that price as a fixed sale the poster can buy. A want-ad serial
			// that does not resolve is logged and ignored (no free listing).
			price, ok := resolveWantedPrice(l, ctx, s.WorldId(), body.WishSerial())
			if !ok {
				l.Errorf("Character [%d] sent a want-ad offer (SALE_CURRENT_ITEM) for unresolved want-ad serial [%d]; ignoring.", s.CharacterId(), body.WishSerial())
				return
			}
			emitCreateListing(l, ctx, s, buildCreateListingFromSaleCurrentItem(*body, s.WorldId(), s.CharacterId(), s.AccountId(), sellerName(l, ctx, s), price))
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
			// TEMP (task-102 filtering bring-up): the client encodes the top-tab/view
			// in category and the inventory sub-tab in categorySub (CITC this[26]/[27],
			// consumed by the per-item action dispatch sub_5BC1D5). Log them so the
			// view->number mapping can be pinned before filtering is wired.
			l.Infof("[MTS browse] character [%d] GET_ITC_LIST category [%d] categorySub [%d] page [%d] searchOption [%d] searchCondition [%q].", s.CharacterId(), body.Category(), body.CategorySub(), body.Page(), body.SearchOption(), body.SearchCondition())
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
			// SEARCH is an ITEM-NAME search: the term is resolved to matching item
			// template ids via atlas-data and the browse is filtered on them. Hits
			// surface in the same GetItcListDone result view. Same latch contract as
			// GET_ITC_LIST: send requestSent=1 so OnGetITCListDone clears
			// m_bITCRequestSent and the next tab/search is not blocked.
			f, hasResults := browseFilterFromSearchItcList(l, ctx, *body)
			if !hasResults {
				// The term matched no items (or atlas-data failed). An empty
				// TemplateIds would browse UNFILTERED (all listings), so write an
				// empty page directly instead of running the browse.
				writeEmptyBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), 0, 0, 0, 1)
				return
			}
			writeBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), 0, 0, 0, 1, f)
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
			// player's intended bid is bidPrice + bidRange: CITCBidAuctionDlg keeps the
			// current base in this[40] (bidPrice) and the dialed-up increment in this[39]
			// (bidRange), and only sends a bid when this[40]+this[39] < buyNow
			// (OnButtonClicked, v83 0x5c373e). Sending bidPrice alone under-records the
			// bid and — for any re-bid where bidPrice == the prior bid — falls below the
			// currentBid + minIncrement floor, so atlas-mts rejects it as
			// BidAuctionFailed (client SP_4760 "you cannot make a consecutive bid").
			if err := mtsproc.NewProcessor(l, ctx).PlaceBid(uuid.New(), s.WorldId(), body.ItcSn(), s.CharacterId(), s.AccountId(), body.BidPrice()+body.BidRange()); err != nil {
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
			// You cannot cart your own listing — the cart is items you intend to buy
			// from others (mirrors the reference handler's seller<>self guard).
			if lm.SellerId() == s.CharacterId() {
				l.Debugf("Character [%d] tried to cart their own listing (serial [%d]); rejecting.", s.CharacterId(), body.ItcSn())
				writeWishFailure(l, ctx, wp, s, fieldpkt.MtsOperationSetZzimFailedBody())
				return
			}
			if err := mtsproc.NewProcessor(l, ctx).RegisterWish(s.WorldId(), s.CharacterId(), lm.TemplateId(), lm.ListValue(), mtsmsg.WishOriginSetZzim); err != nil {
				l.WithError(err).Errorf("Unable to emit SET_ZZIM RegisterWish for character [%d] item [%d].", s.CharacterId(), lm.TemplateId())
				writeWishFailure(l, ctx, wp, s, fieldpkt.MtsOperationSetZzimFailedBody())
			}
		case ItcOperationDeleteZzim:
			body := &fieldsb.ItcOperationDeleteZzim{}
			body.Decode(l, ctx)(r, readerOptions)
			// Remove-from-cart: the Cart now renders each favorited item's live
			// LISTING (itcSn = listing serial, see mts/cart.Items), so the wire serial
			// is a LISTING serial — resolve it to the listing's item and remove the
			// character's CART wish for that item (type-scoped so a wanted entry for
			// the same item is never touched).
			emitRemoveCartWishByListingSerial(l, ctx, wp, s, body.ItcSn())
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
			// models (characterId, itemId, price); the wire price is persisted so the
			// want-ad view shows the real price. The remaining count/duration/feeOption/
			// desc fields are still unmodeled and intentionally NOT persisted. The
			// RegisterWishItemDone/Failed result is written by the status consumer from
			// the WISH_ADDED event (Origin REGISTER_WISH).
			if body.Count() != 0 || body.Duration() != 0 || body.FeeOption() != 0 || body.Description() != "" {
				l.Debugf("REGISTER_WISH_ENTRY for character [%d] item [%d] dropped unmodeled fields (count [%d] duration [%d] fee [%d] desc [%q]).", s.CharacterId(), body.ItemId(), body.Count(), body.Duration(), body.FeeOption(), body.Description())
			}
			if err := mtsproc.NewProcessor(l, ctx).RegisterWish(s.WorldId(), s.CharacterId(), body.ItemId(), body.Price(), mtsmsg.WishOriginRegisterWish); err != nil {
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
	assetId, found := resolveSellerAssetId(l, ctx, s.CharacterId(), invType, args.SlotPos, args.TemplateId, args.CashId)
	if !found {
		l.Errorf("Character [%d] tried to list template [%d] (cashId [%d]) at slot [%d] not present in inventory type [%d]; aborting list.", s.CharacterId(), args.TemplateId, args.CashId, args.SlotPos, invType)
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
		args.MinIncrement,
		args.Category,
		args.SubCategory,
	); err != nil {
		l.WithError(err).Errorf("Unable to emit CREATE_LISTING for character [%d].", s.CharacterId())
	}
}

// resolveSellerAssetId looks up the seller's inventory compartment of the given
// type and returns the real asset id (the DB asset id atlas-mts releases) of the
// item the seller is listing. The register packet carries the item's inventory
// SLOT (slotPos), which uniquely identifies the selected stack — so the primary
// match is by slot, with templateId (and cashId for cash items) as a consistency
// guard against a stale/mismatched slot. If the slot does not resolve, fall back
// to the first asset matching templateId (+cashId). Returns (0, false) if the
// compartment lookup fails or nothing matches.
func resolveSellerAssetId(l logrus.FieldLogger, ctx context.Context, characterId uint32, invType inventory.Type, slotPos uint32, templateId uint32, cashId int64) (uint32, bool) {
	comp, err := compartment.NewProcessor(l, ctx).GetByType(characterId, invType)
	if err != nil {
		l.WithError(err).Errorf("Unable to load inventory type [%d] for character [%d] while resolving MTS listing asset.", invType, characterId)
		return 0, false
	}
	for _, a := range comp.Assets() {
		if uint32(a.Slot()) == slotPos && a.TemplateId() == templateId && (cashId == 0 || a.CashId() == cashId) {
			return a.Id(), true
		}
	}
	// Slot did not resolve (e.g. moved between selection and send) — fall back to
	// the first stack matching the item identity.
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
	var items []fieldcb.MtsItem
	switch {
	case category == itcSectionCart && subCategory == 0:
		// My Page -> Cart (section 4 / sub 0): the viewer's added-to-cart entries
		// (SET_ZZIM). Each cart entry is rendered as its favorited item's current
		// best active LISTING (nITCSN = listing serial, all-in price via ToMtsItem's
		// contract fee) so BUY_ZZIM / DELETE_ZZIM address a real listing and the
		// price shows with fees — see mts/cart.Items.
		items = mtscart.Items(l, ctx, s.WorldId(), s.CharacterId())
	case category == itcSectionCart && subCategory == 1:
		// My Page -> Offers (section 4 / sub 1): the viewer's OWN want-ads
		// (REGISTER_WISH_ENTRY -> wish type=wanted).
		items = wishItems(l, ctx, s.CharacterId(), mtswish.TypeWanted, 0)
	case category == itcSectionCart && subCategory == 2:
		// My Page -> History (section 4 / sub 2): the viewer's settled
		// purchase/sale log, read from atlas-mts. On a REST error write an empty
		// page rather than blocking the client UI.
		ts, err := mtstransaction.NewProcessor(l, ctx).GetByCharacter(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve MTS transaction history for character [%d]; writing empty page.", s.CharacterId())
			ts = nil
		}
		items = make([]fieldcb.MtsItem, 0, len(ts))
		for _, m := range ts {
			items = append(items, mtstransaction.ToMtsItem(m))
		}
	case category == itcSectionCart && subCategory == 3:
		// My Page -> Auction (section 4 / sub 3): the viewer's OWN auction listings.
		items = ownAuctionItems(l, ctx, s)
	case category == itcSectionWanted:
		// Wanted (section 2): ALL want-ads in the world, across every character
		// (NOT just the viewer's own). Each MtsItem carries the wish serial as
		// nITCSN, the wish price, and the owner's character name as the seller column.
		items = wantedWorldItems(l, ctx, s)
	default:
		// Public marketplace browse: listings filtered by (section=category,
		// item-type=subCategory), excluding the requesting character's OWN listings
		// (For Sale / Auction show others' items; your own appear under My Page /
		// Not Yet Sold). Sections that hold no sale listings (wanted/my-page) return
		// an empty page.
		//
		// The browse is UNPAGED (PageSize=-1): the client builds its page
		// selector from categoryItemCnt as ceil(total/16)
		// (CITCWnd_List::ChangeCategorySub, v83 0x5BDD12), so the total must
		// count EVERY match, not one page's slice — the requested 16-item
		// window is cut below, uniformly for every view.
		f.ExcludeSellerId = s.CharacterId()
		f.Page = 0
		f.PageSize = -1
		ms, err := mtslisting.NewProcessor(l, ctx).Browse(s.WorldId(), f)
		if err != nil {
			l.WithError(err).Errorf("Unable to browse MTS listings for character [%d]; writing empty page.", s.CharacterId())
			ms = nil
		}
		items = make([]fieldcb.MtsItem, 0, len(ms))
		for _, m := range ms {
			items = append(items, mtsItemFromListing(m))
		}
	}

	// categoryItemCnt carries the TOTAL match count (drives the client's page
	// selector); the packet's item list carries only the requested 16-item
	// window. This also holds for the full-list arms above (cart/wanted/
	// history/own-auction), which previously stuffed every row into one page.
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), category, subCategory, page, sortType, sortColumn, mtsPageWindow(items, page), requestSent)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS browse page to character [%d].", s.CharacterId())
	}

	// Every browse also re-pushes the seller's own panels — "Not Yet Sold" (active
	// listings) and "Transfer Inventory" (take-home holdings) — so they stay current
	// after any operation. The v83 client does not re-request these itself; this
	// mirrors the reference handler, which re-sends both on every interaction and is
	// the real fix for the panels lagging after a cancel/take-home.
	announceUserSaleItems(l, ctx, wp, s)
	announceUserPurchaseItems(l, ctx, wp, s)
}

// writeEmptyBrowsePage writes a GetItcListDone result with zero items for the
// given view, then re-pushes the seller's own panels (as writeBrowsePage does). It
// is used by SEARCH_ITC_LIST when the search term matches no items: an empty
// TemplateIds filter would browse unfiltered (all listings), so the empty result
// must be written directly rather than via the browse path. The latch contract is
// identical to writeBrowsePage — requestSent MUST be 1 so OnGetITCListDone clears
// m_bITCRequestSent and the next request is not frozen.
func writeEmptyBrowsePage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, requestSent byte) {
	items := make([]fieldcb.MtsItem, 0)
	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), category, subCategory, page, sortType, sortColumn, items, requestSent)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce empty MTS search page to character [%d].", s.CharacterId())
	}
	announceUserSaleItems(l, ctx, wp, s)
	announceUserPurchaseItems(l, ctx, wp, s)
}

// wishItems loads a character's wish entries of one kind (cart/wanted) and maps
// them to ITCITEMs for the Cart / Wanted views. When categorySub is non-zero the
// list is narrowed to that inventory category (derived from each wish's itemId),
// so the Wanted sub-tabs (equip/use/...) filter like the public browse does.
func wishItems(l logrus.FieldLogger, ctx context.Context, characterId uint32, wishType string, categorySub uint32) []fieldcb.MtsItem {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, wishType)
	if err != nil {
		l.WithError(err).Errorf("Unable to load %s wishes for character [%d]; writing empty page.", wishType, characterId)
		ws = nil
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		if categorySub != 0 {
			if it, ok := inventory.TypeFromItemId(item.Id(w.ItemId())); !ok || uint32(it) != categorySub {
				continue
			}
		}
		items = append(items, mtsItemFromWish(w))
	}
	return items
}

// ownAuctionItems loads the viewer's OWN active auction listings (My Page ->
// Auction, section 4 / sub 3) and maps each to an ITCITEM. Unlike the public
// Auction tab (section 3, which excludes the viewer's own listings), this view is
// scoped to the viewer as the seller, so it browses with SellerId set and no
// ExcludeSellerId. A REST error yields an empty list rather than blocking the page.
func ownAuctionItems(l logrus.FieldLogger, ctx context.Context, s session.Model) []fieldcb.MtsItem {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(s.WorldId(), mtslisting.BrowseFilter{
		SellerId: s.CharacterId(),
		SaleType: itcSaleTypeAuction,
	})
	if err != nil {
		l.WithError(err).Errorf("Unable to browse own MTS auctions for character [%d]; writing empty page.", s.CharacterId())
		ms = nil
	}
	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtsItemFromListing(m))
	}
	return items
}

// wantedWorldItems loads ALL want-ads in the viewer's world, across every
// character (the cross-character Wanted tab, section 2), and maps each to an
// ITCITEM that carries the wish serial as nITCSN, the wish price, and the want-ad
// owner's character name as the seller column. A REST error yields an empty list
// rather than blocking the page.
func wantedWorldItems(l logrus.FieldLogger, ctx context.Context, s session.Model) []fieldcb.MtsItem {
	ws, err := mtswish.NewProcessor(l, ctx).GetWantedByWorld(byte(s.WorldId()))
	if err != nil {
		l.WithError(err).Errorf("Unable to load world want-ads for world [%d]; writing empty page.", byte(s.WorldId()))
		ws = nil
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		items = append(items, mtsWantedItem(l, ctx, w))
	}
	return items
}

// mtsWantedItem maps one cross-character want-ad to an ITCITEM, resolving the
// owner's display name into the seller (sGameID) column. The name lookup is
// best-effort: a failure yields "" (a blank seller column) and the item is STILL
// included — a want-ad must never be dropped because its owner's name could not
// be resolved. Mirrors the sellerName helper's error tolerance.
func mtsWantedItem(l logrus.FieldLogger, ctx context.Context, w mtswish.Model) fieldcb.MtsItem {
	ownerName := ""
	c, err := character.NewProcessor(l, ctx).GetById()(w.CharacterId())
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve owner name for want-ad character [%d]; rendering with empty seller column.", w.CharacterId())
	} else {
		ownerName = c.Name()
	}
	return mtswish.ToMtsItemWithSeller(w, ownerName)
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

// emitRemoveCartWishByListingSerial removes the character's CART wish for the item
// of the listing addressed by serial. The Cart renders each favorited item's live
// LISTING (nITCSN = listing serial, see mts/cart.Items), so DELETE_ZZIM carries a
// LISTING serial: resolve it to the listing's item, find the character's CART entry
// for that item (type-scoped so a wanted entry for the same item is never touched),
// and emit REMOVE_WISH tagged DELETE_ZZIM. A serial that no longer resolves (the
// favorited listing sold/expired) writes DeleteZzimFailed — the entry would already
// have dropped off the Cart on its next render.
func emitRemoveCartWishByListingSerial(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, serial uint32) {
	failBody := fieldpkt.MtsOperationDeleteZzimFailedBody()

	lm, err := mtslisting.NewProcessor(l, ctx).GetBySerial(s.WorldId(), serial)
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve listing serial [%d] for DELETE_ZZIM, character [%d].", serial, s.CharacterId())
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(s.CharacterId(), mtswish.TypeCart)
	if err != nil {
		l.WithError(err).Errorf("Unable to load cart entries for character [%d] to resolve DELETE_ZZIM (serial [%d]).", s.CharacterId(), serial)
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	var wm mtswish.Model
	found := false
	for _, w := range ws {
		if w.ItemId() == lm.TemplateId() {
			wm = w
			found = true
			break
		}
	}
	if !found {
		l.Errorf("Character [%d] has no cart entry for item [%d] (listing serial [%d]) on DELETE_ZZIM.", s.CharacterId(), lm.TemplateId(), serial)
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	wishId, err := uuid.Parse(wm.Id())
	if err != nil {
		l.WithError(err).Errorf("Cart wish id [%s] is not a valid uuid (character [%d]).", wm.Id(), s.CharacterId())
		writeWishFailure(l, ctx, wp, s, failBody)
		return
	}

	if err := mtsproc.NewProcessor(l, ctx).RemoveWish(s.WorldId(), wishId, s.CharacterId(), mtsmsg.WishOriginDeleteZzim); err != nil {
		l.WithError(err).Errorf("Unable to emit DELETE_ZZIM RemoveWish [%s] for character [%d].", wishId.String(), s.CharacterId())
		writeWishFailure(l, ctx, wp, s, failBody)
	}
}

// emitRemoveWishByWishSerial resolves a WISH serial (the nITCSN the client echoed
// from the wish-tab ITCITEM that VIEW_WISH populated with the wish entry's own
// serial) directly to the wish entry, then emits REMOVE_WISH tagged with the
// origin (CANCEL_WISH). Unlike emitRemoveCartWishByListingSerial (DELETE_ZZIM),
// the wire serial here is a wish serial, NOT a listing serial — a wish has no
// listing — so
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
// ITCITEM carrying the wished item template plus its price. On a REST error an
// empty list is written so the client UI is not left hanging.
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
	// Delegates to the shared mtswish.ToMtsItem so the VIEW_WISH arm and the
	// consumer's post-mutation Cart/Wanted re-push produce identical wire bytes.
	return mtswish.ToMtsItem(w)
}
