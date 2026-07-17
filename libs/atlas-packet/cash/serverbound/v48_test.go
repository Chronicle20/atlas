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
