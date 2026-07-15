package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v79 CASHSHOP_OPERATION (op 0x12F) family verification —
// CCashShop::OnCashItemResult @0x4720ed (GMS_v79_1_DEVM.exe, port 13340). The
// dispatcher does Decode1(mode) then switches over the cash-item result modes;
// the CASHSHOP_OPERATION siblings route to:
//
//	0x43 OnCashItemResLoadLockerDone   @0x472484 -> CashShopInventory
//	0x44 OnCashItemResLoadLockerFailed @0x47250a -> LoadInventoryFailure
//	move L<->S done / purchase / wishlist / capacity sub-handlers (sub_472xxx..)
//	  -> CashItemMovedToInventory / CashItemMovedToCashInventory /
//	     CashShopPurchaseSuccess / WishListLoad / WishListUpdate /
//	     InventoryCapacitySuccess / InventoryCapacityFailed
//
// The per-version dispatcher mode bytes are non-uniform (resolved from the
// Stage-C operations table); the BODY codecs gate only on MajorVersion()>12, so
// v79 and v83 (both >12) produce identical bytes. Each v79 encode is therefore
// byte-equal to the IDA-verified v83 encode (cross-version equality, the
// door/SpawnDoor discipline). The same literal mode byte is used for both
// contexts to isolate the body comparison.

// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashCashShopInventory version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashCashShopPurchaseSuccess version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v79 ida=0x4720ed
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v79 ida=0x4720ed
func TestCashShopOperationArmsV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	item := testItem()
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	wish := []uint32{5000000, 5000001, 5000002}
	type arm struct {
		name string
		v79  []byte
		v83  []byte
	}
	arms := []arm{
		{"MovedToCashInventory", NewCashItemMovedToCashInventory(0x50, item).Encode(l, v79)(nil), NewCashItemMovedToCashInventory(0x50, item).Encode(l, v83)(nil)},
		{"MovedToInventory", NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v79)(nil), NewCashItemMovedToInventory(0x51, 3, asset).Encode(l, v83)(nil)},
		{"ShopInventory", NewCashShopInventory(0x4D, []CashInventoryItem{item}, 4, 3).Encode(l, v79)(nil), NewCashShopInventory(0x4D, []CashInventoryItem{item}, 4, 3).Encode(l, v83)(nil)},
		{"PurchaseSuccess", NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v79)(nil), NewCashShopPurchaseSuccess(0x4F, item).Encode(l, v83)(nil)},
		{"CapacitySuccess", NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v79)(nil), NewInventoryCapacitySuccess(0x55, 2, 32).Encode(l, v83)(nil)},
		{"CapacityFailed", NewInventoryCapacityFailed(0x56, 7).Encode(l, v79)(nil), NewInventoryCapacityFailed(0x56, 7).Encode(l, v83)(nil)},
		{"LoadInventoryFailure", NewLoadInventoryFailure(0x44, 7).Encode(l, v79)(nil), NewLoadInventoryFailure(0x44, 7).Encode(l, v83)(nil)},
		{"WishListLoad", NewWishListLoad(0x4E, wish).Encode(l, v79)(nil), NewWishListLoad(0x4E, wish).Encode(l, v83)(nil)},
		{"WishListUpdate", NewWishListUpdate(0x59, wish).Encode(l, v79)(nil), NewWishListUpdate(0x59, wish).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v79, a.v83) {
			t.Errorf("%s v79 != v83\n v79: % x\n v83: % x", a.name, a.v79, a.v83)
		}
	}
}
