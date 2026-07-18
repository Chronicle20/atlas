package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 CASHSHOP_OPERATION (op 255) family verification —
// CCashShop::OnCashItemResult @0x461379 (GMS_v61.1_U_DEVM.exe, port 13338).
//
// The v61 CCashShop dispatch is two-level like v72: CCashShop::OnPacket routes
// the outer opcode 255 to the inner mode dispatcher OnCashItemResult @0x461379,
// which Decode1(mode) then switches the cash-item result modes. The clientbound
// opcode is Δ-36 vs v72 (255 vs 291), and the per-mode dispatch bytes shift with
// the version, but the dispatch (mode) byte is resolved from the tenant
// operations table — NOT the codec. The codec BODY gates only on
// MajorVersion()>12 (and a lone >=95 field in shop_inventory.go), so v61 (>12,
// <95) produces bodies byte-identical to the IDA-verified v83 encode. Each arm
// is therefore compared to its v83 encode (cross-version equality, the same
// discipline the v72/v79 fixtures use). A single literal mode byte isolates the
// body comparison.

// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashCashShopInventory version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashCashShopPurchaseSuccess version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v61 ida=0x461379
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v61 ida=0x461379
func TestCashShopOperationArmsV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	item := testItem()
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	wish := []uint32{5000000, 5000001, 5000002}
	type arm struct {
		name string
		v61  []byte
		v83  []byte
	}
	arms := []arm{
		{"MovedToCashInventory", NewCashItemMovedToCashInventory(0x50, item).Encode(l, v61)(nil), NewCashItemMovedToCashInventory(0x50, item).Encode(l, v83)(nil)},
		{"MovedToInventory", NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v61)(nil), NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v83)(nil)},
		{"ShopInventory", NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v61)(nil), NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v83)(nil)},
		{"PurchaseSuccess", NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v61)(nil), NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v83)(nil)},
		{"CapacitySuccess", NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v61)(nil), NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v83)(nil)},
		{"CapacityFailed", NewInventoryCapacityFailed(0x56, 7).Encode(l, v61)(nil), NewInventoryCapacityFailed(0x56, 7).Encode(l, v83)(nil)},
		{"LoadInventoryFailure", NewLoadInventoryFailure(0x38, 7).Encode(l, v61)(nil), NewLoadInventoryFailure(0x38, 7).Encode(l, v83)(nil)},
		{"WishListLoad", NewWishListLoad(0x4E, wish).Encode(l, v61)(nil), NewWishListLoad(0x4E, wish).Encode(l, v83)(nil)},
		{"WishListUpdate", NewWishListUpdate(0x59, wish).Encode(l, v61)(nil), NewWishListUpdate(0x59, wish).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v61, a.v83) {
			t.Errorf("%s v61 != v83\n v61: % x\n v83: % x", a.name, a.v61, a.v83)
		}
	}
}
