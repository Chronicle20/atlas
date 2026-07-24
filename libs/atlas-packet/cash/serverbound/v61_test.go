package serverbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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

// TestItemUseMegaphoneBytesV61 pins the v61 basic-Megaphone USE_CASH_ITEM
// sub-body. IDA v61 CWvsContext::SendConsumeCashItemUseRequest @0x832a5d:
// outer header (COutPacket opcode 0x49, Encode2(slot), Encode4(itemId)
// @0x832abe-0x832ae4) carries NO update_time. Jumptable case label
// loc_832B08 covers types 12,13 (Megaphone/SuperMegaphone). `cmp Str2,0Dh;
// jnz loc_832BA9` — type 12 (this test) takes the jnz path. Message tail
// @0x832ddc-0x832e06:
//
//	EncodeStr(message)          @0x832df0
//	cmp Str2,0Dh; jnz skip      @0x832df9 (type 12 SKIPS whisper)
//
// CORRECTION (legacy TV/item/triple gap-fill pass): the case body falls
// through into the SAME shared rate-check-and-send tail architecture
// IDA-confirmed on v48/v72/v79 (v61 mirrors v72/v79's case-33-style tail:
// rate-check -> on success `call SetExclRequestSent; Encode4; SendPacket`).
// update_time IS present (trailing uint32) — the earlier "shared cleanup, no
// Encode4" reading stopped short of this shared tail.
// Wire (v61): message(str) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v61 ida=0x832a5d
func TestItemUseMegaphoneBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!")) + "39300000"
	if got != want {
		t.Errorf("v61 item use megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseSuperMegaphoneBytesV61 pins the v61 SuperMegaphone USE_CASH_ITEM
// sub-body. Same case label loc_832B08 (types 12,13); type 13 (this test)
// falls through the `cmp Str2,0Dh; jnz` at @0x832b0b into the larger dialog
// path, then the SAME message tail:
//
//	EncodeStr(message)          @0x832df0
//	cmp Str2,0Dh; jnz skip      @0x832df9 (type 13 MATCHES -> whisper emitted)
//	Encode1(whisper)            @0x832e01
//
// CORRECTION (legacy TV/item/triple gap-fill pass): same shared
// rate-check-and-send tail as basic Megaphone — see that test's comment.
// update_time IS present (trailing uint32).
// Wire (v61): message(str) + whisper(bool) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v61 ida=0x832a5d
func TestItemUseSuperMegaphoneBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01" + "31d40000"
	if got != want {
		t.Errorf("v61 item use super megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseItemMegaphoneBytesV61 pins the v61 Item Megaphone (5076xxx)
// serverbound wire (v61 is the earliest version with this item — v48 has
// none, per atlas-data item-strings). IDA v61: SendConsumeCashItemUseRequest's
// jumptable case 14 @0x832e37 only constructs/shows a dedicated dialog
// (0x5A0-byte ZAllocEx alloc + sub_55CAB8 ctor — byte-identical layout
// pattern to v72/v79: ctor 0x114, OnCreate sub_55CCFB 0x63c, OnCommand
// sub_55D430 0x2d, validate+send sub_55DC01). Full decompile of sub_55DC01
// (0x55dcd1-0x55dd77):
//
//	COutPacket ctor(73=0x49)                              @0x55dcd1
//	Encode2(*(WORD*)(this+120))         = slot             @0x55dce7
//	Encode4(*(DWORD*)(this+124))        = itemId           @0x55dcf2
//	EncodeStr(CCtrlEdit::GetText())     = message          @0x55dd13
//	Encode1(*(DWORD*)(*(DWORD*)(this+1396)+72)) = whisper  @0x55dd24
//	Encode1(*(DWORD*)(this+140)!=0)     = hasItem          @0x55dd36
//	  if hasItem: Encode4(this+128)=invType, Encode4(this+132)=slotPos
//	                                                         @0x55dd4c/0x55dd5a
//	call SetExclRequestSent(); push eax; Encode4(eax) = updateTime @0x55dd68
//	SendPacket()                                            @0x55dd77
//
// Wire (v61): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32)+slot(int32)] + updateTime(uint32 trailing) — matches
// ItemUseItemMegaphone.Encode(updateTimeFirst=false) exactly.
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=gms_v61 ida=0x55dc01
func TestItemUseItemMegaphoneBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewItemUseItemMegaphone(false)
	input.message = "Item hello!"
	input.whisper = true
	input.hasItem = true
	input.invType = 2
	input.slot = 5
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0b00" + hex.EncodeToString([]byte("Item hello!")) + "01" + "01" + "02000000" + "05000000" + "39300000"
	if got != want {
		t.Errorf("v61 item use item megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseMapleTVBytesV61 pins the v61 Maple TV (5075xxx) serverbound
// wire for tvType 0. IDA v61: jumptable case 45 @0x834d1c (first of SIX
// consecutive TV cases 45-50 — shifted -1 vs v72/v79/v87's 46-51 because
// v61's switch lacks the extra "case 15" that v72/v79 have between the
// Megaphone family (cases 12,13) and the Item Megaphone case (case 14 is
// identical in both, but everything past it shifts by 1 in v61: avatar is
// case 41 not 42, TV is 45-50 not 46-51). Directly traced the encode tail
// @0x835052-0x8350d6 — same structure as v72/v79's case 46 tail:
//
//	call sub_839D03 (bool check) -> neg/sbb/and/add idiom -> byte of 1 or 3
//	                                                          @0x835052-0x835062
//	Encode1(that byte)                    = pad               @0x83506b
//	EncodeStr(receiverName)                                    @0x835082
//	EncodeStr(line[0..4]) x5 (5th call not re-fetched but follows the
//	  identical var_14/var_10/arg_8/arg_0/String2/var_30 pattern verified in
//	  v72/v79)                                                 @0x835099-0x8350d6+
//
// then falls through to the shared rate-check-and-send tail.
// Wire (v61, tvType 0): pad(byte) + receiverName(str) + 5×line(str) +
// updateTime(uint32 trailing) — matches ItemUseMapleTV.Encode(tvType=0,
// updateTimeFirst=false) exactly.
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v61 ida=0x832a5d
func TestItemUseMapleTVBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewItemUseMapleTV(false, 0)
	input.pad = 3
	input.receiverName = "Receiver"
	input.lines = [5]string{"line0", "line1", "line2", "line3", "line4"}
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "03" +
		"0800" + hex.EncodeToString([]byte("Receiver")) +
		"0500" + hex.EncodeToString([]byte("line0")) +
		"0500" + hex.EncodeToString([]byte("line1")) +
		"0500" + hex.EncodeToString([]byte("line2")) +
		"0500" + hex.EncodeToString([]byte("line3")) +
		"0500" + hex.EncodeToString([]byte("line4")) +
		"39300000"
	if got != want {
		t.Errorf("v61 item use maple tv bytes: got %s, want %s", got, want)
	}
}
