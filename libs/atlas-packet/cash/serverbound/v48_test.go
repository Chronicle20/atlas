package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(160) call site in GMS_v48_1_DEVM.exe @port 13337. The v48
// CCashShop cash-operation family sends COutPacket(160) + Encode1(sub-op mode)
// + body; the dispatcher strips the op byte + mode, so the fixtures pin the
// BODY only. v48 is the OLDEST anchor and is uniformly leaner than v61: the
// currency int present in v61 is absent below v61 (buyOmitsCurrency, GMS < 61).

// TestShopOperationBuyBytesV48 pins the v48 buy body. IDA v48 CCashShop::OnBuy
// @0x44b0cf, send @0x44b38a: COutPacket(160) Encode1(2)=mode, Encode1(v29==2)=
// isPoints, Encode4(a2)=serialNumber. No currency int (added at v61), no
// trailing IsZeroGoods (added at v72). Body = isPoints(1)+serialNumber(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v48 ida=0x44b0cf
func TestShopOperationBuyBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 2, zero: 3, oneADay: 1, eventSN: 4}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "01"+"02000000" {
		t.Errorf("v48 buy bytes: got %s, want 0102000000 (isPoints+serial, no currency)", got)
	}
}

// TestShopOperationBuyCoupleBytesV48 pins the v48 couple-buy body. IDA v48
// CCashShop::OnBuyCouple @0x44b4c1, send @0x44b79b: COutPacket(160) Encode1(0x1A)
// =mode, Encode1(v37==2)=isPoints, Encode4(a2)=serialNumber. No currency int
// (dropped below v61). Body = isPoints(1)+serialNumber(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyCouple version=gms_v48 ida=0x44b4c1
func TestShopOperationBuyCoupleBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyCouple{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "01"+"08070605" {
		t.Errorf("v48 couple bytes: got %s, want 0108070605", got)
	}
}

// TestShopOperationBuyPackageBytesV48 pins the v48 package-buy body. IDA v48
// CCashShop::OnBuyPackage @0x44b837, send @0x44b9e1: COutPacket(160) Encode1(0x1C)
// =mode, Encode4(a2)=serialNumber. Body = serialNumber(4) only == v61 legacy.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v48 ida=0x44b837
func TestShopOperationBuyPackageBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyPackage{pointType: true, option: 1, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "08070605" {
		t.Errorf("v48 package bytes: got %s, want 08070605", got)
	}
}

// TestShopOperationGiftBytesV48 pins the v48 gift body. IDA v48 CCashShop::OnGift
// @0x44ba5d, send @0x44bd4e: COutPacket(160) Encode1(3)=mode, Encode4(ask_SPW)=
// birthday int, Encode4(a2)=serialNumber, EncodeStr(name), EncodeStr(message).
// Body = birthday(4)+serialNumber(4)+name+message == the GMS<87 gift path. (The
// modeled fname SendGiftsPacket is not present in v48; the gift is CCashShop::
// OnGift @0x44ba5d.)
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v48 ida=0x44ba5d
func TestShopOperationGiftBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0x05060708, oneADay: 1, name: "", message: ""}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v48 gift bytes: got %s", got)
	}
}

// TestShopOperationSetWishlistBytesV48 pins the v48 set-wishlist body. IDA v48
// CCashShop::OnSetWish @0x44ce9b, send @0x44cf78: COutPacket(160) Encode1(4)=mode
// then a fixed 10-iter Encode4 loop over the wishlist serials. Body = 10×Encode4,
// no count prefix == v61.
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v48 ida=0x44ce9b
func TestShopOperationSetWishlistBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationSetWishlist{serialNumbers: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	want := "01000000" + "02000000" + "03000000" + "04000000" + "05000000" +
		"06000000" + "07000000" + "08000000" + "09000000" + "0a000000"
	if got != want {
		t.Errorf("v48 wishlist bytes: got %s, want %s", got, want)
	}
}

// TestShopOperationBuyNormalBytesV48 pins the v48 OnBuyNormal body. The v48
// IDB-labeled CCashShop::OnBuyNormal @0x44cbb2 is gift-shaped (send @0x44cdaf):
// COutPacket(160) Encode1(0x1F)=mode, Encode4(ask_SPW)=spw int, Encode4(a2)=
// serialNumber, EncodeStr(name), EncodeStr(message). Body = spw(4)+serial(4)+
// name+message (v48-only; v83+ OnBuyNormal is a bare serial(4)).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=gms_v48 ida=0x44cbb2
func TestShopOperationBuyNormalBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyNormal{spw: 0x01020304, serialNumber: 0x05060708, name: "", message: ""}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v48 buyNormal bytes: got %s", got)
	}
}

// TestShopOperationBuyFriendshipBytesV48 pins the v48 friendship-ring body. IDA
// v48 CCashShop::OnBuyFriendship @0x44c879, send @0x44cadb: COutPacket(160)
// Encode1((serial/1000==9110)+5)=mode, Encode1(v37==2)=pointType, Encode1(1)=
// constant flag byte, Encode4(a2)=serialNumber. Body = pointType(1)+flag(1)+
// serialNumber(4) (v48-only friendship-ring/equip-slot buy; the currency int
// present in v61 is a flag byte here).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v48 ida=0x44c879
func TestShopOperationBuyFriendshipBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyFriendship{isPoints: true, flag: 1, serialNumber: 0x05060708, currency: 0x01020304, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "01"+"01"+"08070605" {
		t.Errorf("v48 friendship bytes: got %s, want 010108070605", got)
	}
}

// TestItemUseMegaphoneBytesV48 pins the v48 basic-Megaphone USE_CASH_ITEM
// sub-body. IDA v48 CWvsContext::SendConsumeCashItemUseRequest @0x70e495:
// the outer header (COutPacket ctor opcode 0x3E, Encode2(slot), Encode4
// (itemId) @0x70e4f9-0x70e517) carries NO update_time. The cash-slot-type
// switch (@0x70e51f sub_47742E, jumptable @0x70e53c) maps BOTH type 12
// (Megaphone) and type 13 (SuperMegaphone) to the shared case label
// loc_70E543. Inside, `cmp type,0Dh; jnz loc_70E5E4` — type 12 (this test)
// takes the jnz branch to the message-input dialog; after trim/length
// validation the send tail @loc_70E800-0x70E830 does:
//
//	EncodeStr(message)              @0x70e814
//	cmp type,0Dh; jnz skip whisper  @0x70e819 (type 12 SKIPS the whisper byte)
//
// CORRECTION (legacy TV/item/triple gap-fill pass): the case body then falls
// through (via `cmp eax,ebx; jz loc_70E845` on an unrelated "attached
// commodity" pointer, normally nil) into the SHARED jumptable-case-34 tail
// @loc_711D60: rate-check (sub_4A2518(0,500)) then, on success,
// `call SetExclRequestSent (GetTickCount-style read of g_CWvsApp+0x18);
// push eax; call Encode4 @0x711d9f; call sub_711EC9 (SendPacket)`. update_time
// IS present (trailing uint32), contradicting the earlier "shared cleanup, no
// Encode4" reading which stopped short of this shared tail.
// Wire (v48): message(str) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v48 ida=0x70e495
func TestItemUseMegaphoneBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!")) + "39300000"
	if got != want {
		t.Errorf("v48 item use megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseSuperMegaphoneBytesV48 pins the v48 SuperMegaphone USE_CASH_ITEM
// sub-body. Same case label loc_70E543 as basic Megaphone (jumptable cases
// 12,13); type 13 (this test) FALLS THROUGH the `cmp type,0Dh; jnz` at
// @0x70e546 into a distinct (larger) dialog allocation, then the SAME
// message tail @0x70e800-0x70e830:
//
//	EncodeStr(message)          @0x70e814
//	cmp type,0Dh; jnz skip      @0x70e819 (type 13 MATCHES -> whisper emitted)
//	Encode1(whisper)            @0x70e825
//
// CORRECTION (legacy TV/item/triple gap-fill pass): same shared jumptable
// case-34 tail as basic Megaphone (loc_711D60 @0x711d96 Encode4) — see that
// test's comment. update_time IS present (trailing uint32).
// Wire (v48): message(str) + whisper(bool) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v48 ida=0x70e495
func TestItemUseSuperMegaphoneBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01" + "31d40000"
	if got != want {
		t.Errorf("v48 item use super megaphone bytes: got %s, want %s", got, want)
	}
}
