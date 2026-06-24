package handler

import (
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
	// auction wire: slotPos 9 (a4), quantity 2 (v22), buyNow 5000, duration 48h
	item := packetmodel.NewAsset(false, 9, 1302000, time.Time{}).SetStackableInfo(2, 0, 0)
	p := fieldsb.NewItcOperationRegisterAuction(0x12, item, 9, 2, 1, 5000, 1, 0, 48)

	args := buildCreateListingFromRegisterAuction(p, testWorldId, testSellerId, testSellerAccountId, testSellerName)

	if args.SaleType != itcSaleTypeAuction {
		t.Errorf("saleType: want %s got %s", itcSaleTypeAuction, args.SaleType)
	}
	if args.TemplateId != 1302000 {
		t.Errorf("templateId: want 1302000 got %d", args.TemplateId)
	}
	if args.SlotPos != 9 {
		t.Errorf("slotPos: want 9 (a4) got %d", args.SlotPos)
	}
	if args.Quantity != 2 {
		t.Errorf("quantity: want 2 (v22) got %d", args.Quantity)
	}
	if args.BuyNowPrice == nil || *args.BuyNowPrice != 5000 {
		t.Errorf("buyNowPrice: want 5000, got %v", args.BuyNowPrice)
	}
	// the auction carries no separate list price; buy-now doubles as the list value
	if args.ListValue != 5000 {
		t.Errorf("listValue: want 5000 (buy-now), got %d", args.ListValue)
	}
	if args.DurationHours != 48 {
		t.Errorf("durationHours: want 48 got %d", args.DurationHours)
	}
}

func TestBuildCreateListingFromSaleCurrentItem(t *testing.T) {
	// sale-current: itemType 2, slotPos 7, item qty 3
	item := packetmodel.NewAsset(false, 7, 2000000, time.Time{}).SetStackableInfo(3, 0, 0)
	p := fieldsb.NewItcOperationSaleCurrentItem(3, 2, 7, item, 0)

	args := buildCreateListingFromSaleCurrentItem(p, testWorldId, testSellerId, testSellerAccountId, testSellerName)

	if args.SaleType != itcSaleTypeFixed {
		t.Errorf("saleType: want %s got %s", itcSaleTypeFixed, args.SaleType)
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
	// no price on the wire -> 0 (atlas-mts rejects against the price floor)
	if args.ListValue != 0 {
		t.Errorf("listValue: want 0 (no wire price), got %d", args.ListValue)
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

func TestBrowseFilterFromSearchItcList(t *testing.T) {
	// mode 6, category 2, sub 1, opt 0, searchName "Bob"
	p := fieldsb.NewItcOperationTabSearch(6, 2, 1, 0, "Bob")
	f := browseFilterFromSearchItcList(p)
	if f.SellerName != "Bob" {
		t.Errorf("sellerName: want Bob got %s", f.SellerName)
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
