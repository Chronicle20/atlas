package serverbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v61 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(196) call site in GMS_v61.1_U_DEVM.exe @port 13338. The CCashShop
// cash-purchase family sends COutPacket(196) + Encode1(sub-op mode) + body; the
// dispatcher strips the op byte + mode, so the fixtures pin the BODY only. All
// bodies match the verified v72 anchor EXCEPT ShopOperationBuy, whose trailing
// IsZeroGoods int is absent in v61 (version-gated — see buyOmitsTrailingZero).

// TestCheckWalletBytesV61 pins the v61 CHECK_CASH wire: EMPTY. sub_45C33E
// @0x45c33e builds COutPacket(195) @0x45c362 and SendPacket()s with zero Encode*.
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v61 ida=0x45c33e
func TestCheckWalletBytesV61(t *testing.T) {
	got := CheckWallet{}.Encode(nil, pt.CreateContext("GMS", 61, 1))(nil)
	if !bytes.Equal(got, []byte{}) {
		t.Errorf("v61 checkWallet bytes: got % x, want empty", got)
	}
}

// TestShopOperationBuyBytesV61 pins the v61 buy body. IDA v61 CCashShop::OnBuy
// @0x457ea4, send @0x4581eb: COutPacket(196) Encode1(3)=mode, Encode1(isPoints),
// Encode4(currency), Encode4(serialNumber). Unlike v72 (which appends a trailing
// IsZeroGoods Encode4), v61 STOPS after serialNumber — the trailing zero is
// absent (buyOmitsTrailingZero, GMS<72). Body = isPoints(1)+currency(4)+serial(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v61 ida=0x457ea4
func TestShopOperationBuyBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 2, zero: 3, oneADay: 1, eventSN: 4}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01"+"01000000"+"02000000" {
		t.Errorf("v61 bytes: got %s, want 010100000002000000 (no trailing zero)", got)
	}
}

// TestShopOperationBuyCoupleBytesV61 pins the v61 legacy couple-buy body. IDA v61
// CCashShop::OnBuyCouple @0x45832d: COutPacket(196) Encode1(mode) then Encode1
// (isPoints), Encode4(currency), Encode4(serialNumber). Body == v72 legacy
// (legacyGMS <83): isPoints(1)+currency(4)+serial(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyCouple version=gms_v61 ida=0x45832d
func TestShopOperationBuyCoupleBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyCouple{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01"+"04030201"+"08070605" {
		t.Errorf("v61 bytes: got %s, want 010403020108070605", got)
	}
}

// TestShopOperationBuyFriendshipBytesV61 pins the v61 legacy friendship-buy body.
// IDA v61 CCashShop::OnBuyFriendship @0x4574b0: mode + isPoints(1)+currency(4)+
// serial(4) == v72 legacy.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v61 ida=0x4574b0
func TestShopOperationBuyFriendshipBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyFriendship{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01"+"04030201"+"08070605" {
		t.Errorf("v61 bytes: got %s, want 010403020108070605", got)
	}
}

// TestShopOperationBuyPackageBytesV61 pins the v61 package-buy body. IDA v61
// CCashShop::OnBuyPackage @0x4586ce: mode + Encode4(serialNumber). Body =
// serialNumber(4) only == v72.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v61 ida=0x4586ce
func TestShopOperationBuyPackageBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyPackage{pointType: true, option: 1, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "08070605" {
		t.Errorf("v61 bytes: got %s, want 08070605", got)
	}
}

// TestShopOperationEnableEquipSlotBytesV61 pins the v61 enable-equip-slot body.
// The v61 send is the IDB-mislabeled CCashShop::OnIncCharacterSlotCount @0x459928
// (send @0x459bb2): COutPacket(196) Encode1((v/1000==9110)+6=mode 6|7) then
// Encode1(pointType), Encode4(currency), Encode1(flag=1), Encode4(serial). Body
// after the mode byte = pointType(1)+currency(4)+flag(1)+serial(4) == v72 legacy.
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v61 ida=0x459928
func TestShopOperationEnableEquipSlotBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationEnableEquipSlot{pointType: true, currency: 0x01020304, flag: 1, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01"+"04030201"+"01"+"08070605" {
		t.Errorf("v61 bytes: got %s, want 01040302010108070605", got)
	}
}

// TestShopOperationGiftBytesV61 pins the v61 regular-gift body. IDA v61
// SendGiftsPacket sub_45C607 @0x45c607, send @0x45c981: COutPacket(196) Encode1
// (mode 4 = regular gift) Encode4(birthday) [no option for mode 4] Encode4(serial)
// EncodeStr(name) EncodeStr(message). Body == v72: birthday(4)+serial(4)+name+
// message. (The couple/friendship gift arms use other modes with an option int;
// the regular-gift arm matches the Atlas ShopOperationGift <87 encode.)
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v61 ida=0x45c607
func TestShopOperationGiftBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0x05060708, oneADay: 1, name: "", message: ""}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v61 bytes: got %s", got)
	}
}

// TestShopOperationSetWishlistBytesV61 pins the v61 set-wishlist body. IDA v61
// CCashShop::OnSetWish @0x45a345: COutPacket(196) Encode1(mode) then a fixed
// 10-iter Encode4 loop over the wishlist serials. Body == v72: 10×Encode4, no
// count prefix.
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v61 ida=0x45a345
func TestShopOperationSetWishlistBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationSetWishlist{serialNumbers: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	want := "01000000" + "02000000" + "03000000" + "04000000" + "05000000" +
		"06000000" + "07000000" + "08000000" + "09000000" + "0a000000"
	if got != want {
		t.Errorf("v61 bytes: got %s, want %s", got, want)
	}
}
