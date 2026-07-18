package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 CASHSHOP_CASH_ITEM_RESULT (op 256) family verification —
// CCashShop::OnCashItemResult @0x4537a8 (GMS_v48_1_DEVM.exe, port 13337).
//
// The v48 CCashShop dispatch is two-level like v61/v72: CCashShop::OnPacket
// routes the outer opcode to the inner mode dispatcher OnCashItemResult
// @0x4537a8, which Decode1(mode) then switches the cash-item result modes
// (~40 client cases spanning 0x29-0x6B; Atlas models the 9 supported arms). In
// v48 the result dispatcher lives at outer opcode 256 (CASHSHOP_CASH_ITEM_RESULT)
// rather than the CASHSHOP_OPERATION slot used from v61 onward — the whole cash
// clientbound opcode block is Δ≈-36 vs v61. The per-mode dispatch bytes are
// resolved from the tenant operations table, NOT the codec. The codec BODY
// gates only on MajorVersion()>12 (and a lone >=95 field in shop_inventory.go),
// so v48 (>12, <95) produces bodies byte-identical to the IDA-verified v83
// encode — the same cross-version-equality discipline the v61/v72/v79 fixtures
// use. A single literal mode byte isolates the body comparison.
//
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashCashShopInventory version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashCashShopPurchaseSuccess version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v48 ida=0x4537a8
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v48 ida=0x4537a8
func TestCashShopOperationArmsV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v48 := pt.CreateContext("GMS", 48, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	item := testItem()
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	wish := []uint32{5000000, 5000001, 5000002}
	type arm struct {
		name string
		v48  []byte
		v83  []byte
	}
	arms := []arm{
		{"MovedToCashInventory", NewCashItemMovedToCashInventory(0x50, item).Encode(l, v48)(nil), NewCashItemMovedToCashInventory(0x50, item).Encode(l, v83)(nil)},
		{"MovedToInventory", NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v48)(nil), NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v83)(nil)},
		{"ShopInventory", NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v48)(nil), NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v83)(nil)},
		{"PurchaseSuccess", NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v48)(nil), NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v83)(nil)},
		{"CapacitySuccess", NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v48)(nil), NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v83)(nil)},
		{"CapacityFailed", NewInventoryCapacityFailed(0x56, 7).Encode(l, v48)(nil), NewInventoryCapacityFailed(0x56, 7).Encode(l, v83)(nil)},
		{"LoadInventoryFailure", NewLoadInventoryFailure(0x38, 7).Encode(l, v48)(nil), NewLoadInventoryFailure(0x38, 7).Encode(l, v83)(nil)},
		{"WishListLoad", NewWishListLoad(0x4E, wish).Encode(l, v48)(nil), NewWishListLoad(0x4E, wish).Encode(l, v83)(nil)},
		{"WishListUpdate", NewWishListUpdate(0x59, wish).Encode(l, v48)(nil), NewWishListUpdate(0x59, wish).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v48, a.v83) {
			t.Errorf("%s v48 != v83\n v48: % x\n v83: % x", a.name, a.v48, a.v83)
		}
	}
}
