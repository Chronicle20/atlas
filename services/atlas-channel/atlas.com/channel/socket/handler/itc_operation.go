package handler

import (
	"atlas-channel/character"
	mtsproc "atlas-channel/mts"
	mtslisting "atlas-channel/mts/listing"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

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
	WorldId             world.Id
	SellerId            uint32
	SellerAccountId     uint32
	SellerName          string
	SaleType            string
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
// (mode 2, fixed-price) onto CreateListingArgs. AssetId is the item-slot blob's
// slot (the inventory slot atlas-mts removes); SourceInventoryType is the packet's
// itemType byte. Category/SubCategory are left empty — the register packet carries
// no category, and atlas-mts categorizes server-side from the item; the browse
// filter treats empty as unfiltered. Price is the packet's price (NX list value).
func buildCreateListingFromRegisterSale(p fieldsb.ItcOperationRegisterSale, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	return CreateListingArgs{
		WorldId:             worldId,
		SellerId:            sellerId,
		SellerAccountId:     sellerAccountId,
		SellerName:          sellerName,
		SaleType:            itcSaleTypeFixed,
		SourceInventoryType: p.ItemType(),
		AssetId:             uint32(p.Item().Slot()),
		Quantity:            p.Quantity(),
		ListValue:           p.Price(),
		BuyNowPrice:         nil,
		DurationHours:       0,
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
		WorldId:             worldId,
		SellerId:            sellerId,
		SellerAccountId:     sellerAccountId,
		SellerName:          sellerName,
		SaleType:            itcSaleTypeAuction,
		SourceInventoryType: p.ItemType(),
		AssetId:             uint32(p.Item().Slot()),
		Quantity:            p.Quantity(),
		ListValue:           buyNow,
		BuyNowPrice:         &buyNow,
		DurationHours:       int(p.DurationHrs()),
	}
}

// buildCreateListingFromSaleCurrentItem maps the verified
// ItcOperationSaleCurrentItem (mode 3, sell currently-selected item) onto
// CreateListingArgs. It is a fixed-price sale of the item at slotPos. The packet
// carries no price field — SaleCurrentItem is the "list at the previously-entered
// price" follow-up; with no price on the wire the channel sends ListValue 0 and
// atlas-mts rejects it against the price floor (a clean RegisterSaleEntryFailed)
// rather than guessing a price.
func buildCreateListingFromSaleCurrentItem(p fieldsb.ItcOperationSaleCurrentItem, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string) CreateListingArgs {
	return CreateListingArgs{
		WorldId:             worldId,
		SellerId:            sellerId,
		SellerAccountId:     sellerAccountId,
		SellerName:          sellerName,
		SaleType:            itcSaleTypeFixed,
		SourceInventoryType: p.ItemType(),
		AssetId:             p.SlotPos(),
		Quantity:            p.Item().Quantity(),
		ListValue:           0,
		BuyNowPrice:         nil,
		DurationHours:       0,
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
	item := packetmodel.NewAsset(false, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
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
			writeBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), body.Page(), body.SortType(), body.SortColumn(), browseFilterFromGetItcList(*body))
		case ItcOperationSearchItcList:
			body := &fieldsb.ItcOperationTabSearch{}
			body.Decode(l, ctx)(r, readerOptions)
			// SEARCH surfaces hits in the same GetItcListDone result view.
			writeBrowsePage(l, ctx, wp, s, body.Category(), body.CategorySub(), 0, 0, 0, browseFilterFromSearchItcList(*body))
		case ItcOperationRegisterWishEntry,
			ItcOperationSetZzim,
			ItcOperationDeleteZzim,
			ItcOperationViewWish,
			ItcOperationBuyWish,
			ItcOperationCancelWish,
			ItcOperationBuy,
			ItcOperationBuyZzim,
			ItcOperationPlaceBid,
			ItcOperationBuyAuctionImm:
			// Routed-but-unimplemented seam. Sibling arm tasks (buy/bid/wish) decode
			// the matching fieldsb.ItcOperation* body and emit the corresponding
			// COMMAND_TOPIC_MTS command + clientbound result.
			l.Infof("Character [%d] sent routed-but-unimplemented ITC_OPERATION [%s] (mode [%d]).", s.CharacterId(), key, p.Mode())
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

// writeBrowsePage queries atlas-mts REST for the listing page and writes the
// synchronous GetItcListDone result to the requesting session. Each MtsItem.itcSn
// is the listing's serial (from the REST itcSn). On a REST error an empty page is
// written so the client UI is not left hanging.
func writeBrowsePage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, f mtslisting.BrowseFilter) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(s.WorldId(), f)
	if err != nil {
		l.WithError(err).Errorf("Unable to browse MTS listings for character [%d]; writing empty page.", s.CharacterId())
		ms = nil
	}

	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtsItemFromListing(m))
	}

	body := fieldpkt.MtsOperationGetItcListDoneBody(uint32(len(items)), category, subCategory, page, sortType, sortColumn, items, 0)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS browse page to character [%d].", s.CharacterId())
	}
}
