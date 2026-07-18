package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// GMS v48 PlayerInteraction serverbound bodies. All send-sites body-verified in
// GMS_v48_1_DEVM.exe (port 13337) from their COutPacket(93) send-site (v48
// PLAYER_INTERACTION serverbound opcode = 93/0x5D; v61 = 111). The leading mode
// byte is dispatcher-framed (COutPacket(93)+Encode1(mode)), NOT part of these
// sub-struct bodies; each fixture pins only the post-mode body. v48 sits below
// every legacy gate (chatHasUpdateTime false, tradeCrcPresent false), so every
// body encode order equals the verified v61 fixtures. The v48 mode table is
// shifted -1 relative to v61 for modes above the ~0xB insertion point (chat 6,
// invite 2 unchanged), but the mode is framed and never enters these bodies.

// packet-audit:verify packet=interaction/serverbound/InteractionOperationChat version=gms_v48 ida=0x546a05
func TestOperationChatV48Bytes(t *testing.T) {
	// v48 sub_546A05 @0x546a30: Encode1(6)=mode then EncodeStr(message). No
	// leading updateTime (GMS>=87). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationChat{message: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "02006869" {
		t.Errorf("v48 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v48 ida=0x4cbd0e
func TestOperationFieldAddToBlackListV48Bytes(t *testing.T) {
	// v48 sub_4CBD0E @0x4cbd34: Encode1(0x1C)=mode then EncodeStr(name). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationFieldAddToBlackList{name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "02006869" {
		t.Errorf("v48 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v48 ida=0x4cbd88
func TestOperationFieldRemoveFromBlackListV48Bytes(t *testing.T) {
	// v48 sub_4CBD88 @0x4cbdae: Encode1(0x1D)=mode then EncodeStr(name). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationFieldRemoveFromBlackList{name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "02006869" {
		t.Errorf("v48 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v48 ida=0x4c5100
func TestOperationInviteV48Bytes(t *testing.T) {
	// v48 sub_4C5100 @0x4c528d: Encode1(2)=mode then Encode4(targetCharacterId).
	// (Same fn first opens the room with mode 0; the invite arm is the mode-2
	// send.) Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationInvite{targetCharacterId: 0x12345678}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "78563412" {
		t.Errorf("v48 bytes: got %s, want 78563412", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v48 ida=0x53875d
func TestOperationMemoryGameFlipCardV48Bytes(t *testing.T) {
	// v48 sub_53875D @0x53877f: Encode1(0x3D)=mode then Encode1(a2),Encode1(a1);
	// caller sub_538613 calls sub_53875D(index, firstFlag) so wire = first,index.
	// Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameFlipCard{first: true, index: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0102" {
		t.Errorf("v48 bytes: got %s, want 0102", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v48 ida=0x578388
func TestOperationMemoryGameMoveStoneV48Bytes(t *testing.T) {
	// v48 sub_578388 @0x5783ad: Encode1(0x39)=mode then EncodeBuffer(point,8),Encode1(color).
	// int64 point + byte color. Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameMoveStone{point: 1, color: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "010000000000000002" {
		t.Errorf("v48 bytes: got %s, want 010000000000000002", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v48 ida=0x573b11
func TestOperationMemoryGameRetreatAnswerV48Bytes(t *testing.T) {
	// v48 sub_573A54 @0x573a7a: Encode1(0x2C)=mode then Encode1(YesNo==6). The
	// retreat REQUEST (server mode 0x2B) routes through sub_5731A9 to this answer
	// send; v48 carries the body bool (unlike the 0x31/0x32 game-control toggle).
	// Answer-with-body pairing == v61 (0x2D). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameRetreatAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "01" {
		t.Errorf("v48 bytes: got %s, want 01", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v48 ida=0x573a54
func TestOperationMemoryGameTieAnswerV48Bytes(t *testing.T) {
	// v48 sub_573B11 @0x573b37: Encode1(0x30)=mode then Encode1(YesNo==6). Tie
	// REQUEST (server mode 0x2F) via sub_5731A9. Body bool == v61 (0x31).
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameTieAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "01" {
		t.Errorf("v48 bytes: got %s, want 01", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v48 ida=0x58847f
func TestOperationMerchantBuyV48Bytes(t *testing.T) {
	// v48 sub_58847F @0x5887d1: shared BuyItem; Encode1(isMerchant?0x1F:0x14)=mode
	// then Encode1(index),Encode2(qty); NO itemCRC (tradeCrcPresent false). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantBuy{index: 3, quantity: 25}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "031900" {
		t.Errorf("v48 bytes: got %s, want 031900", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v48 ida=0x58883f
func TestOperationMerchantPutItemV48Bytes(t *testing.T) {
	// v48 sub_58883F @0x588a51: shared PutItem; Encode1(isMerchant?0x1E:0x13)=mode
	// then Encode1(invType),Encode2(slot),Encode2(qty),Encode2(set),Encode4(price). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantPutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v48 bytes: got %s, want 0205006400070040420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v48 ida=0x588b4b
func TestOperationMerchantRemoveItemV48Bytes(t *testing.T) {
	// v48 sub_588B4B @0x588c18: shared MoveItemToInventory; Encode1(isMerchant?0x23:0x18)=mode
	// then Encode2(index). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0500" {
		t.Errorf("v48 bytes: got %s, want 0500", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v48 ida=0x588dfc
func TestOperationPersonalStoreAddToBlackListV48Bytes(t *testing.T) {
	// v48 sub_588DFC @0x588ea4: Encode1(0x19)=mode then Encode1(slot),EncodeStr(name). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreAddToBlackList{slot: 2, name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0202006869" {
		t.Errorf("v48 bytes: got %s, want 0202006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v48 ida=0x58847f
func TestOperationPersonalStoreBuyV48Bytes(t *testing.T) {
	// v48 sub_58847F @0x5887d1: shared BuyItem; Encode1(isMerchant?0x1F:0x14)=mode
	// then Encode1(index),Encode2(qty); NO itemCRC. Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreBuy{index: 3, quantity: 25}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "031900" {
		t.Errorf("v48 bytes: got %s, want 031900", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v48 ida=0x58883f
func TestOperationPersonalStorePutItemV48Bytes(t *testing.T) {
	// v48 sub_58883F @0x588a51: shared PutItem; Encode1(isMerchant?0x1E:0x13)=mode
	// then Encode1(invType),Encode2(slot),Encode2(qty),Encode2(set),Encode4(price). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStorePutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v48 bytes: got %s, want 0205006400070040420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v48 ida=0x588b4b
func TestOperationPersonalStoreRemoveItemV48Bytes(t *testing.T) {
	// v48 sub_588B4B @0x588c18: shared MoveItemToInventory; Encode1(isMerchant?0x23:0x18)=mode
	// then Encode2(index). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "0500" {
		t.Errorf("v48 bytes: got %s, want 0500", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v48 ida=0x588d46
func TestOperationPersonalStoreSetBlackListV48Bytes(t *testing.T) {
	// v48 sub_588D46 @0x588d77: Encode1(0x1B)=mode then Encode2(count),loop EncodeStr(name). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreSetBlackList{entries: []string{"ab"}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "010002006162" {
		t.Errorf("v48 bytes: got %s, want 010002006162", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v48 ida=0x5e819a
func TestOperationTradeAddMesoV48Bytes(t *testing.T) {
	// v48 sub_5E819A @0x5e830d: Encode1(0xE)=mode then Encode4(amount). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationTradeAddMeso{amount: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "40420f00" {
		t.Errorf("v48 bytes: got %s, want 40420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v48 ida=0x5e836c
func TestOperationTradeConfirmV48Bytes(t *testing.T) {
	// v48 sub_5E836C @0x5e83ec: Encode1(0xF)=mode only, no entry list
	// (tradeCrcPresent false for v48). Bodyless, == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "" {
		t.Errorf("v48 bytes: got %s, want (empty)", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v48 ida=0x5e7f74
func TestOperationTradePutItemV48Bytes(t *testing.T) {
	// v48 sub_5E7F74 @0x5e8109: Encode1(0xD)=mode then Encode1(invType),Encode2(slot),
	// Encode2(qty),Encode1(targetSlot). Body == v61.
	l, _ := testlog.NewNullLogger()
	input := OperationTradePutItem{inventoryType: 2, slot: 5, quantity: 100, targetSlot: 3}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "020500640003" {
		t.Errorf("v48 bytes: got %s, want 020500640003", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v48 ida=0x5e836c
func TestOperationTransactionV48Bytes(t *testing.T) {
	// v48: the cash trade-room confirm shares the bodyless base Trade path
	// (sub_5E836C @0x5e83ec) — Encode1(0xF) mode only, no entry list
	// (tradeCrcPresent false). Bodyless, == v61. No v48 opcode-93 send-site emits
	// a cash entry list (same as v61, which also aliases Transaction to the
	// bodyless Trade send).
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 48, 1))(nil))
	if got != "" {
		t.Errorf("v48 bytes: got %s, want (empty)", got)
	}
}
