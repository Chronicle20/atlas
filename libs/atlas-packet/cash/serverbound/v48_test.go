package serverbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 cash-shop operation byte fixtures. All arms send COutPacket(160)
// (CASHSHOP_OPERATION) followed by a mode byte, which the dispatcher strips
// before the codec body.

// TestIncreaseStorageBytesV48 — CCashShop::OnIncTrunkCount @0x44aad1 encodes
// Encode1(6) @0x44abf4 (mode), Encode1(isPoints) @0x44ac04, Encode1(0) @0x44ac0d.
// No Encode4 between them, so the v48 body after the mode is two bytes. v83
// @0x46c55b and v87 @0x4763e0 read Decode1, Decode1, Decode4, Decode1.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=gms_v48 ida=0x44aad1
func TestIncreaseStorageBytesV48(t *testing.T) {
	in := ShopOperationIncreaseStorage{isPoints: true, currency: 4000, item: false}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x01, 0x00} // isPoints, item — no currency int
	if !bytes.Equal(got, want) {
		t.Errorf("v48 increase storage:\n got % x\nwant % x", got, want)
	}
	// v83 keeps the currency int.
	v83 := in.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(v83, []byte{0x01, 0xA0, 0x0F, 0x00, 0x00, 0x00}) {
		t.Errorf("v83 increase storage: got % x", v83)
	}
}

// TestMoveFromCashInventoryBytesV48 — CCashShop::OnMoveCashItemLtoS @0x44ec2c
// encodes Encode1(0x0A) @0x44edb7 (mode), EncodeBuffer(sn, 8) @0x44edc5,
// Encode1(inventoryType) @0x44edd0, Encode2(slot) @0x44eddb. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationMoveFromCashInventory version=gms_v48 ida=0x44ec2c
func TestMoveFromCashInventoryBytesV48(t *testing.T) {
	in := ShopOperationMoveFromCashInventory{serialNumber: 0x0102030405060708, inventoryType: 2, slot: 3}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // sn — EncodeBuffer(8)
		0x02,       // inventoryType — Encode1
		0x03, 0x00, // slot          — Encode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 move from cash inventory:\n got % x\nwant % x", got, want)
	}
}

// TestMoveToCashInventoryBytesV48 — CCashShop::OnMoveCashItemStoL @0x44ee3a
// encodes Encode1(0x0B) @0x44eeb6 (mode), EncodeBuffer(sn, 8) @0x44eec4,
// Encode1(inventoryType) @0x44eecf. No slot short on this arm. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationMoveToCashInventory version=gms_v48 ida=0x44ee3a
func TestMoveToCashInventoryBytesV48(t *testing.T) {
	in := ShopOperationMoveToCashInventory{serialNumber: 0x0102030405060708, inventoryType: 2}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x02}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 move to cash inventory:\n got % x\nwant % x", got, want)
	}
}

// TestBuyWorldTransferBytesV48 — CCashShop::SendBuyTransferWorldItemPacket
// @0x44fbfc encodes Encode1(0x24) @0x44fc24 (mode), Encode4(serialNumber)
// @0x44fc2f, Encode4(targetWorld) @0x44fc3a. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=gms_v48 ida=0x44fbfc
func TestBuyWorldTransferBytesV48(t *testing.T) {
	in := ShopOperationBuyWorldTransfer{serialNumber: 0x01020304, targetWorld: 2}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x04, 0x03, 0x02, 0x01, 0x02, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 buy world transfer:\n got % x\nwant % x", got, want)
	}
}

// TestBuyNameChangeBytesV48 — CCashShop::SendBuyNameChangeItemPacket @0x44f9a5
// encodes the mode byte, Encode4(serialNumber), EncodeStr(oldName),
// EncodeStr(newName). Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNameChange version=gms_v48 ida=0x44f9a5
func TestBuyNameChangeBytesV48(t *testing.T) {
	in := ShopOperationBuyNameChange{serialNumber: 0x01020304, oldName: "ab", newName: "cd"}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // serialNumber — Encode4
		0x02, 0x00, 'a', 'b', // oldName      — EncodeStr
		0x02, 0x00, 'c', 'd', // newName      — EncodeStr
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 buy name change:\n got % x\nwant % x", got, want)
	}
}

// TestBuyNormalBytesV48 — CCashShop::OnBuyNormal @0x44cbb2 builds COutPacket(160)
// and encodes Encode1(0x1F) @0x44cdbd (mode), Encode4(spw) @0x44cdc8 - the
// ask_SPW() result - Encode4(serialNumber) @0x44cdd3, EncodeStr(name) @0x44cdec
// and EncodeStr(message) @0x44ce05. That is exactly the buyOmitsCurrency branch
// the codec already takes for GMS below 61; no change needed.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=gms_v48 ida=0x44cbb2
func TestBuyNormalBytesV48(t *testing.T) {
	in := ShopOperationBuyNormal{spw: 0x01020304, serialNumber: 0x0A0B0C0D, name: "ab", message: "cd"}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // spw          — Encode4 @0x44cdc8
		0x0D, 0x0C, 0x0B, 0x0A, // serialNumber — Encode4 @0x44cdd3
		0x02, 0x00, 'a', 'b', // name         — EncodeStr @0x44cdec
		0x02, 0x00, 'c', 'd', // message      — EncodeStr @0x44ce05
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 buy normal:\n got % x\nwant % x", got, want)
	}
}

// TestSetWishlistBytesV48 — CCashShop::OnSetWish @0x44ce9b builds COutPacket(160),
// encodes Encode1(4) @0x44cf86 (mode) and then loops Encode4 @0x44cfb3 exactly ten
// times (`while (v17 < 10)` @0x44cfc3). The codec's decoder reads a fixed ten, so
// the shapes agree; no change needed.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v48 ida=0x44ce9b
func TestSetWishlistBytesV48(t *testing.T) {
	sns := make([]uint32, 10)
	for i := range sns {
		sns[i] = uint32(i + 1)
	}
	got := ShopOperationSetWishlist{serialNumbers: sns}.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	if len(got) != 40 {
		t.Fatalf("v48 set wishlist length = %d, want 40 (ten Encode4)", len(got))
	}
	want := make([]byte, 0, 40)
	for i := 1; i <= 10; i++ {
		want = append(want, byte(i), 0x00, 0x00, 0x00)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 set wishlist:\n got % x\nwant % x", got, want)
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
