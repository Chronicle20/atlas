package handler

import (
	"context"
	"testing"
	"time"

	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// itcOperationsTable mirrors the tenant template ITC_OPERATION
// options.operations table (template_gms_*_1.json). Config values arrive as
// float64 (JSON numbers), matching how isMessengerShopOperation reads them.
func itcOperationsTable() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"REGISTER_SALE":       float64(2),
			"SALE_CURRENT_ITEM":   float64(3),
			"REGISTER_WISH_ENTRY": float64(4),
			"GET_ITC_LIST":        float64(5),
			"SEARCH_ITC_LIST":     float64(6),
			"CANCEL_SALE":         float64(7),
			"TAKE_HOME":           float64(8),
			"SET_ZZIM":            float64(9),
			"DELETE_ZZIM":         float64(10),
			"VIEW_WISH":           float64(11),
			"BUY_WISH":            float64(12),
			"CANCEL_WISH":         float64(13),
			"BUY":                 float64(16),
			"BUY_ZZIM":            float64(17),
			"REGISTER_AUCTION":    float64(18),
			"PLACE_BID":           float64(19),
			"BUY_AUCTION_IMM":     float64(20),
		},
	}
}

func TestResolveItcOperationKey(t *testing.T) {
	resolve := resolveItcOperationKey(logrus.New())
	options := itcOperationsTable()

	cases := []struct {
		mode byte
		want string
	}{
		{2, ItcOperationRegisterSale},
		{3, ItcOperationSaleCurrentItem},
		{4, ItcOperationRegisterWishEntry},
		{5, ItcOperationGetItcList},
		{6, ItcOperationSearchItcList},
		{7, ItcOperationCancelSale},
		{8, ItcOperationTakeHome},
		{9, ItcOperationSetZzim},
		{10, ItcOperationDeleteZzim},
		{11, ItcOperationViewWish},
		{12, ItcOperationBuyWish},
		{13, ItcOperationCancelWish},
		{16, ItcOperationBuy},
		{17, ItcOperationBuyZzim},
		{18, ItcOperationRegisterAuction},
		{19, ItcOperationPlaceBid},
		{20, ItcOperationBuyAuctionImm},
	}

	for _, c := range cases {
		got, ok := resolve(options, c.mode)
		if !ok {
			t.Errorf("mode [%d]: expected resolution to [%s], got none", c.mode, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("mode [%d]: expected [%s], got [%s]", c.mode, c.want, got)
		}
	}
}

func TestResolveItcOperationKeyUnknownMode(t *testing.T) {
	resolve := resolveItcOperationKey(logrus.New())
	options := itcOperationsTable()

	// Mode 99 is not in the table (mirrors the ResolveCode default that crashes
	// the client when a mode is mis-mapped). It must resolve to nothing rather
	// than mis-route.
	if _, ok := resolve(options, 99); ok {
		t.Errorf("mode [99]: expected no resolution, got one")
	}
	// Mode 0/1/14/15 fall in gaps of the table — must not resolve.
	for _, m := range []byte{0, 1, 14, 15} {
		if _, ok := resolve(options, m); ok {
			t.Errorf("mode [%d]: expected no resolution (gap), got one", m)
		}
	}
}

func TestResolveItcOperationKeyMissingTable(t *testing.T) {
	resolve := resolveItcOperationKey(logrus.New())
	if _, ok := resolve(map[string]interface{}{}, 2); ok {
		t.Errorf("missing operations table: expected no resolution, got one")
	}
}

// --- decode->command mapping (task-102) ---------------------------------------

const (
	testWorldId         = 1
	testSellerId        = uint32(8001)
	testSellerAccountId = uint32(7001)
	testSellerName      = "Aria"
)

func TestBuildCreateListingFromRegisterSale(t *testing.T) {
	// register-sale wire: slotPos 5 (a4), quantity 7 (v22), price 1000 (a3)
	item := packetmodel.NewAsset(false, 5, 1302000, time.Time{}).SetStackableInfo(7, 0, 0)
	p := fieldsb.NewItcOperationRegisterSale(2, item, 5, 7, 1000, 1, 0)

	args := buildCreateListingFromRegisterSale(p, testWorldId, testSellerId, testSellerAccountId, testSellerName)

	if args.SaleType != itcSaleTypeFixed {
		t.Errorf("saleType: want %s got %s", itcSaleTypeFixed, args.SaleType)
	}
	// SourceInventoryType + AssetId are resolved from the seller's live inventory
	// in emitCreateListing (the blob has no slot); the build function carries the
	// item identity (templateId/cashId) instead.
	if args.TemplateId != 1302000 {
		t.Errorf("templateId: want 1302000 got %d", args.TemplateId)
	}
	if args.CashId != 0 {
		t.Errorf("cashId: want 0 got %d", args.CashId)
	}
	if args.SlotPos != 5 {
		t.Errorf("slotPos: want 5 (a4) got %d", args.SlotPos)
	}
	if args.Quantity != 7 {
		t.Errorf("quantity: want 7 (v22, not slotPos) got %d", args.Quantity)
	}
	if args.ListValue != 1000 {
		t.Errorf("listValue: want 1000 got %d", args.ListValue)
	}
	if args.BuyNowPrice != nil {
		t.Errorf("buyNowPrice: want nil for fixed sale, got %v", *args.BuyNowPrice)
	}
	if args.DurationHours != 0 {
		t.Errorf("durationHours: want 0 for fixed, got %d", args.DurationHours)
	}
	if args.SellerId != testSellerId || args.SellerAccountId != testSellerAccountId || args.SellerName != testSellerName {
		t.Errorf("seller identity not carried: id=%d acct=%d name=%s", args.SellerId, args.SellerAccountId, args.SellerName)
	}
	if args.WorldId != testWorldId {
		t.Errorf("worldId: want %d got %d", testWorldId, args.WorldId)
	}
}

func TestBuildCreateListingFromRegisterAuction(t *testing.T) {
	// auction wire: slotPos 9, quantity 2, startingBid 1000 (selector), buyNow 5000,
	// duration 48h (the Encode1 BYTE), bid increment 10 (the trailing Encode4). The
	// two prices are the starting bid and the buy-now price; the lower becomes the
	// listValue / first-bid floor, the higher the buy-now ceiling. Field order is
	// (…, durationHrs byte, flag byte, minIncrement uint32) — the task-102 label fix.
	item := packetmodel.NewAsset(false, 9, 1302000, time.Time{}).SetStackableInfo(2, 0, 0)
	p := fieldsb.NewItcOperationRegisterAuction(0x12, item, 9, 2, 1000, 5000, 48, 0, 10)

	args := buildCreateListingFromRegisterAuction(p, testWorldId, testSellerId, testSellerAccountId, testSellerName)

	if args.SaleType != itcSaleTypeAuction {
		t.Errorf("saleType: want %s got %s", itcSaleTypeAuction, args.SaleType)
	}
	if args.TemplateId != 1302000 {
		t.Errorf("templateId: want 1302000 got %d", args.TemplateId)
	}
	if args.SlotPos != 9 {
		t.Errorf("slotPos: want 9 got %d", args.SlotPos)
	}
	if args.Quantity != 2 {
		t.Errorf("quantity: want 2 got %d", args.Quantity)
	}
	if args.BuyNowPrice == nil || *args.BuyNowPrice != 5000 {
		t.Errorf("buyNowPrice: want 5000 (higher price), got %v", args.BuyNowPrice)
	}
	// the lower of the two wire prices is the starting bid / list value (first-bid floor)
	if args.ListValue != 1000 {
		t.Errorf("listValue: want 1000 (starting bid), got %d", args.ListValue)
	}
	// the Encode1 byte is the DURATION (not itemType); the trailing Encode4 is the
	// bid INCREMENT (not the duration).
	if args.DurationHours != 48 {
		t.Errorf("durationHours: want 48 (the byte field) got %d", args.DurationHours)
	}
	if args.MinIncrement != 10 {
		t.Errorf("minIncrement: want 10 (the trailing Encode4) got %d", args.MinIncrement)
	}
}

func TestBuildCreateListingFromRegisterAuctionPriceOrderIndependent(t *testing.T) {
	// The wire field order (selector then buyNowPrice) does not constrain which
	// price is the starting bid: the client guarantees buyNow > startingBid, so
	// min/max recovers (startingBid, buyNow) regardless of which slot holds which.
	item := packetmodel.NewAsset(false, 9, 1302000, time.Time{}).SetStackableInfo(2, 0, 0)
	// selector (5000) > buyNowPrice (1000): still listValue=1000, buyNow=5000.
	p := fieldsb.NewItcOperationRegisterAuction(0x12, item, 9, 2, 5000, 1000, 48, 0, 10)

	args := buildCreateListingFromRegisterAuction(p, testWorldId, testSellerId, testSellerAccountId, testSellerName)

	if args.ListValue != 1000 {
		t.Errorf("listValue: want 1000 (min), got %d", args.ListValue)
	}
	if args.BuyNowPrice == nil || *args.BuyNowPrice != 5000 {
		t.Errorf("buyNowPrice: want 5000 (max), got %v", args.BuyNowPrice)
	}
}

func TestBuildCreateListingFromSaleCurrentItem(t *testing.T) {
	// want-ad offer: itemType 2, slotPos 7, item qty 3, target want-ad serial 4244.
	// The offered item is escrowed as an `offer` listing linked to want-ad 4244
	// (owner 991) at the resolved want-ad asking price (3500).
	item := packetmodel.NewAsset(false, 7, 2000000, time.Time{}).SetStackableInfo(3, 0, 0)
	p := fieldsb.NewItcOperationSaleCurrentItem(3, 2, 7, item, 4244)

	const wantOwnerId uint32 = 991
	// want-ad wants 5 units; the offerer has a stack of 3 -> escrow all 3 (the
	// smaller of wanted vs stack).
	args := buildCreateListingFromSaleCurrentItem(p, testWorldId, testSellerId, testSellerAccountId, testSellerName, wantOwnerId, 3500, 5)

	if args.SaleType != itcSaleTypeOffer {
		t.Errorf("saleType: want %s got %s", itcSaleTypeOffer, args.SaleType)
	}
	if args.TemplateId != 2000000 {
		t.Errorf("templateId: want 2000000 got %d", args.TemplateId)
	}
	if args.CashId != 0 {
		t.Errorf("cashId: want 0 got %d", args.CashId)
	}
	if args.Quantity != 3 {
		t.Errorf("quantity: want 3 got %d", args.Quantity)
	}
	// listValue is the resolved want-ad asking price the offer fulfills
	if args.ListValue != 3500 {
		t.Errorf("listValue: want 3500 (want-ad price), got %d", args.ListValue)
	}
	// the offer must be linked back to the target want-ad (serial + poster)
	if args.OfferWishSerial != 4244 {
		t.Errorf("offerWishSerial: want 4244 got %d", args.OfferWishSerial)
	}
	if args.OfferWishOwnerId != wantOwnerId {
		t.Errorf("offerWishOwnerId: want %d got %d", wantOwnerId, args.OfferWishOwnerId)
	}

	// Clamp: a want-ad for 2 with the offerer's stack of 3 escrows only 2 units.
	clamped := buildCreateListingFromSaleCurrentItem(p, testWorldId, testSellerId, testSellerAccountId, testSellerName, wantOwnerId, 3500, 2)
	if clamped.Quantity != 2 {
		t.Errorf("clamped quantity: want 2 (wanted<stack), got %d", clamped.Quantity)
	}
}

func TestBrowseFilterFromGetItcList(t *testing.T) {
	// mode 5, category 3, sub 1, page 2, sortType 1, sortColumn 1, opt 1, ""
	p := fieldsb.NewItcOperationChangedPage(5, 3, 1, 2, 1, 1, 1, "")
	f := browseFilterFromGetItcList(p)
	if f.Page != 2 {
		t.Errorf("page: want 2 got %d", f.Page)
	}
	if f.SellerName != "" {
		t.Errorf("sellerName: want empty for browse, got %s", f.SellerName)
	}
}

// TestBrowseFilterFromSearchItcList_EmptyName asserts that an empty search term
// browses the view unfiltered (hasResults=true, no TemplateIds, no SellerName) and
// applies the view (category/sub) filters. The empty-name path does not hit
// atlas-data, so this is deterministic without a server.
func TestBrowseFilterFromSearchItcList_EmptyName(t *testing.T) {
	// mode 6, category 3, sub 2, opt 0, searchName ""
	p := fieldsb.NewItcOperationTabSearch(6, 3, 2, 0, "")
	f, hasResults := browseFilterFromSearchItcList(logrus.New(), context.Background(), p)
	if !hasResults {
		t.Errorf("empty search name should browse the view (hasResults=true), got false")
	}
	if f.SellerName != "" {
		t.Errorf("sellerName: must be unused for item-name search, got %q", f.SellerName)
	}
	if len(f.TemplateIds) != 0 {
		t.Errorf("templateIds: want none for empty search, got %v", f.TemplateIds)
	}
	if f.Category != "3" {
		t.Errorf("category: want 3, got %q", f.Category)
	}
	if f.SubCategory != "2" {
		t.Errorf("subCategory: want 2, got %q", f.SubCategory)
	}
}

// TestBrowseFilterFromSearchItcList_ResolveErrorShowsNoResults asserts that when
// the search term cannot be resolved (here, atlas-data is unreachable because no
// DATA_SERVICE_URL is configured in the test env), the helper reports hasResults
// =false so the caller writes an empty page rather than leaking the whole
// marketplace via an empty filter.
func TestBrowseFilterFromSearchItcList_ResolveErrorShowsNoResults(t *testing.T) {
	p := fieldsb.NewItcOperationTabSearch(6, 1, 0, 0, "Sword")
	_, hasResults := browseFilterFromSearchItcList(logrus.New(), context.Background(), p)
	if hasResults {
		t.Errorf("a failed/empty resolve must report hasResults=false (no results), got true")
	}
}

func TestMtsItemFromListing_CarriesSerialAsItcSn(t *testing.T) {
	m := mtslisting.RestModel{
		Id:           "abc",
		WorldId:      1,
		ItcSn:        4242,
		SellerName:   "Aria",
		TemplateId:   1302000,
		Quantity:     1,
		ListValue:    1000,
		BuyNowPrice:  5000,
		CurrentBid:   1200,
		MinIncrement: 100,
	}
	model, err := mtslisting.Extract(m)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	it := mtsItemFromListing(model)
	if it.ItcSn() != 4242 {
		t.Errorf("itcSn: want 4242 (the listing serial), got %d", it.ItcSn())
	}
	if it.Price() != 1000 {
		t.Errorf("price: want 1000 (list value), got %d", it.Price())
	}
	if it.Item().TemplateId() != 1302000 {
		t.Errorf("item template: want 1302000, got %d", it.Item().TemplateId())
	}
}

// TestMtsItemFromWish_CarriesSerial is the H5 round-trip guard: a wish entry
// renders as an ITCITEM whose nITCSN is the wish entry's OWN per-(tenant, world)
// serial (not 0). The client echoes that nITCSN back verbatim on CANCEL_WISH
// (IDA: CITC::OnCancelWish, v83 0x59fb07, Encode4 of the item's nITCSN), so the
// channel can resolve the cancel to the wish. The old behavior wrote 0 here,
// which meant the client always sent 0 and the cancel never resolved.
func TestMtsItemFromWish_CarriesSerial(t *testing.T) {
	wm, err := mtswish.Extract(mtswish.RestModel{Id: "w1", WorldId: 0, Serial: 7777, CharacterId: 9001, ItemId: 1302000})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	it := mtsItemFromWish(wm)
	if it.Item().TemplateId() != 1302000 {
		t.Errorf("item template: want 1302000, got %d", it.Item().TemplateId())
	}
	if it.ItcSn() != 7777 {
		t.Errorf("itcSn: want 7777 (the wish entry's serial), got %d", it.ItcSn())
	}
	if it.Price() != 0 {
		t.Errorf("price: want 0 (a wish has no price), got %d", it.Price())
	}
}

// TestApplyItcViewFilters locks the GET_ITC_LIST browse->filter mapping: the
// category (marketplace section / top tab) maps to the listing `category` filter
// and the item sub-tab (categorySub, 1-4 = inventory type; 0 = all = unfiltered)
// to the `subCategory` filter — a straight equality filter, no per-view casing.
func TestApplyItcViewFilters(t *testing.T) {
	cases := []struct {
		name            string
		category        uint32
		categorySub     uint32
		wantCategory    string
		wantSubCategory string
	}{
		{"for-sale all", 1, 0, "1", ""},
		{"for-sale use sub-tab", 1, 2, "1", "2"},
		{"auction all", 3, 0, "3", ""},
		{"auction equip sub-tab", 3, 1, "3", "1"},
		{"wanted section", 2, 0, "2", ""},
		{"my-page etc sub-tab", 4, 4, "4", "4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var f mtslisting.BrowseFilter
			applyItcViewFilters(&f, c.category, c.categorySub)
			if f.Category != c.wantCategory {
				t.Errorf("category: want %q got %q", c.wantCategory, f.Category)
			}
			if f.SubCategory != c.wantSubCategory {
				t.Errorf("subCategory: want %q got %q", c.wantSubCategory, f.SubCategory)
			}
		})
	}
}
