package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 CASHSHOP_OPERATION (op 291) family verification —
// CCashShop::OnCashItemResult @0x470e13 (GMS_v72.1_U_DEVM.exe, port 13339).
//
// The v72 CCashShop dispatch is two-level: CCashShop::OnPacket @0x470b2d (the
// address the registry records) switches the outer opcode and routes op 291 to
// CCashShop::OnCashItemResult @0x470e13, which then Decode1(mode) and switches
// over the cash-item result modes. (The registry ida.address points at the
// outer OnPacket; the mode dispatcher body is the inner @0x470e13 — the export
// carries the correct inner address.)
//
// The per-mode dispatch bytes are −12 (0x0C) shifted vs v79 (deep CCashShop
// region): OnCashItemResLoadLockerDone is v72 case 0x37 vs v79 0x43, and
// OnCashItemResLoadLockerFailed is v72 0x38 vs v79 0x44 — decompile-confirmed on
// both IDBs. The dispatch (mode) byte is resolved from the tenant operations
// table, not the codec; the codec BODY gates only on MajorVersion()>12 (and a
// lone >=95 field in shop_inventory.go), so v72 and v83 (both >12 and <95)
// produce byte-identical bodies. Each v72 encode is therefore byte-equal to the
// IDA-verified v83 encode (cross-version equality, the same discipline the v79
// fixture uses). A single literal mode byte is used for both contexts to
// isolate the body comparison.

// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashCashShopInventory version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashCashShopPurchaseSuccess version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v72 ida=0x470e13
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v72 ida=0x470e13
func TestCashShopOperationArmsV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	item := testItem()
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	wish := []uint32{5000000, 5000001, 5000002}
	type arm struct {
		name string
		v72  []byte
		v83  []byte
	}
	arms := []arm{
		{"MovedToCashInventory", NewCashItemMovedToCashInventory(0x50, item).Encode(l, v72)(nil), NewCashItemMovedToCashInventory(0x50, item).Encode(l, v83)(nil)},
		{"MovedToInventory", NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v72)(nil), NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v83)(nil)},
		{"ShopInventory", NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v72)(nil), NewCashShopInventory(0x37, []CashInventoryItem{item}, 4, 3).Encode(l, v83)(nil)},
		{"PurchaseSuccess", NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v72)(nil), NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v83)(nil)},
		{"CapacitySuccess", NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v72)(nil), NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v83)(nil)},
		{"CapacityFailed", NewInventoryCapacityFailed(0x56, 7).Encode(l, v72)(nil), NewInventoryCapacityFailed(0x56, 7).Encode(l, v83)(nil)},
		{"LoadInventoryFailure", NewLoadInventoryFailure(0x38, 7).Encode(l, v72)(nil), NewLoadInventoryFailure(0x38, 7).Encode(l, v83)(nil)},
		{"WishListLoad", NewWishListLoad(0x4E, wish).Encode(l, v72)(nil), NewWishListLoad(0x4E, wish).Encode(l, v83)(nil)},
		{"WishListUpdate", NewWishListUpdate(0x59, wish).Encode(l, v72)(nil), NewWishListUpdate(0x59, wish).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v72, a.v83) {
			t.Errorf("%s v72 != v83\n v72: % x\n v83: % x", a.name, a.v72, a.v83)
		}
	}
}
