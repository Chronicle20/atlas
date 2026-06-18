package handler

import (
	"testing"

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
