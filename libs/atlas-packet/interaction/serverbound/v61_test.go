package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// GMS v61 PlayerInteraction serverbound bodies. The mode byte is
// dispatcher-framed (COutPacket(111)+Encode1(mode)), NOT part of these
// sub-struct bodies; each fixture pins only the post-mode body. All v61
// send-sites were body-verified in GMS_v61.1_U_DEVM.exe (port 13338) and the
// body encode order equals the verified GMS v72 anchor (v61 sits below every
// v72-specific gate: chatHasUpdateTime false, tradeCrcPresent false). Bytes
// therefore equal the committed v72 fixtures.

// packet-audit:verify packet=interaction/serverbound/InteractionOperationChat version=gms_v61 ida=0x5bfce3
func TestOperationChatV61Bytes(t *testing.T) {
	// v61 sub_5BFCE3 @0x5bfd1c: Encode1(6)=mode then EncodeStr(message). No
	// leading updateTime (that is GMS>=87). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationChat{message: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "02006869" {
		t.Errorf("v61 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v61 ida=0x4ef97f
func TestOperationFieldAddToBlackListV61Bytes(t *testing.T) {
	// v61 sub_4EF97F @0x4ef9a5: Encode1(0x1D)=mode then EncodeStr(name). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationFieldAddToBlackList{name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "02006869" {
		t.Errorf("v61 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v61 ida=0x4ef9f9
func TestOperationFieldRemoveFromBlackListV61Bytes(t *testing.T) {
	// v61 sub_4EF9F9 @0x4efa1f: Encode1(0x1E)=mode then EncodeStr(name). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationFieldRemoveFromBlackList{name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "02006869" {
		t.Errorf("v61 bytes: got %s, want 02006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v61 ida=0x4e87e1
func TestOperationInviteV61Bytes(t *testing.T) {
	// v61 sub_4E87E1 @0x4e891f: Encode1(2)=mode then Encode4(targetCharacterId).
	// Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationInvite{targetCharacterId: 0x12345678}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "78563412" {
		t.Errorf("v61 bytes: got %s, want 78563412", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v61 ida=0x5b10fa
func TestOperationMemoryGameFlipCardV61Bytes(t *testing.T) {
	// v61 sub_5B10FA @0x5b111c: Encode1(0x3E)=mode then Encode1(first),Encode1(index).
	// Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameFlipCard{first: true, index: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0102" {
		t.Errorf("v61 bytes: got %s, want 0102", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v61 ida=0x5fc4d7
func TestOperationMemoryGameMoveStoneV61Bytes(t *testing.T) {
	// v61 sub_5FC4D7 @0x5fc4fc: Encode1(0x3A)=mode then EncodeBuffer(point,8),Encode1(color).
	// int64 point + byte color. Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameMoveStone{point: 1, color: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "010000000000000002" {
		t.Errorf("v61 bytes: got %s, want 010000000000000002", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v61 ida=0x5f7ba3
func TestOperationMemoryGameRetreatAnswerV61Bytes(t *testing.T) {
	// v61 sub_5F7BA3 @0x5f7bc9: Encode1(0x2D)=mode then Encode1(YesNo==6). bool. Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameRetreatAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01" {
		t.Errorf("v61 bytes: got %s, want 01", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v61 ida=0x5f7c60
func TestOperationMemoryGameTieAnswerV61Bytes(t *testing.T) {
	// v61 sub_5F7C60 @0x5f7c86: Encode1(0x31)=mode then Encode1(YesNo==6). bool. Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameTieAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "01" {
		t.Errorf("v61 bytes: got %s, want 01", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v61 ida=0x60d81e
func TestOperationMerchantBuyV61Bytes(t *testing.T) {
	// v61 sub_60D81E @0x60db63: shared BuyItem; Encode1(isMerchant?0x20:0x15)=mode
	// then Encode1(index),Encode2(qty); NO itemCRC (tradeCrcPresent false). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantBuy{index: 3, quantity: 25}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "031900" {
		t.Errorf("v61 bytes: got %s, want 031900", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v61 ida=0x60dbe2
func TestOperationMerchantPutItemV61Bytes(t *testing.T) {
	// v61 sub_60DBE2 @0x60de56: shared PutItem; Encode1(isMerchant?0x1F:0x14)=mode
	// then Encode1(invType),Encode2(slot),Encode2(qty),Encode2(set),Encode4(price). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantPutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v61 bytes: got %s, want 0205006400070040420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v61 ida=0x60df59
func TestOperationMerchantRemoveItemV61Bytes(t *testing.T) {
	// v61 sub_60DF59 @0x60e019: shared MoveItemToInventory; Encode1(isMerchant?0x24:0x19)=mode
	// then Encode2(index). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0500" {
		t.Errorf("v61 bytes: got %s, want 0500", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v61 ida=0x60e20a
func TestOperationPersonalStoreAddToBlackListV61Bytes(t *testing.T) {
	// v61 sub_60E20A @0x60e2b2: Encode1(0x1A)=mode then Encode1(slot),EncodeStr(name). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreAddToBlackList{slot: 2, name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0202006869" {
		t.Errorf("v61 bytes: got %s, want 0202006869", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v61 ida=0x60d81e
func TestOperationPersonalStoreBuyV61Bytes(t *testing.T) {
	// v61 sub_60D81E @0x60db63: shared BuyItem; Encode1(isMerchant?0x20:0x15)=mode
	// then Encode1(index),Encode2(qty); NO itemCRC (tradeCrcPresent false). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreBuy{index: 3, quantity: 25}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "031900" {
		t.Errorf("v61 bytes: got %s, want 031900", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v61 ida=0x60dbe2
func TestOperationPersonalStorePutItemV61Bytes(t *testing.T) {
	// v61 sub_60DBE2 @0x60de56: shared PutItem; Encode1(isMerchant?0x1F:0x14)=mode
	// then Encode1(invType),Encode2(slot),Encode2(qty),Encode2(set),Encode4(price). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStorePutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v61 bytes: got %s, want 0205006400070040420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v61 ida=0x60df59
func TestOperationPersonalStoreRemoveItemV61Bytes(t *testing.T) {
	// v61 sub_60DF59 @0x60e019: shared MoveItemToInventory; Encode1(isMerchant?0x24:0x19)=mode
	// then Encode2(index). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "0500" {
		t.Errorf("v61 bytes: got %s, want 0500", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v61 ida=0x60e154
func TestOperationPersonalStoreSetBlackListV61Bytes(t *testing.T) {
	// v61 sub_60E154 @0x60e185: Encode1(0x1C)=mode then Encode2(count),loop EncodeStr(name). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreSetBlackList{entries: []string{"ab"}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "010002006162" {
		t.Errorf("v61 bytes: got %s, want 010002006162", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v61 ida=0x68ca10
func TestOperationTradeAddMesoV61Bytes(t *testing.T) {
	// v61 sub_68CA10 @0x68cb76: Encode1(0xF)=mode then Encode4(amount). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationTradeAddMeso{amount: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "40420f00" {
		t.Errorf("v61 bytes: got %s, want 40420f00", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v61 ida=0x68cbe3
func TestOperationTradeConfirmV61Bytes(t *testing.T) {
	// v61 sub_68CBE3 @0x68cc56: Encode1(0x10)=mode only, no entry list
	// (tradeCrcPresent false for v61). Bodyless, == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "" {
		t.Errorf("v61 bytes: got %s, want (empty)", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v61 ida=0x68c7e6
func TestOperationTradePutItemV61Bytes(t *testing.T) {
	// v61 sub_68C7E6 @0x68c96c: Encode1(0xE)=mode then Encode1(invType),Encode2(slot),
	// Encode2(qty),Encode1(targetSlot). Body == v72.
	l, _ := testlog.NewNullLogger()
	input := OperationTradePutItem{inventoryType: 2, slot: 5, quantity: 100, targetSlot: 3}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "020500640003" {
		t.Errorf("v61 bytes: got %s, want 020500640003", got)
	}
}

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v61 ida=0x68cbe3
func TestOperationTransactionV61Bytes(t *testing.T) {
	// v61: the cash trade-room confirm shares the bodyless base CTradingRoomDlg::Trade
	// path (sub_68CBE3 @0x68cc56) — Encode1(mode) only, no entry list (tradeCrcPresent
	// false). Bodyless, == v72. There is no v61 opcode-111 send-site emitting a cash
	// entry list (all 52 COutPacket(111) send-sites enumerated by byte signature).
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 61, 1))(nil))
	if got != "" {
		t.Errorf("v61 bytes: got %s, want (empty)", got)
	}
}
